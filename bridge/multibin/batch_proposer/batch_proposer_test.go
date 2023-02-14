package batchproposer_test

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
	batchproposer "scroll-tech/bridge/multibin/batch_proposer"
	"scroll-tech/common/docker"
	"scroll-tech/database"
)

var ( // config
	cfg *config.Config

	// docker consider handler.
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	ormFactory database.OrmFactory

	l2Cli *ethclient.Client
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

func TestStartAndStopBatchproposer(t *testing.T) {
	setEnv(t)
	batchProposer, err := batchproposer.NewL2BatchProposer(context.Background(), l2Cli, cfg.L2Config, ormFactory)
	assert.NoError(t, err)
	defer func() {
		batchProposer.Stop()
		free(t)
	}()
	// Start all modules.
	batchProposer.Start()
}
