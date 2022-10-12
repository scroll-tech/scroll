package internal_test

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	db_config "scroll-tech/store/config"

	coordinator_config "scroll-tech/coordinator/config"
	"scroll-tech/coordinator/message"

	bridge_config "scroll-tech/bridge/config"

	"scroll-tech/internal/mock"
)

var DB_CONFIG = &db_config.DBConfig{
	DriverName: db_config.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
	DSN:        db_config.GetEnvWithDefault("TEST_DB_DSN", "postgres://postgres:123456@localhost:5440/testInegration_db?sslmode=disable"),
}

var TEST_CONFIG = &mock.TestConfig{
	L2GethTestConfig: mock.L2GethTestConfig{
		HPort: 0,
		WPort: 8568,
	},
	DbTestconfig: mock.DbTestconfig{
		DbName: "testInegration_db",
		DbPort: 5440,
	},
}

func TestL2BackendAndRollerManager(t *testing.T) {
	assert := assert.New(t)
	bridge_cfg, err := bridge_config.NewConfig("./config.json")
	assert.NoError(err)

	coordinator_cfg, err := coordinator_config.NewConfig("./config.json")
	assert.NoError(err)

	l2Backend, img_geth, img_db := mock.Mockl2gethDocker(t, bridge_cfg, TEST_CONFIG)
	bridge_cfg.L2Config.Endpoint = img_geth.Endpoint()
	l2Backend.Start()
	defer l2Backend.Stop()
	defer img_geth.Stop()
	defer img_db.Stop()

	mock.MockClearDB(assert, DB_CONFIG)
	db := mock.MockPrepareDB(assert, DB_CONFIG)

	rollerManager := mock.SetupRollerManager(assert, coordinator_cfg, db)
	defer rollerManager.Stop()

	// connect the mock client
	u := url.URL{Scheme: "ws", Host: "localhost:9000", Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	assert.NoError(err)
	defer c.Close()
	mock.PerformHandshake(assert, c)

	//start to create a transaction to l2geth
	client, err := ethclient.Dial(img_geth.Endpoint())
	assert.NoError(err)

	mock.MockSendTxToL2Client(assert, client)

	<-time.After(10 * time.Second)

	_, payload, err := c.ReadMessage()
	assert.NoError(err)

	msg := &message.Msg{}
	assert.NoError(json.Unmarshal(payload, msg))
	assert.Equal(msg.Type, message.BlockTrace)

	traces := &message.BlockTraces{}
	assert.NoError(json.Unmarshal(payload, traces))

}
