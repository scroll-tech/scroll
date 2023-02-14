package eventwatcher_test

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
	eventwatcher "scroll-tech/bridge/multibin/event_watcher"
	"scroll-tech/common/docker"
	"scroll-tech/database"
)

var ( // config
	cfg *config.Config

	// docker consider handler.
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	l1Cli *ethclient.Client
	l2Cli *ethclient.Client

	ormFactory database.OrmFactory
)

func setEnv(t *testing.T) (err error) {
	cfg, err = config.NewConfig("../../config.json")
	assert.NoError(t, err)
	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = l1gethImg.Endpoint()
	cfg.L1Config.Endpoint = l1gethImg.Endpoint()

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = l2gethImg.Endpoint()
	cfg.L2Config.Endpoint = l2gethImg.Endpoint()

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()
	l1Cli, err = ethclient.Dial(cfg.L1Config.Endpoint)
	assert.NoError(t, err)
	l2Cli, err = ethclient.Dial(cfg.L2Config.Endpoint)
	assert.NoError(t, err)

	ormFactory, err = database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	return err
}

func free(t *testing.T) {
	if ormFactory != nil {
		assert.NoError(t, ormFactory.Close())
	}
	if dbImg != nil {
		assert.NoError(t, dbImg.Stop())
	}
	if l1gethImg != nil {
		assert.NoError(t, l1gethImg.Stop())
	}
	if l2gethImg != nil {
		assert.NoError(t, l2gethImg.Stop())
	}
}

func TestEventWatcher(t *testing.T) {
	if err := setEnv(t); err != nil {
		t.Fatal(err)
	}
	t.Run("TestStartAndStopL1EventWatcher", testStartAndStopL1EventWatcher)
	t.Run("testStartAndStopL2EventWatcher", testStartAndStopL2EventWatcher)

	defer free(t)
}

func testStartAndStopL1EventWatcher(t *testing.T) {
	l1watcher := eventwatcher.NewL1EventWatcher(context.Background(), l1Cli, cfg.L1Config, ormFactory)
	defer l1watcher.Stop()
	// Start all modules.
	l1watcher.Start()
}

func testStartAndStopL2EventWatcher(t *testing.T) {
	l2watcher := eventwatcher.NewL2EventWatcher(context.Background(), l1Cli, cfg.L2Config, ormFactory)
	defer l2watcher.Stop()
	// Start all modules.
	l2watcher.Start()
}
