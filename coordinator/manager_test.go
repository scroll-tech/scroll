package coordinator_test

import (
	"context"
	"encoding/json"
	mathrand "math/rand"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
	"scroll-tech/common/message"
	"scroll-tech/common/utils"
	db_config "scroll-tech/database"
	"scroll-tech/database/orm"

	"scroll-tech/bridge/mock"

	"scroll-tech/coordinator"
	"scroll-tech/coordinator/config"
)

const managerAddr = "localhost:8132"
const managerPort = ":8132"

var (
	DB_CONFIG = &db_config.DBConfig{
		DriverName: utils.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
		DSN:        utils.GetEnvWithDefault("TEST_DB_DSN", "postgres://postgres:123456@localhost:5436/testmanager_db?sslmode=disable"),
	}

	TEST_CONFIG = &mock.TestConfig{
		L2GethTestConfig: mock.L2GethTestConfig{
			HPort: 8536,
			WPort: 0,
		},
		DbTestconfig: mock.DbTestconfig{
			DbName: "testmanager_db",
			DbPort: 5436,
		},
	}
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance
)

func setupEnv(t *testing.T) {
	// initialize l2geth docker image
	l2gethImg = mock.NewTestL2Docker(t, TEST_CONFIG)
	// initialize db docker image
	dbImg = mock.GetDbDocker(t, TEST_CONFIG)
}

func TestFunction(t *testing.T) {
	// Setup
	setupEnv(t)

	t.Run("TestHandshake", func(t *testing.T) {
		verifierEndpoint := setupMockVerifier(t)

		rollerManager := setupRollerManager(t, verifierEndpoint, nil)
		defer rollerManager.Stop()

		// Set up client
		u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.NoError(t, err)
		defer c.Close()

		mock.PerformHandshake(t, c)

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

		rollerManager := setupRollerManager(t, verifierEndpoint, nil)
		defer rollerManager.Stop()

		// Set up client
		u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.NoError(t, err)
		defer c.Close()

		// Wait for the handshake timeout to pass
		<-time.After(coordinator.HandshakeTimeout + 1*time.Second)

		mock.PerformHandshake(t, c)

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
		rollerManager := setupRollerManager(t, verifierEndpoint, nil)
		defer rollerManager.Stop()

		// Set up and register 2 clients
		for i := 0; i < 2; i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer c.Close()

			mock.PerformHandshake(t, c)

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
		// prepare DB
		mock.ClearDB(t, DB_CONFIG)
		db := mock.PrepareDB(t, DB_CONFIG)

		// Test with two clients to make sure traces messages aren't duplicated
		// to rollers.
		numClients := uint8(2)
		verifierEndpoint := setupMockVerifier(t)
		rollerManager := setupRollerManager(t, verifierEndpoint, db)

		// Set up and register `numClients` clients
		conns := make([]*websocket.Conn, numClients)
		for i := 0; i < int(numClients); i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			defer c.Close()

			mock.PerformHandshake(t, c)

			conns[i] = c
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

			traces := &message.Task{}
			assert.NoError(t, json.Unmarshal(payload, traces))

		}

		rollerManager.Stop()
	})

	t.Run("TestIdleRollerSelection", func(t *testing.T) {
		// Test with two clients to make sure traces messages aren't duplicated
		// to rollers.
		numClients := uint8(2)
		verifierEndpoint := setupMockVerifier(t)

		mock.ClearDB(t, DB_CONFIG)
		db := mock.PrepareDB(t, DB_CONFIG)

		// Ensure only one roller is picked per session.
		rollerManager := setupRollerManager(t, verifierEndpoint, db)
		defer rollerManager.Stop()

		// Set up and register `numClients` clients
		conns := make([]*websocket.Conn, numClients)
		for i := 0; i < int(numClients); i++ {
			u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			assert.NoError(t, err)
			c.SetReadDeadline(time.Now().Add(100 * time.Second))

			mock.PerformHandshake(t, c)
			time.Sleep(1 * time.Second)
			conns[i] = c
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
		// Test first roller and check if we have one roller idle one roller busy
		time.Sleep(4 * time.Second)

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
		time.Sleep(4 * time.Second)

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

func setupMockVerifier(t *testing.T) string {
	id := strconv.Itoa(mathrand.Int())
	verifierEndpoint := "/tmp/" + id + ".sock"
	err := os.RemoveAll(verifierEndpoint)
	assert.NoError(t, err)

	mock.SetupMockVerifier(t, verifierEndpoint)

	return verifierEndpoint
}
