package l2_test

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/config"
	"scroll-tech/common/docker"
	"scroll-tech/common/viper"
)

var (
	// docker consider handler.
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance

	// l2geth client
	l2Cli *ethclient.Client
)

func setupEnv(t *testing.T) (err error) {
	// Load config.
	assert.NoError(t, config.NewConfig("../config.json"))

	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	viper.Set("l2_config.relayer_config.sender_config.endpoint", l1gethImg.Endpoint())
	viper.Set("l1_config.endpoint", l1gethImg.Endpoint())

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	viper.Set("l1_config.relayer_config.sender_config.endpoint", l2gethImg.Endpoint())
	viper.Set("l2_config.endpoint", l2gethImg.Endpoint())

	// Create db container.
	driverName := viper.GetViper().GetString("db_config.driver_name")
	dbImg = docker.NewTestDBDocker(t, driverName)

	viper.Set("db_config.driver_name", driverName)
	viper.Set("db_config.dsn", dbImg.Endpoint())

	// Create l2geth client.
	l2Cli, err = ethclient.Dial(viper.GetViper().GetString("l2_config.endpoint"))
	assert.NoError(t, err)

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

	// Run l2 relayer test cases.
	t.Run("TestCreateNewRelayer", testCreateNewRelayer)
	t.Run("TestL2RelayerProcessSaveEvents", testL2RelayerProcessSaveEvents)
	t.Run("testL2RelayerProcessPendingBatches", testL2RelayerProcessPendingBatches)
	t.Run("testL2RelayerProcessCommittedBatches", testL2RelayerProcessCommittedBatches)

	t.Cleanup(func() {
		free(t)
	})
}
