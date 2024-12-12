package watcher

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/database"
	"scroll-tech/common/testcontainers"
	"scroll-tech/database/migrate"

	"scroll-tech/rollup/internal/config"
)

var (
	// config
	cfg *config.Config

	testApps *testcontainers.TestcontainerApps

	// l2geth client
	l2Cli *ethclient.Client

	// block trace
	block1 *encoding.Block
	block2 *encoding.Block
)

func setupEnv(t *testing.T) (err error) {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.LogfmtFormat()))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	// Load config.
	cfg, err = config.NewConfig("../../../conf/config.json")
	assert.NoError(t, err)

	testApps = testcontainers.NewTestcontainerApps()
	assert.NoError(t, testApps.StartPostgresContainer())
	assert.NoError(t, testApps.StartPoSL1Container())
	assert.NoError(t, testApps.StartL2GethContainer())

	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint, err = testApps.GetPoSL1EndPoint()
	assert.NoError(t, err)
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint, err = testApps.GetL2GethEndPoint()
	assert.NoError(t, err)

	dsn, err := testApps.GetDBEndPoint()
	assert.NoError(t, err)
	cfg.DBConfig = &database.Config{
		DSN:        dsn,
		DriverName: "postgres",
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	}

	// Create l2geth client.
	l2Cli, err = testApps.GetL2GethClient()
	assert.NoError(t, err)

	block1 = readBlockFromJSON(t, "../../../testdata/blockTrace_02.json")
	block2 = readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")

	return err
}

func setupDB(t *testing.T) *gorm.DB {
	db, err := database.InitDB(cfg.DBConfig)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))
	return db
}

func TestMain(m *testing.M) {
	defer func() {
		if testApps != nil {
			testApps.Free()
		}
	}()
	m.Run()
}

func TestFunction(t *testing.T) {
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}

	// Run l1 watcher test cases.
	t.Run("TestL1WatcherClientFetchBlockHeader", testL1WatcherClientFetchBlockHeader)

	// Run l2 watcher test cases.
	t.Run("TestFetchRunningMissingBlocks", testFetchRunningMissingBlocks)

	// Run chunk proposer test cases.
	t.Run("TestChunkProposerLimitsCodecV4", testChunkProposerLimitsCodecV4)
	t.Run("TestChunkProposerBlobSizeLimitCodecV4", testChunkProposerBlobSizeLimitCodecV4)

	// Run batch proposer test cases.
	t.Run("TestBatchProposerLimitsCodecV4", testBatchProposerLimitsCodecV4)
	t.Run("TestBatchCommitGasAndCalldataSizeEstimationCodecV4", testBatchCommitGasAndCalldataSizeEstimationCodecV4)
	t.Run("TestBatchProposerBlobSizeLimitCodecV4", testBatchProposerBlobSizeLimitCodecV4)
	t.Run("TestBatchProposerMaxChunkNumPerBatchLimitCodecV4", testBatchProposerMaxChunkNumPerBatchLimitCodecV4)

	// Run bundle proposer test cases.
	t.Run("TestBundleProposerLimitsCodecV4", testBundleProposerLimitsCodecV4)
}

func readBlockFromJSON(t *testing.T, filename string) *encoding.Block {
	data, err := os.ReadFile(filename)
	assert.NoError(t, err)

	block := &encoding.Block{}
	assert.NoError(t, json.Unmarshal(data, block))
	return block
}
