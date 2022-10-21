package l1_test

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database"

	"scroll-tech/bridge/mock"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"

	"scroll-tech/common/utils"
)

var TEST_CONFIG = &mock.TestConfig{
	L1GethTestConfig: mock.L1GethTestConfig{
		HPort: 0,
		WPort: 8570,
	},
	DbTestconfig: mock.DbTestconfig{
		DbName: "testl1relayer_db",
		DbPort: 5440,
		DB_CONFIG: &database.DBConfig{
			DriverName: utils.GetEnvWithDefault("TEST_DB_DRIVER", "postgres"),
			DSN:        utils.GetEnvWithDefault("TEST_DB_DSN", "postgres://postgres:123456@localhost:5440/testl1relayer_db?sslmode=disable"),
		},
	},
}

// TestCreateNewRelayer test create new relayer instance and stop
func TestCreateNewL1Relayer(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)
	l1docker := mock.NewL1Docker(t, TEST_CONFIG)
	defer l1docker.Stop()

	client, err := ethclient.Dial(l1docker.Endpoint())
	assert.NoError(t, err)

	dbImg := mock.NewDBDocker(t, TEST_CONFIG)
	dbImg.Start()
	defer dbImg.Stop()
	db, err := database.NewOrmFactory(TEST_CONFIG.DB_CONFIG)
	assert.NoError(t, err)

	relayer, err := l1.NewLayer1Relayer(context.Background(), client, 1, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)

	relayer.Start()

	defer relayer.Stop()

}
