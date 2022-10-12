package rollers_test

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

	bridge_config "scroll-tech/config"
	"scroll-tech/store"
	db_config "scroll-tech/store/config"
	"scroll-tech/store/orm"

	"scroll-tech/bridge"

	"scroll-tech/internal/mock"

	rollers "scroll-tech/coordinator"
	coordinator_config "scroll-tech/coordinator/config"
	"scroll-tech/coordinator/message"
)

const managerAddr = "localhost:8132"
const managerPort = ":8132"

var DB_CONFIG = &db_config.DBConfig{
	DriverName: db_config.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
	DSN:        db_config.GetEnvWithDefault("TEST_DB_DSN", "postgres://postgres:123456@localhost:5436/testmanager_db?sslmode=disable"),
}

var TEST_CONFIG = &mock.TestConfig{
	L2GethTestConfig: mock.L2GethTestConfig{
		HPort: 8536,
		WPort: 0,
	},
	DbTestconfig: mock.DbTestconfig{
		DbName: "testmanager_db",
		DbPort: 5436,
	},
}

func TestHandshake(t *testing.T) {
	assert := assert.New(t)
	cfg, err := bridge_config.NewConfig("../config.json")
	assert.NoError(err)

	verifierEndpoint := setupMockVerifier(assert)

	// Set up mock l2 geth
	mockL2BackendClient, imgGeth, imgDb := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer imgGeth.Stop()
	defer imgDb.Stop()

	rollerManager := setupRollerManager(assert, 1, mockL2BackendClient, verifierEndpoint, nil)
	defer rollerManager.Stop()

	// Set up client
	u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)
	defer c.Close()

	mock.PerformHandshake(assert, c)

	// Roller manager should send a Websocket over the WsChan
	select {
	case <-rollerManager.GetRollerChan():
		// Test succeeded
	case <-time.After(rollers.HandshakeTimeout):
		t.Fail()
	}
}

func TestHandshakeTimeout(t *testing.T) {
	assert := assert.New(t)
	cfg, err := bridge_config.config.NewConfig("../config.json")
	assert.NoError(err)

	verifierEndpoint := setupMockVerifier(assert)

	// Set up mock l2 geth
	mockL2BackendClient, imgGeth, imgDb := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer imgGeth.Stop()
	defer imgDb.Stop()

	rollerManager := setupRollerManager(assert, 1, mockL2BackendClient, verifierEndpoint, nil)
	defer rollerManager.Stop()

	// Set up client
	u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)
	defer c.Close()

	// Wait for the handshake timeout to pass
	<-time.After(rollers.HandshakeTimeout + 1*time.Second)

	mock.PerformHandshake(assert, c)

	// No websocket should be received
	select {
	case <-rollerManager.GetRollerChan():
		t.Fail()
	case <-time.After(1 * time.Second):
		// Test succeeded
	}
}

func TestTwoConnections(t *testing.T) {
	assert := assert.New(t)
	cfg, err := bridge_config.NewConfig("../config.json")
	assert.NoError(err)

	verifierEndpoint := setupMockVerifier(assert)

	// Set up mock l2 geth
	mockL2BackendClient, imgGeth, imgDb := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	defer imgGeth.Stop()
	defer imgDb.Stop()

	rollerManager := setupRollerManager(assert, 1, mockL2BackendClient, verifierEndpoint, nil)
	defer rollerManager.Stop()

	// Set up and register 2 clients
	for i := 0; i < 2; i++ {
		u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.NoError(err)
		defer c.Close()

		mock.PerformHandshake(assert, c)

		// Roller manager should send a Websocket over the WsChan
		select {
		case <-rollerManager.GetRollerChan():
			// Test succeeded
		case <-time.After(rollers.HandshakeTimeout):
			t.Fail()
		}
	}
}

func TestTriggerProofGenerationSession(t *testing.T) {
	assert := assert.New(t)
	cfg, err := bridge_config.NewConfig("../config.json")
	assert.NoError(err)

	l2, l2geth_img, db_imge := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	l2.Start()
	defer l2.Stop()
	defer l2geth_img.Stop()
	defer db_imge.Stop()

	// prepare DB
	mock.MockClearDB(assert, DB_CONFIG)
	db := mock.MockPrepareDB(assert, DB_CONFIG)

	// Test with two clients to make sure traces messages aren't duplicated
	// to rollers.
	numClients := uint8(2)
	verifierEndpoint := setupMockVerifier(assert)
	rollerManager := setupRollerManager(assert, 1, l2, verifierEndpoint, db)

	// Set up and register `numClients` clients
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < int(numClients); i++ {
		u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.NoError(err)
		defer c.Close()

		mock.PerformHandshake(assert, c)

		conns[i] = c
	}

	var results []*types.BlockResult

	templateBlockResult, err := os.ReadFile("../../internal/testdata/blockResult_orm.json")
	assert.NoError(err)
	blockResult := &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(err)
	results = append(results, blockResult)
	templateBlockResult, err = os.ReadFile("../../internal/testdata/blockResult_delegate.json")
	assert.NoError(err)
	blockResult = &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(err)
	results = append(results, blockResult)

	err = db.InsertBlockResultsWithStatus(context.Background(), results, orm.BlockUnassigned)
	assert.NoError(err)

	// Need to send a tx to trigger block committed
	// Sleep for a little bit, so that we can avoid prematurely fetching connections. Trigger for manager is 3 seconds
	time.Sleep(4 * time.Second)

	// Both rollers should now receive a `BlockTraces` message and should send something back.
	for _, c := range conns {
		mt, payload, err := c.ReadMessage()
		assert.NoError(err)

		assert.Equal(mt, websocket.BinaryMessage)

		msg := &message.Msg{}
		assert.NoError(json.Unmarshal(payload, msg))
		assert.Equal(msg.Type, message.BlockTrace)

		traces := &message.BlockTraces{}
		assert.NoError(json.Unmarshal(payload, traces))

	}

	rollerManager.Stop()
}

func TestIdleRollerSelection(t *testing.T) {
	assert := assert.New(t)
	cfg, err := bridge_config.NewConfig("../config.json")
	assert.NoError(err)

	l2, l2geth_img, db_imge := mock.Mockl2gethDocker(t, cfg, TEST_CONFIG)
	l2.Start()
	defer l2.Stop()
	defer l2geth_img.Stop()
	defer db_imge.Stop()
	// Test with two clients to make sure traces messages aren't duplicated
	// to rollers.
	numClients := uint8(2)
	verifierEndpoint := setupMockVerifier(assert)

	mock.MockClearDB(assert, DB_CONFIG)
	db := mock.MockPrepareDB(assert, DB_CONFIG)

	// Ensure only one roller is picked per session.
	rollerManager := setupRollerManager(assert, 1, l2, verifierEndpoint, db)
	defer rollerManager.Stop()

	// Set up and register `numClients` clients
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < int(numClients); i++ {
		u := url.URL{Scheme: "ws", Host: managerAddr, Path: "/"}

		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		assert.NoError(err)
		c.SetReadDeadline(time.Now().Add(100 * time.Second))

		mock.PerformHandshake(assert, c)
		time.Sleep(1 * time.Second)
		conns[i] = c
	}
	defer func() {
		for _, conn := range conns {
			assert.NoError(conn.Close())
		}
	}()

	assert.Equal(2, rollerManager.GetNumberOfIdleRollers())

	templateBlockResult, err := os.ReadFile("../../internal/testdata/blockResult_orm.json")
	assert.NoError(err)
	blockResult := &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(err)
	err = db.InsertBlockResultsWithStatus(context.Background(), []*types.BlockResult{blockResult}, orm.BlockUnassigned)
	assert.NoError(err)

	// Sleep for a little bit, so that we can avoid prematurely fetching connections.
	// Test first roller and check if we have one roller idle one roller busy
	time.Sleep(4 * time.Second)

	assert.Equal(1, rollerManager.GetNumberOfIdleRollers())

	templateBlockResult, err = os.ReadFile("../../internal/testdata/blockResult_delegate.json")
	assert.NoError(err)
	blockResult = &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(err)
	err = db.InsertBlockResultsWithStatus(context.Background(), []*types.BlockResult{blockResult}, orm.BlockUnassigned)
	assert.NoError(err)

	// Sleep for a little bit, so that we can avoid prematurely fetching connections.
	// Test Second roller and check if we have all rollers busy
	time.Sleep(4 * time.Second)

	for _, c := range conns {
		c.ReadMessage()
	}

	assert.Equal(0, rollerManager.GetNumberOfIdleRollers())
}

func setupRollerManager(assert *assert.Assertions, rollersPerSession uint8, l2 bridge.MockL2BackendClient, verifierEndpoint string, orm store.OrmFactory) *rollers.Manager {
	rollerManager, err := rollers.New(context.Background(), &coordinator_config.RollerManagerConfig{
		Endpoint:          managerPort,
		RollersPerSession: rollersPerSession,
		VerifierEndpoint:  verifierEndpoint,
		CollectionTime:    1,
	}, orm)
	assert.NoError(err)

	assert.NoError(rollerManager.Start())

	return rollerManager
}

func setupMockVerifier(assert *assert.Assertions) string {
	id := strconv.Itoa(mathrand.Int())
	verifierEndpoint := "/tmp/" + id + ".sock"
	err := os.RemoveAll(verifierEndpoint)
	assert.NoError(err)

	mock.SetupMockVerifier(assert, verifierEndpoint)

	return verifierEndpoint
}
