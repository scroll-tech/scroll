package l2_test

import (
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

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

	vp *viper.Viper
)

func setupEnv(t *testing.T) (err error) {
	// Load config.
	vp, err = viper.NewViper("../config.json")
	assert.NoError(t, err)

	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	vp.Set("l2_config.relayer_config.sender_config.endpoint", l1gethImg.Endpoint())
	vp.Set("l1_config.endpoint", l1gethImg.Endpoint())

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	vp.Set("l1_config.relayer_config.sender_config.endpoint", l2gethImg.Endpoint())
	vp.Set("l2_config.endpoint", l2gethImg.Endpoint())

	// Create db container.
	driverName := vp.Sub("db_config").GetString("driver_name")
	dbImg = docker.NewTestDBDocker(t, driverName)
	vp.Set("db_config.dsn", dbImg.Endpoint())
	vp.Set("db_config.max_open_num", 200)
	vp.Set("db_config.max_idle_num", 20)

	// Create l2geth client.
	l2Cli, err = ethclient.Dial(vp.Sub("l2_config").GetString("endpoint"))
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
	//t.Run("TestMonitorBridgeContract", testMonitorBridgeContract)
	//t.Run("TestFetchMultipleSentMessageInOneBlock", testFetchMultipleSentMessageInOneBlock)

	// Run l2 relayer test cases.
	//t.Run("TestCreateNewRelayer", testCreateNewRelayer)
	//t.Run("TestL2RelayerProcessSaveEvents", testL2RelayerProcessSaveEvents)
	//t.Run("testL2RelayerProcessPendingBatches", testL2RelayerProcessPendingBatches)
	//t.Run("testL2RelayerProcessCommittedBatches", testL2RelayerProcessCommittedBatches)

	t.Cleanup(func() {
		free(t)
	})
}
