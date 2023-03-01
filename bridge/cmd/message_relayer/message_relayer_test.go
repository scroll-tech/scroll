package messagerelayer_test

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	messagerelayer "scroll-tech/bridge/cmd/message_relayer"
	"scroll-tech/bridge/config"
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

func TestMsgRelayer(t *testing.T) {
	if err := setEnv(t); err != nil {
		t.Fatal(err)
	}
	t.Run("TestStartAndStopL1MessageRelayer", testStartAndStopL1MessageRelayer)
	t.Run("testStartAndStopL2MessageRelayer", testStartAndStopL2MessageRelayer)

	defer free(t)
}

func testStartAndStopL1MessageRelayer(t *testing.T) {
	l1relayer, err := messagerelayer.NewL1MsgRelayer(context.Background(), cfg.L1Config.Confirmations.Int64(), ormFactory, cfg.L1Config.RelayerConfig)
	assert.NoError(t, err)
	defer l1relayer.Stop()
	// Start all modules.
	l1relayer.Start()
}

func testStartAndStopL2MessageRelayer(t *testing.T) {
	l2relayer, err := messagerelayer.NewL2MsgRelayer(context.Background(), ormFactory, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer l2relayer.Stop()
	// Start all modules.
	l2relayer.Start()
}
