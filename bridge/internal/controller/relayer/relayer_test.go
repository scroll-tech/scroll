package relayer

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/bridge/internal/config"
	bridgeTypes "scroll-tech/bridge/internal/types"
)

var (
	// config
	cfg *config.Config

	base *docker.App

	// l2geth client
	l2Cli *ethclient.Client

	// l2 block
	wrappedBlock1 *bridgeTypes.WrappedBlock
	wrappedBlock2 *bridgeTypes.WrappedBlock

	// chunk
	chunk1     *bridgeTypes.Chunk
	chunk2     *bridgeTypes.Chunk
	chunkHash1 string
	chunkHash2 string
)

func setupEnv(t *testing.T) {
	// Load config.
	var err error
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
	assert.NoError(t, err)
	wrappedBlock1 = &bridgeTypes.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace1, wrappedBlock1)
	assert.NoError(t, err)
	chunk1 = &bridgeTypes.Chunk{Blocks: []*bridgeTypes.WrappedBlock{wrappedBlock1}}
	chunkHashBytes1, err := chunk1.Hash(0)
	assert.NoError(t, err)
	chunkHash1 = hex.EncodeToString(chunkHashBytes1)

	templateBlockTrace2, err := os.ReadFile("../../../testdata/blockTrace_03.json")
	assert.NoError(t, err)
	wrappedBlock2 = &bridgeTypes.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace2, wrappedBlock2)
	assert.NoError(t, err)
	chunk2 = &bridgeTypes.Chunk{Blocks: []*bridgeTypes.WrappedBlock{wrappedBlock2}}
	chunkHashBytes2, err := chunk2.Hash(chunk1.NumL1Messages(0))
	assert.NoError(t, err)
	chunkHash2 = hex.EncodeToString(chunkHashBytes2)
}

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	m.Run()

	base.Free()
}

func TestFunctions(t *testing.T) {
	setupEnv(t)
	// Run l1 relayer test cases.
	t.Run("TestCreateNewL1Relayer", testCreateNewL1Relayer)
	t.Run("TestL1RelayerProcessSaveEvents", testL1RelayerProcessSaveEvents)
	t.Run("TestL1RelayerMsgConfirm", testL1RelayerMsgConfirm)
	t.Run("TestL1RelayerGasOracleConfirm", testL1RelayerGasOracleConfirm)
	t.Run("TestL1RelayerProcessGasPriceOracle", testL1RelayerProcessGasPriceOracle)

	// Run l2 relayer test cases.
	t.Run("TestCreateNewRelayer", testCreateNewRelayer)
	t.Run("TestL2RelayerProcessCommittedBatches", testL2RelayerProcessCommittedBatches)
	t.Run("TestL2RelayerSkipBatches", testL2RelayerSkipBatches)
	t.Run("TestL2RelayerRollupConfirm", testL2RelayerRollupConfirm)
	t.Run("TestL2RelayerGasOracleConfirm", testL2RelayerGasOracleConfirm)
	t.Run("TestLayer2RelayerProcessGasPriceOracle", testLayer2RelayerProcessGasPriceOracle)
	// TODO(colinlyguo): add ProcessPendingBatches test.
}
