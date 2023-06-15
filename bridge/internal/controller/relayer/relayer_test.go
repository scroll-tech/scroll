package relayer

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
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

	// block trace
	wrappedBlock1 *bridgeTypes.WrappedBlock
	wrappedBlock2 *bridgeTypes.WrappedBlock

	// batch data
	batchData1 *bridgeTypes.BatchData
	batchData2 *bridgeTypes.BatchData
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
	parentBatch1 := &bridgeTypes.BatchInfo{
		Index:     0,
		Hash:      "0x0cc6b102c2924402c14b2e3a19baccc316252bfdc44d9ec62e942d34e39ec729",
		StateRoot: "0x2579122e8f9ec1e862e7d415cef2fb495d7698a8e5f0dddc5651ba4236336e7d",
	}
	batchData1 = bridgeTypes.NewBatchData(parentBatch1, []*bridgeTypes.WrappedBlock{wrappedBlock1}, nil)

	templateBlockTrace2, err := os.ReadFile("../../../testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	wrappedBlock2 = &bridgeTypes.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace2, wrappedBlock2); err != nil {
		return err
	}
	parentBatch2 := &bridgeTypes.BatchInfo{
		Index:     batchData1.Batch.BatchIndex,
		Hash:      batchData1.Hash().Hex(),
		StateRoot: batchData1.Batch.NewStateRoot.String(),
	}
	batchData2 = bridgeTypes.NewBatchData(parentBatch2, []*bridgeTypes.WrappedBlock{wrappedBlock2}, nil)

	log.Info("batchHash", "batchhash1", batchData1.Hash().Hex(), "batchhash2", batchData2.Hash().Hex())

	return err
}

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	m.Run()

	base.Free()
}

func TestFunctions(t *testing.T) {
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}
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
	t.Run("TestL2RelayerMsgConfirm", testL2RelayerMsgConfirm)
	t.Run("TestL2RelayerRollupConfirm", testL2RelayerRollupConfirm)
	t.Run("TestL2RelayerGasOracleConfirm", testL2RelayerGasOracleConfirm)
	t.Run("TestLayer2RelayerProcessGasPriceOracle", testLayer2RelayerProcessGasPriceOracle)
	t.Run("TestLayer2RelayerSendCommitTx", testLayer2RelayerSendCommitTx)
}
