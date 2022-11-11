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

	dbImg := docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	defer dbImg.Stop()
	cfg.DBConfig.DSN = dbImg.Endpoint()

	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2docker := docker.NewTestL2Docker(t)
	defer l2docker.Stop()

	l2Client, err := ethclient.Dial(l2docker.Endpoint())
	assert.NoError(t, err)

	relayer, err := l1.NewLayer1Relayer(context.Background(), l2Client, cfg.L2Config.RelayerConfig, db)
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}
