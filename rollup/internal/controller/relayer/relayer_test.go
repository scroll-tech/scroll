package relayer

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
)

var (
	// config
	cfg *config.Config

	base *docker.App

	// l2geth client
	l2Cli *ethclient.Client

	// l2 block
	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock

	// chunk
	chunk1     *types.Chunk
	chunk2     *types.Chunk
	chunkHash1 common.Hash
	chunkHash2 common.Hash
)

func setupEnv(t *testing.T) {
	// Load config.
	var err error
	cfg, err = config.NewConfig("../../../conf/config.json")
	assert.NoError(t, err)

	base.RunImages(t)

	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	cfg.DBConfig = &database.Config{
		DSN:        base.DBConfig.DSN,
		DriverName: base.DBConfig.DriverName,
		MaxOpenNum: base.DBConfig.MaxOpenNum,
		MaxIdleNum: base.DBConfig.MaxIdleNum,
	}
	port, err := rand.Int(rand.Reader, big.NewInt(10000))
	assert.NoError(t, err)
	svrPort := strconv.FormatInt(port.Int64()+50000, 10)
	cfg.L2Config.RelayerConfig.ChainMonitor.BaseURL = "http://localhost:" + svrPort

	// Create l2geth client.
	l2Cli, err = base.L2Client()
	assert.NoError(t, err)

	templateBlockTrace1, err := os.ReadFile("../../../testdata/blockTrace_02.json")
	assert.NoError(t, err)
	wrappedBlock1 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace1, wrappedBlock1)
	assert.NoError(t, err)
	chunk1 = &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock1}}
	chunkHash1, err = chunk1.Hash(0)
	assert.NoError(t, err)

	templateBlockTrace2, err := os.ReadFile("../../../testdata/blockTrace_03.json")
	assert.NoError(t, err)
	wrappedBlock2 = &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace2, wrappedBlock2)
	assert.NoError(t, err)
	chunk2 = &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock2}}
	chunkHash2, err = chunk2.Hash(chunk1.NumL1Messages(0))
	assert.NoError(t, err)
}

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	m.Run()

	base.Free()
}

func TestFunctions(t *testing.T) {
	setupEnv(t)
	srv, err := mockChainMonitorServer(cfg.L2Config.RelayerConfig.ChainMonitor.BaseURL)
	assert.NoError(t, err)
	defer srv.Close()

	// Run l1 relayer test cases.
	t.Run("TestCreateNewL1Relayer", testCreateNewL1Relayer)
	t.Run("TestL1RelayerGasOracleConfirm", testL1RelayerGasOracleConfirm)
	t.Run("TestL1RelayerProcessGasPriceOracle", testL1RelayerProcessGasPriceOracle)

	// Run l2 relayer test cases.
	t.Run("TestCreateNewRelayer", testCreateNewRelayer)
	t.Run("TestL2RelayerProcessPendingBatches", testL2RelayerProcessPendingBatches)
	t.Run("TestL2RelayerProcessCommittedBatches", testL2RelayerProcessCommittedBatches)
	t.Run("TestL2RelayerFinalizeTimeoutBatches", testL2RelayerFinalizeTimeoutBatches)
	t.Run("TestL2RelayerCommitConfirm", testL2RelayerCommitConfirm)
	t.Run("TestL2RelayerFinalizeConfirm", testL2RelayerFinalizeConfirm)
	t.Run("TestL2RelayerGasOracleConfirm", testL2RelayerGasOracleConfirm)
	t.Run("TestLayer2RelayerProcessGasPriceOracle", testLayer2RelayerProcessGasPriceOracle)
	// test getBatchStatusByIndex
	t.Run("TestGetBatchStatusByIndex", testGetBatchStatusByIndex)
}
