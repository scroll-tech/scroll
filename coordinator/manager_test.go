package coordinator_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
	"scroll-tech/common/message"
	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"

	"scroll-tech/coordinator"
	"scroll-tech/coordinator/config"
	mockRoller "scroll-tech/coordinator/mock"
)

const managerAddr = "localhost:8132"
const managerPort = ":8132"

var (
	dbConfig = &database.DBConfig{
		DriverName: "postgres",
	}
	l2gethImg  docker.ImgInstance
	dbImg      docker.ImgInstance
	roller     mockRoller.MockRoller
	rollers    []mockRoller.MockRoller
	numClients int
)

func setupEnv(t *testing.T) {
	// initialize l2geth docker image
	l2gethImg = docker.NewTestL2Docker(t)
	// initialize db docker image
	dbImg = docker.NewTestDBDocker(t, "postgres")
	dbConfig.DSN = dbImg.Endpoint()
	numClients = 2
	rollers = make([]mockRoller.MockRoller, numClients)
	for i := 0; i < numClients; i++ {
		rollers[i] = mockRoller.MustNewmockRoller()
	}
	roller = rollers[0]
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

		err = roller.PerformHandshake(c)
		assert.NoError(t, err)

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

		err = roller.PerformHandshake(c)
		assert.NoError(t, err)

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

		// Set up and register numClients clients
		for i := 0; i < numClients; i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer c.Close()

			err = rollers[i].PerformHandshake(c)
			assert.NoError(t, err)

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
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		rollerManager := setupRollerManager(t, verifierEndpoint, db)
		defer rollerManager.Stop()

		// Set up and register `numClients` clients
		conns := make([]*websocket.Conn, numClients)
		for i := 0; i < numClients; i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			var conn *websocket.Conn
			conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer conn.Close()

			err = rollers[i].PerformHandshake(conn)
			assert.NoError(t, err)

			conns[i] = conn
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
		for _, c := range conns {
			mt, payload, err := c.ReadMessage()
			assert.NoError(t, err)

			assert.Equal(t, mt, websocket.BinaryMessage)

			msg := &message.Msg{}
			assert.NoError(t, json.Unmarshal(payload, msg))
			assert.Equal(t, msg.Type, message.TaskMsgType)

			traces := &message.TaskMsg{}
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
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		// Ensure only one roller is picked per session.
		rollerManager := setupRollerManager(t, verifierEndpoint, db)
		defer rollerManager.Stop()

		// Set up and register `numClients` clients
		conns := make([]*websocket.Conn, numClients)
		for i := 0; i < numClients; i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			var conn *websocket.Conn
			conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			conn.SetReadDeadline(time.Now().Add(100 * time.Second))

			err = rollers[i].PerformHandshake(conn)
			assert.NoError(t, err)
			time.Sleep(1 * time.Second)
			conns[i] = conn
		}
		defer func() {
			for _, conn := range conns {
				assert.NoError(t, conn.Close())
			}
		}()

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

		for _, c := range conns {
			c.ReadMessage()
		}

		assert.Equal(t, 0, rollerManager.GetNumberOfIdleRollers())
	})

	t.Run("TestGracefulRestart", func(t *testing.T) {
		// Create db handler and reset db.
		db, err := database.NewOrmFactory(dbConfig)
		assert.NoError(t, err)
		assert.NoError(t, migrate.ResetDB(db.GetDB().DB))

		// Test with two clients to make sure traces messages aren't duplicated
		// to rollers.
		verifierEndpoint := setupMockVerifier(t)
		defer os.RemoveAll(verifierEndpoint)

		rollerManager := setupRollerManager(t, verifierEndpoint, db)
		hasStopped := false
		defer func() {
			if !hasStopped {
				defer rollerManager.Stop()
			}
		}()

		// Set up and register `numClients` clients
		conns := make([]*websocket.Conn, numClients)
		for i := 0; i < numClients; i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			var conn *websocket.Conn
			conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer conn.Close()

			err = rollers[i].PerformHandshake(conn)
			assert.NoError(t, err)

			conns[i] = conn
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
		for _, c := range conns {
			var mt int
			var payload []byte
			mt, payload, err = c.ReadMessage()
			assert.NoError(t, err)

			assert.Equal(t, mt, websocket.BinaryMessage)

			msg := &message.Msg{}
			assert.NoError(t, json.Unmarshal(payload, msg))
			assert.Equal(t, msg.Type, message.TaskMsgType)

			traces := &message.TaskMsg{}
			assert.NoError(t, json.Unmarshal(payload, traces))
		}

		// restart coordinator
		rollerManager.Stop()
		hasStopped = true

		rollerManager = setupRollerManager(t, verifierEndpoint, db)
		defer rollerManager.Stop()

		for i := 0; i < numClients; i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			var conn *websocket.Conn
			conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer conn.Close()

			err = rollers[i].PerformHandshake(conn)
			assert.NoError(t, err)

			conns[i] = conn
		}

		for _, c := range conns {
			proofMsg := &message.ProofMsg{
				Status: message.StatusProofError,
				ID:     "0",
				Proof:  &message.AggProof{},
			}
			payload, err := json.Marshal(proofMsg)
			assert.NoError(t, err)
			msg := &message.Msg{
				Type:    message.ProofMsgType,
				Payload: payload,
			}
			msgByt, err := json.Marshal(msg)
			assert.NoError(t, err)
			err = c.WriteMessage(websocket.BinaryMessage, msgByt)
			assert.NoError(t, err)
		}
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
