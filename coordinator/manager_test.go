package coordinator_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/coordinator"
	"scroll-tech/coordinator/config"

	"scroll-tech/common/docker"
	"scroll-tech/common/message"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
)

const managerAddr = "localhost:8132"
const managerPort = ":8132"

var (
	dbConfig = &database.DBConfig{
		DriverName: "postgres",
	}
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance
)

func setupEnv(t *testing.T) {
	// initialize l2geth docker image
	l2gethImg = docker.NewTestL2Docker(t)
	// initialize db docker image
	dbImg = docker.NewTestDBDocker(t, "postgres")
	dbConfig.DSN = dbImg.Endpoint()
}

func TestFunction(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("TestHandshake", func(t *testing.T) {
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		rollerManager := setupRollerManager(t, verifierEndpoint, nil)
		defer rollerManager.Stop()

		// Set up client
		u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.NoError(t, err)
		defer c.Close()

		performHandshake(t, c)

		// Roller manager should send a Websocket over the GetRollerChan
		select {
		case <-rollerManager.GetRollerChan():
			// Test succeeded
		case <-time.After(coordinator.HandshakeTimeout):
			t.Fail()
		}
	})

	t.Run("TestHandshakeTimeout", func(t *testing.T) {
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		rollerManager := setupRollerManager(t, verifierEndpoint, nil)
		defer rollerManager.Stop()

		// Set up client
		u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.NoError(t, err)
		defer c.Close()

		// Wait for the handshake timeout to pass
		<-time.After(coordinator.HandshakeTimeout + 1*time.Second)

		performHandshake(t, c)

		// No websocket should be received
		select {
		case <-rollerManager.GetRollerChan():
			t.Fail()
		case <-time.After(1 * time.Second):
			// Test succeeded
		}
	})

	t.Run("TestTwoConnections", func(t *testing.T) {
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		rollerManager := setupRollerManager(t, verifierEndpoint, nil)
		defer rollerManager.Stop()

		// Set up and register 2 clients
		for i := 0; i < 2; i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer c.Close()

			performHandshake(t, c)

			// Roller manager should send a Websocket over the GetRollerChan
			select {
			case <-rollerManager.GetRollerChan():
				// Test succeeded
			case <-time.After(coordinator.HandshakeTimeout):
				t.Fail()
			}
		}
	})

	t.Run("TestTriggerProofGenerationSession", func(t *testing.T) {
		// Create db handler and reset db.
		db, err := database.NewOrmFactory(dbConfig)
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

		// Test with two clients to make sure traces messages aren't duplicated
		// to rollers.
		numClients := uint8(2)
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		rollerManager := setupRollerManager(t, verifierEndpoint, db)
		defer rollerManager.Stop()

		// Set up and register `numClients` clients
		conns := make([]*websocket.Conn, numClients)
		for i := 0; i < int(numClients); i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			var conn *websocket.Conn
			conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer conn.Close()

			performHandshake(t, conn)

			conns[i] = conn
		}

		var results []*types.BlockResult

		templateBlockResult, err := os.ReadFile("../common/testdata/blockResult_orm.json")
		assert.NoError(t, err)
		blockResult := &types.BlockResult{}
		err = json.Unmarshal(templateBlockResult, blockResult)
		assert.NoError(t, err)
		results = append(results, blockResult)
		templateBlockResult, err = os.ReadFile("../common/testdata/blockResult_delegate.json")
		assert.NoError(t, err)
		blockResult = &types.BlockResult{}
		err = json.Unmarshal(templateBlockResult, blockResult)
		assert.NoError(t, err)
		results = append(results, blockResult)

		err = db.InsertBlockResultsWithStatus(context.Background(), results, orm.BlockUnassigned)
		assert.NoError(t, err)

		// Both rollers should now receive a `BlockTraces` message and should send something back.
		for _, c := range conns {
			mt, payload, err := c.ReadMessage()
			assert.NoError(t, err)

			assert.Equal(t, mt, websocket.BinaryMessage)

			msg := &message.Msg{}
			assert.NoError(t, json.Unmarshal(payload, msg))
			assert.Equal(t, msg.Type, message.BlockTrace)

			traces := &message.BlockTraces{}
			assert.NoError(t, json.Unmarshal(payload, traces))

		}
	})

	t.Run("TestIdleRollerSelection", func(t *testing.T) {
		// Create db handler and reset db.
		db, err := database.NewOrmFactory(dbConfig)
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

		// Test with two clients to make sure traces messages aren't duplicated
		// to rollers.
		numClients := uint8(2)
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		// Ensure only one roller is picked per session.
		rollerManager := setupRollerManager(t, verifierEndpoint, db)
		defer rollerManager.Stop()

		// Set up and register `numClients` clients
		conns := make([]*websocket.Conn, numClients)
		for i := 0; i < int(numClients); i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			var conn *websocket.Conn
			conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			conn.SetReadDeadline(time.Now().Add(100 * time.Second))

			performHandshake(t, conn)
			time.Sleep(1 * time.Second)
			conns[i] = conn
		}
		defer func() {
			for _, conn := range conns {
				assert.NoError(t, conn.Close())
			}
		}()

		assert.Equal(t, 2, rollerManager.GetNumberOfIdleRollers())

		templateBlockResult, err := os.ReadFile("../common/testdata/blockResult_orm.json")
		assert.NoError(t, err)
		blockResult := &types.BlockResult{}
		err = json.Unmarshal(templateBlockResult, blockResult)
		assert.NoError(t, err)
		err = db.InsertBlockResultsWithStatus(context.Background(), []*types.BlockResult{blockResult}, orm.BlockUnassigned)
		assert.NoError(t, err)

		// Sleep for a little bit, so that we can avoid prematurely fetching connections.
		// Test Second roller and check if we have all rollers busy
		time.Sleep(3 * time.Second)

		assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())

		templateBlockResult, err = os.ReadFile("../common/testdata/blockResult_delegate.json")
		assert.NoError(t, err)
		blockResult = &types.BlockResult{}
		err = json.Unmarshal(templateBlockResult, blockResult)
		assert.NoError(t, err)
		err = db.InsertBlockResultsWithStatus(context.Background(), []*types.BlockResult{blockResult}, orm.BlockUnassigned)
		assert.NoError(t, err)

		// Sleep for a little bit, so that we can avoid prematurely fetching connections.
		// Test Second roller and check if we have all rollers busy
		time.Sleep(3 * time.Second)

		for _, c := range conns {
			c.ReadMessage()
		}

		assert.Equal(t, 0, rollerManager.GetNumberOfIdleRollers())
	})
	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l2gethImg.Stop())
		assert.NoError(t, dbImg.Stop())
	})
}

func setupRollerManager(t *testing.T, verifierEndpoint string, orm orm.BlockResultOrm) *coordinator.Manager {
	rollerManager, err := coordinator.New(context.Background(), &config.RollerManagerConfig{
		Endpoint:          managerPort,
		RollersPerSession: 1,
		VerifierEndpoint:  verifierEndpoint,
		CollectionTime:    1,
	}, orm)
	assert.NoError(t, err)

	assert.NoError(t, rollerManager.Start())

	return rollerManager
}

// performHandshake sets up a websocket client to connect to the roller manager.
func performHandshake(t *testing.T, c *websocket.Conn) {
	// Try to perform handshake
	pk, sk := generateKeyPair()
	authMsg := &message.AuthMessage{
		Identity: message.Identity{
			Name:      "testRoller",
			Timestamp: time.Now().UnixNano(),
			PublicKey: common.Bytes2Hex(pk),
		},
		Signature: "",
	}

	hash, err := authMsg.Identity.Hash()
	assert.NoError(t, err)
	sig, err := secp256k1.Sign(hash, sk)
	assert.NoError(t, err)

	authMsg.Signature = common.Bytes2Hex(sig)

	b, err := json.Marshal(authMsg)
	assert.NoError(t, err)

	msg := &message.Msg{
		Type:    message.Register,
		Payload: b,
	}

	b, err = json.Marshal(msg)
	assert.NoError(t, err)

	assert.NoError(t, c.WriteMessage(websocket.BinaryMessage, b))
}

func generateKeyPair() (pubkey, privkey []byte) {
	key, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubkey = elliptic.Marshal(secp256k1.S256(), key.X, key.Y)

	privkey = make([]byte, 32)
	blob := key.D.Bytes()
	copy(privkey[32-len(blob):], blob)

	return pubkey, privkey
}

// setupMockVerifier sets up a mocked verifier for a test case.
func setupMockVerifier(t *testing.T) string {
	verifierEndpoint := "/tmp/" + strconv.Itoa(time.Now().Nanosecond()) + ".sock"

	// Create and listen sock file.
	l, err := net.Listen("unix", verifierEndpoint)
	assert.NoError(t, err)

	go func() {
		conn, err := l.Accept()
		assert.NoError(t, err)

		// Simply read all incoming messages and send a true boolean straight back.
		for {
			// Read length
			buf := make([]byte, 4)
			_, err = io.ReadFull(conn, buf)
			assert.NoError(t, err)

			// Read message
			msgLength := binary.LittleEndian.Uint64(buf)
			buf = make([]byte, msgLength)
			_, err = io.ReadFull(conn, buf)
			assert.NoError(t, err)

			// Return boolean
			buf = []byte{1}
			_, err = conn.Write(buf)
			assert.NoError(t, err)
		}
	}()
	return verifierEndpoint
}
