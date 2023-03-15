package l2

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	geth_types "github.com/scroll-tech/go-ethereum/core/types"
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
	blockTrace1 *geth_types.BlockTrace
	blockTrace2 *geth_types.BlockTrace

	// batch data
	batchData1 *types.BatchData
	batchData2 *types.BatchData
)

func setupEnv(t *testing.T) (err error) {
	// Load config.
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)

	base.RunImages(t)

	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1GethEndpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2GethEndpoint()
	cfg.DBConfig.DSN = base.DBEndpoint()

	// Create l2geth client.
	l2Cli, err = base.L2Client()
	assert.NoError(t, err)

	templateBlockTrace1, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	blockTrace1 = &geth_types.BlockTrace{}
	if err = json.Unmarshal(templateBlockTrace1, blockTrace1); err != nil {
		return err
	}

	parentBatch1 := &types.BlockBatch{
		Index: 1,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData1 = types.NewBatchData(parentBatch1, []*geth_types.BlockTrace{blockTrace1}, nil)

	templateBlockTrace2, err := os.ReadFile("../../common/testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	blockTrace2 = &geth_types.BlockTrace{}
	if err = json.Unmarshal(templateBlockTrace2, blockTrace2); err != nil {
		return err
	}
	parentBatch2 := &types.BlockBatch{
		Index: batchData1.Batch.BatchIndex,
		Hash:  batchData1.Hash().Hex(),
	}
	batchData2 = types.NewBatchData(parentBatch2, []*geth_types.BlockTrace{blockTrace2}, nil)

	fmt.Printf("batchhash1 = %x\n", batchData1.Hash())
	fmt.Printf("batchhash2 = %x\n", batchData2.Hash())

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

	// Run l2 watcher test cases.
	t.Run("TestCreateNewWatcherAndStop", testCreateNewWatcherAndStop)
	t.Run("TestMonitorBridgeContract", testMonitorBridgeContract)
	t.Run("TestFetchMultipleSentMessageInOneBlock", testFetchMultipleSentMessageInOneBlock)

	// Run l2 relayer test cases.
	t.Run("TestCreateNewRelayer", testCreateNewRelayer)
	t.Run("TestL2RelayerProcessSaveEvents", testL2RelayerProcessSaveEvents)
	t.Run("TestL2RelayerProcessCommittedBatches", testL2RelayerProcessCommittedBatches)
	t.Run("TestL2RelayerSkipBatches", testL2RelayerSkipBatches)

	// Run batch proposer test cases.
	t.Run("TestBatchProposerProposeBatch", testBatchProposerProposeBatch)
	t.Run("TestBatchProposerGracefulRestart", testBatchProposerGracefulRestart)
	t.Run("TestProposeBatchWithMessages", testProposeBatchWithMessages)
	t.Run("TestInitializeMissingMessageProof", testInitializeMissingMessageProof)

}
