package watcher

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
	"scroll-tech/common/types"

	"scroll-tech/bridge/config"
)

var (
	// config
	cfg *config.Config

	base *docker.App

	// l2geth client
	l2Cli *ethclient.Client

	// block trace
	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock
)

func setupEnv(t *testing.T) (err error) {
	// Load config.
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)

	base.RunImages(t)

	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	cfg.DBConfig = base.DBConfig

	// Create l2geth client.
	l2Cli, err = base.L2Client()
	assert.NoError(t, err)

	templateBlockTrace1, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock1 = &types.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace1, wrappedBlock1); err != nil {
		return err
	}

	templateBlockTrace2, err := os.ReadFile("../../common/testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock2 = &types.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace2, wrappedBlock2); err != nil {
		return err
	}
	return err
}

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	m.Run()

	base.Free()
}

func TestFunction(t *testing.T) {
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}

	// Run l1 watcher test cases.
	t.Run("TestStartWatcher", testStartWatcher)
	t.Run("testL1WatcherClientFetchBlockHeader", testL1WatcherClientFetchBlockHeader)
	t.Run("testL1WatcherClientFetchContractEvent", testL1WatcherClientFetchContractEvent)
	t.Run("testParseBridgeEventLogs", testParseBridgeEventLogs)

	// Run l2 watcher test cases.
	t.Run("TestCreateNewWatcherAndStop", testCreateNewWatcherAndStop)
	t.Run("TestMonitorBridgeContract", testMonitorBridgeContract)
	t.Run("TestFetchMultipleSentMessageInOneBlock", testFetchMultipleSentMessageInOneBlock)
	t.Run("TestFetchRunningMissingBlocks", testFetchRunningMissingBlocks)

	// Run batch proposer test cases.
	t.Run("TestBatchProposerProposeBatch", testBatchProposerProposeBatch)
	t.Run("TestBatchProposerBatchGeneration", testBatchProposerBatchGeneration)
	t.Run("TestBatchProposerGracefulRestart", testBatchProposerGracefulRestart)
}
