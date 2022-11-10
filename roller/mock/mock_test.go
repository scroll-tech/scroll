package mock

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	message "scroll-tech/common/message"

	"scroll-tech/roller/config"
	"scroll-tech/roller/core"
)

var (
	cfg        *config.Config
	scrollPort = 9020
	mockPath   string
)

func TestMain(m *testing.M) {
	mockPath = "./roller_mock_test"

	_ = os.RemoveAll(mockPath)
	if err := os.Mkdir(mockPath, os.ModePerm); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	scrollPort = rand.Intn(9000)
	cfg = &config.Config{
		RollerName:       "test-roller",
		KeystorePath:     filepath.Join(mockPath, "roller-keystore"),
		KeystorePassword: "mock_test",
		ScrollURL:        fmt.Sprintf("ws://localhost:%d", scrollPort),
		Prover:           &config.ProverConfig{MockMode: true},
		DBPath:           filepath.Join(mockPath, "stack_db"),
	}

	os.Exit(m.Run())
}

func TestRoller(t *testing.T) {
	go mockScroll(t)

	r, err := core.NewRoller(cfg)
	assert.NoError(t, err)

	go r.Run()

	<-time.NewTimer(2 * time.Second).C

	r.Close()
}

func mockScroll(t *testing.T) {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		up := websocket.Upgrader{}
		c, err := up.Upgrade(w, req, nil)
		assert.NoError(t, err, "Upgrade WS")

		var payload []byte
		payload, err = func(c *websocket.Conn) ([]byte, error) {
			for {
				var mt int
				mt, payload, err = c.ReadMessage()
				if err != nil {
					return nil, err
				}

				if mt == websocket.BinaryMessage {
					return payload, nil
				}
			}
		}(c)
		assert.NoError(t, err, "read payload")

		msg := &message.Msg{}
		err = json.Unmarshal(payload, msg)
		assert.NoError(t, err, "json Unmarshal raw payload")

		authMsg := &message.AuthMessage{}
		err = json.Unmarshal(msg.Payload, authMsg)
		assert.NoError(t, err, "json Unmarshal inner payload")

		// Verify signature
		hash, err := authMsg.Identity.Hash()
		assert.NoError(t, err, "authMsg.Identity.Hash()")

		if !secp256k1.VerifySignature(common.FromHex(authMsg.Identity.PublicKey), hash, common.FromHex(authMsg.Signature)[:64]) {
			assert.NoError(t, err, "VerifySignature")
		}
		t.Log("signature verification successfully. Roller: ", authMsg.Identity.Name)
		assert.Equal(t, cfg.RollerName, authMsg.Identity.Name)

		traces := &message.BlockTraces{
			ID:     16,
			Traces: nil,
		}
		msgByt, err := core.MakeMsgByt(message.BlockTrace, traces)
		assert.NoError(t, err, "MakeMsgByt")

		err = c.WriteMessage(websocket.BinaryMessage, msgByt)
		assert.NoError(t, err, "WriteMessage")
	})
	http.ListenAndServe(fmt.Sprintf(":%d", scrollPort), nil)
}
