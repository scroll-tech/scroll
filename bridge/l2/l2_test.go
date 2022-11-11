package l2_test

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/bridge/config"
)

var (
	// config
	cfg *config.Config

	// l1 chain docker consider handler.
	l1gethImg docker.ImgInstance
	// l2 chain docker consider handler.
	l2gethImg docker.ImgInstance

	// l1geth client
	l1Cli *ethclient.Client
	// l2geth client
	l2Cli *ethclient.Client

	// postgres docker handler.
	dbImg docker.ImgInstance
)

func setupEnv(t *testing.T) (err error) {
	// Load config.
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)

	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	cfg.L1Config.Endpoint = l1gethImg.Endpoint()

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	cfg.L2Config.Endpoint = l2gethImg.Endpoint()

	// Create l2geth client.
	l1Cli, err = ethclient.Dial(cfg.L1Config.Endpoint)
	assert.NoError(t, err)

	// Create l2geth client.
	l2Cli, err = ethclient.Dial(cfg.L2Config.Endpoint)
	assert.NoError(t, err)

	// Create db container.
	dbImg = docker.NewTestDBDocker(t, cfg.DBConfig.DriverName)
	cfg.DBConfig.DSN = dbImg.Endpoint()

	return err
}

func free(t *testing.T) {
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

func TestFunction(t *testing.T) {
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}

	// Run l2 watcher test cases.
	t.Run("TestCreateNewWatcherAndStop", testCreateNewWatcherAndStop)
	t.Run("TestMonitorBridgeContract", testMonitorBridgeContract)
	t.Run("TestFetchMultipleSentMessageInOneBlock", testFetchMultipleSentMessageInOneBlock)
	t.Run("TestTraceHasUnsupportedOpcodes", testTraceHasUnsupportedOpcodes)

	// Run l2 relayer test cases.
	t.Run("TestCreateNewRelayer", testCreateNewRelayer)
	t.Run("TestL2RelayerProcessSaveEvents", testL2RelayerProcessSaveEvents)
	t.Run("TestL2RelayerProcessPendingBlocks", testL2RelayerProcessPendingBlocks)
	t.Run("TestL2RelayerProcessCommittedBlocks", testL2RelayerProcessCommittedBlocks)

	t.Cleanup(func() {
		free(t)
	})
}
