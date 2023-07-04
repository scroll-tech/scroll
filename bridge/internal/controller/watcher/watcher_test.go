package watcher

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/docker"
	cutils "scroll-tech/common/utils"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm/migrate"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
)

var (
	// config
	cfg *config.Config

	base *docker.App

	// l2geth client
	l2Cli *ethclient.Client

	// block trace
	wrappedBlock1 *bridgeTypes.WrappedBlock
	wrappedBlock2 *bridgeTypes.WrappedBlock
)

func setupEnv(t *testing.T) (err error) {
	// Load config.
	cfg, err = config.NewConfig("../../../conf/config.json")
	assert.NoError(t, err)

	base.RunImages(t)

	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	cfg.DBConfig = &config.DBConfig{
		DSN:        base.DBConfig.DSN,
		DriverName: base.DBConfig.DriverName,
		MaxOpenNum: base.DBConfig.MaxOpenNum,
		MaxIdleNum: base.DBConfig.MaxIdleNum,
	}

	// Create l2geth client.
	l2Cli, err = base.L2Client()
	assert.NoError(t, err)

	templateBlockTrace1, err := os.ReadFile("../../../testdata/blockTrace_02.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock1 = &bridgeTypes.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace1, wrappedBlock1); err != nil {
		return err
	}

	templateBlockTrace2, err := os.ReadFile("../../../testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock2 = &bridgeTypes.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace2, wrappedBlock2); err != nil {
		return err
	}
	return err
}

func setupDB(t *testing.T) *gorm.DB {
	logger, err1 := cutils.LogSetup(nil)
	assert.NoError(t, err1)
	db, err := utils.InitDB(cfg.DBConfig, logger)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
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
	t.Run("TestStartWatcher", testFetchContractEvent)
	t.Run("TestL1WatcherClientFetchBlockHeader", testL1WatcherClientFetchBlockHeader)
	t.Run("TestL1WatcherClientFetchContractEvent", testL1WatcherClientFetchContractEvent)
	t.Run("TestParseBridgeEventLogsL1QueueTransactionEventSignature", testParseBridgeEventLogsL1QueueTransactionEventSignature)
	t.Run("TestParseBridgeEventLogsL1CommitBatchEventSignature", testParseBridgeEventLogsL1CommitBatchEventSignature)
	t.Run("TestParseBridgeEventLogsL1FinalizeBatchEventSignature", testParseBridgeEventLogsL1FinalizeBatchEventSignature)

	// Run l2 watcher test cases.
	t.Run("TestCreateNewWatcherAndStop", testCreateNewWatcherAndStop)
	t.Run("TestFetchRunningMissingBlocks", testFetchRunningMissingBlocks)
	t.Run("TestParseBridgeEventLogsL2RelayedMessageEventSignature", testParseBridgeEventLogsL2RelayedMessageEventSignature)
	t.Run("TestParseBridgeEventLogsL2FailedRelayedMessageEventSignature", testParseBridgeEventLogsL2FailedRelayedMessageEventSignature)

	// Run chunk-proposer test cases.
	t.Run("TestChunkProposer", testChunkProposer)

	// Run batch-proposer test cases.
	t.Run("TestBatchProposer", testBatchProposer)
}
