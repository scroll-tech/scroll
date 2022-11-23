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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
	"scroll-tech/common/message"
	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"

	"scroll-tech/coordinator"
	"scroll-tech/coordinator/config"
)

const coordinatorAddr = "localhost:8132"
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

		roller := mustNewMockRoller()
		defer roller.stop()

		assert.NoError(t, roller.dialCoordinator())
		assert.NoError(t, roller.performHandshake())

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

		roller := mustNewMockRoller()
		defer roller.stop()

		assert.NoError(t, roller.dialCoordinator())

		// Wait for the handshake timeout to pass
		<-time.After(coordinator.HandshakeTimeout + 1*time.Second)

		assert.NoError(t, roller.performHandshake())

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

		numClients := 2
		rollers := make([]mockRoller, numClients)
		for i := 0; i < numClients; i++ {
			rollers[i] = mustNewMockRoller()
		}
		defer func() {
			for i := 0; i < numClients; i++ {
				rollers[i].stop()
			}
		}()

		// Set up and register numClients clients
		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].dialCoordinator())
			assert.NoError(t, rollers[i].performHandshake())

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

		rollerManager := setupRollerManager(t, "", db)
		defer rollerManager.Stop()

		numClients := 2
		rollers := make([]mockRoller, numClients)
		for i := 0; i < numClients; i++ {
			rollers[i] = mustNewMockRoller()
		}
		defer func() {
			for i := 0; i < numClients; i++ {
				rollers[i].stop()
			}
		}()
		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].dialCoordinator())
			assert.NoError(t, rollers[i].performHandshake())
		}

		dbTx, err := db.Beginx()
		assert.NoError(t, err)
		_, err = db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(1)}, &orm.BlockInfo{Number: uint64(1)}, "0f", 1, 194676)
		assert.NoError(t, err)
		_, err = db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(2)}, &orm.BlockInfo{Number: uint64(2)}, "0e", 1, 194676)
		assert.NoError(t, err)
		err = dbTx.Commit()
		assert.NoError(t, err)

		// Need to send a tx to trigger block committed
		// Sleep for a little bit, so that we can avoid prematurely fetching connections. Trigger for manager is 3 seconds
		time.Sleep(4 * time.Second)

		// Both rollers should now receive a `BlockTraces` message and should send something back.
		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].readMessage())
			assert.NoError(t, rollers[i].sendProof())
		}
		time.Sleep(3 * time.Second)
	})

	t.Run("TestIdleRollerSelection", func(t *testing.T) {
		// Create db handler and reset db.
		db, err := database.NewOrmFactory(dbConfig)
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

		// Test with two clients to make sure traces messages aren't duplicated
		// to rollers.
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		// Ensure only one roller is picked per session.
		rollerManager := setupRollerManager(t, verifierEndpoint, db)
		defer rollerManager.Stop()

		numClients := 2
		rollers := make([]mockRoller, numClients)
		for i := 0; i < numClients; i++ {
			rollers[i] = mustNewMockRoller()
		}
		defer func() {
			for i := 0; i < numClients; i++ {
				rollers[i].stop()
			}
		}()
		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].dialCoordinator())
			assert.NoError(t, rollers[i].performHandshake())
			time.Sleep(1 * time.Second)
		}

		assert.Equal(t, 2, rollerManager.GetNumberOfIdleRollers())

		dbTx, err := db.Beginx()
		assert.NoError(t, err)
		_, err = db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(1)}, &orm.BlockInfo{Number: uint64(1)}, "0f", 1, 194676)
		assert.NoError(t, err)
		err = dbTx.Commit()
		assert.NoError(t, err)

		// Sleep for a little bit, so that we can avoid prematurely fetching connections.
		// Test first roller and check if we have one roller idle one roller busy
		time.Sleep(4 * time.Second)

		assert.Equal(t, 1, rollerManager.GetNumberOfIdleRollers())

		dbTx, err = db.Beginx()
		assert.NoError(t, err)
		_, err = db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(2)}, &orm.BlockInfo{Number: uint64(2)}, "0e", 1, 194676)
		assert.NoError(t, err)
		err = dbTx.Commit()
		assert.NoError(t, err)

		// Sleep for a little bit, so that we can avoid prematurely fetching connections.
		// Test Second roller and check if we have all rollers busy
		time.Sleep(4 * time.Second)

		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].readMessage())
		}

		assert.Equal(t, 0, rollerManager.GetNumberOfIdleRollers())
	})

	t.Run("TestGracefulRestart", func(t *testing.T) {
		// Create db handler and reset db.
		db, err := database.NewOrmFactory(dbConfig)
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

		rollerManager := setupRollerManager(t, "", db)
		hasStopped := false
		defer func() {
			if !hasStopped {
				defer rollerManager.Stop()
			}
		}()

		numClients := 2
		rollers := make([]mockRoller, numClients)
		for i := 0; i < numClients; i++ {
			rollers[i] = mustNewMockRoller()
		}
		defer func() {
			for i := 0; i < numClients; i++ {
				rollers[i].stop()
			}
		}()
		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].dialCoordinator())
			assert.NoError(t, rollers[i].performHandshake())
		}

		dbTx, err := db.Beginx()
		assert.NoError(t, err)
		_, err = db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(1)}, &orm.BlockInfo{Number: uint64(1)}, "0f", 1, 194676)
		assert.NoError(t, err)
		_, err = db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: uint64(2)}, &orm.BlockInfo{Number: uint64(2)}, "0e", 1, 194676)
		assert.NoError(t, err)
		err = dbTx.Commit()
		assert.NoError(t, err)

		// Need to send a tx to trigger block committed
		// Sleep for a little bit, so that we can avoid prematurely fetching connections. Trigger for manager is 3 seconds
		time.Sleep(4 * time.Second)

		// Both rollers should now receive a `BlockTraces` message and should send something back.
		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].readMessage())
		}

		// restart coordinator
		rollerManager.Stop()
		hasStopped = true

		newRollerManager := setupRollerManager(t, "", db)
		defer newRollerManager.Stop()

		for i := 0; i < numClients; i++ {
			assert.NoError(t, rollers[i].dialCoordinator())
			assert.NoError(t, rollers[i].performHandshake())
			assert.NoError(t, rollers[i].sendProof())
		}

		time.Sleep(4 * time.Second)
	})

	// Teardown
	t.Cleanup(func() {
		assert.NoError(t, l2gethImg.Stop())
		assert.NoError(t, dbImg.Stop())
	})
}

func setupRollerManager(t *testing.T, verifierEndpoint string, orm database.OrmFactory) *coordinator.Manager {
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

type mockRoller struct {
	wsURL      string
	publicKey  []byte
	privateKey []byte
	conn       *websocket.Conn
	sessionID  string // currently only one session
}

func mustGenerateKeyPair() (pubkey, privkey []byte) {
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

func mustNewMockRoller() mockRoller {
	u := url.URL{Scheme: "ws", Host: coordinatorAddr, Path: "/"}
	wsURL := u.String()
	publicKey, privateKey := mustGenerateKeyPair()
	return mockRoller{wsURL: wsURL, publicKey: publicKey, privateKey: privateKey}
}

func (r *mockRoller) dialCoordinator() error {
	var err error
	r.conn, _, err = websocket.DefaultDialer.Dial(r.wsURL, nil)
	return err
}

func (r *mockRoller) performHandshake() error {
	// TODO: deal with unconnected conn
	authMsg := &message.AuthMessage{
		Identity: message.Identity{
			Name:      "testRoller",
			Timestamp: time.Now().UnixNano(),
			PublicKey: common.Bytes2Hex(r.publicKey),
		},
		Signature: "",
	}
	hash, err := authMsg.Identity.Hash()
	if err != nil {
		return err
	}
	sig, err := secp256k1.Sign(hash, r.privateKey)
	if err != nil {
		return err
	}
	authMsg.Signature = common.Bytes2Hex(sig)
	b, err := json.Marshal(authMsg)
	if err != nil {
		return err
	}
	msg := &message.Msg{
		Type:    message.RegisterMsgType,
		Payload: b,
	}
	b, err = json.Marshal(msg)
	if err != nil {
		return err
	}
	err = r.conn.WriteMessage(websocket.BinaryMessage, b)
	return err
}

func (r *mockRoller) readMessage() error {
	mt, payload, err := r.conn.ReadMessage()
	if err != nil {
		return err
	}
	if mt != websocket.BinaryMessage {
		log.Crit("mt != websocket.BinaryMessage", "mt", mt)
	}
	msg := &message.Msg{}
	if err := json.Unmarshal(payload, msg); err != nil {
		return err
	}
	if msg.Type != message.TaskMsgType {
		log.Crit("msg.Type != message.TaskMsgType", "msg.Type", msg.Type)
	}
	task := &message.TaskMsg{}
	if err := json.Unmarshal(msg.Payload, task); err != nil {
		return err
	}
	r.sessionID = task.ID
	return nil
}

func (r *mockRoller) sendProof() error {
	proofMsg := &message.ProofMsg{
		Status: message.StatusOk,
		ID:     r.sessionID,
		Proof:  &message.AggProof{},
	}
	payload, err := json.Marshal(proofMsg)
	if err != nil {
		return err
	}
	msg := &message.Msg{
		Type:    message.ProofMsgType,
		Payload: payload,
	}
	msgByt, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.conn.WriteMessage(websocket.BinaryMessage, msgByt)
}

func (r *mockRoller) stop() {
	r.conn.Close()
}
