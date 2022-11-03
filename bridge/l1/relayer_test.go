package l1_test

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/database/migrate"

	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"

	"scroll-tech/database"

	"scroll-tech/common/docker"
)

// TestCreateNewRelayer test create new relayer instance and stop
func TestCreateNewL1Relayer(t *testing.T) {
	cfg, err := config.NewConfig("../config.json")
	assert.NoError(t, err)
	l1docker := docker.NewTestL1Docker(t)
	defer l1docker.Stop()
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1docker.Endpoint()
	cfg.L1Config.Endpoint = l1docker.Endpoint()

	client, err := ethclient.Dial(l1docker.Endpoint())
	assert.NoError(t, err)

	dbImg := docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	defer dbImg.Stop()
	cfg.DBConfig.DSN = dbImg.Endpoint()

	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := l1.NewLayer1Relayer(context.Background(), client, 1, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}
