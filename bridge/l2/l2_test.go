package l2

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/trie"
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
	blockWithWithdrawTrieRoot1 *types.BlockWithWithdrawTrieRoot
	blockWithWithdrawTrieRoot2 *types.BlockWithWithdrawTrieRoot

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
	blockTrace1 := &geth_types.BlockTrace{}
	if err = json.Unmarshal(templateBlockTrace1, blockTrace1); err != nil {
		return err
	}

	parentBatch1 := &types.BlockBatch{
		Index: 1,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	transactions1 := make(geth_types.Transactions, len(blockTrace1.Transactions))
	for i, txData := range blockTrace1.Transactions {
		data, _ := hexutil.Decode(txData.Data)
		transactions1[i] = geth_types.NewTx(&geth_types.LegacyTx{
			Nonce:    txData.Nonce,
			To:       txData.To,
			Value:    txData.Value.ToInt(),
			Gas:      txData.Gas,
			GasPrice: txData.GasPrice.ToInt(),
			Data:     data,
			V:        txData.V.ToInt(),
			R:        txData.R.ToInt(),
			S:        txData.S.ToInt(),
		})
	}
	block1 := geth_types.NewBlock(blockTrace1.Header, transactions1, nil, nil, trie.NewStackTrie(nil))
	blockWithWithdrawTrieRoot1 = &types.BlockWithWithdrawTrieRoot{
		Block:            block1,
		WithdrawTrieRoot: common.Hash{},
	}
	batchData1 = types.NewBatchData(parentBatch1, []*types.BlockWithWithdrawTrieRoot{blockWithWithdrawTrieRoot1}, nil)

	templateBlockTrace2, err := os.ReadFile("../../common/testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	blockTrace2 := &geth_types.BlockTrace{}
	if err = json.Unmarshal(templateBlockTrace2, blockTrace2); err != nil {
		return err
	}
	parentBatch2 := &types.BlockBatch{
		Index: batchData1.Batch.BatchIndex,
		Hash:  batchData1.Hash().Hex(),
	}
	transactions2 := make(geth_types.Transactions, len(blockTrace2.Transactions))
	for i, txData := range blockTrace2.Transactions {
		data, _ := hexutil.Decode(txData.Data)
		transactions2[i] = geth_types.NewTx(&geth_types.LegacyTx{
			Nonce:    txData.Nonce,
			To:       txData.To,
			Value:    txData.Value.ToInt(),
			Gas:      txData.Gas,
			GasPrice: txData.GasPrice.ToInt(),
			Data:     data,
			V:        txData.V.ToInt(),
			R:        txData.R.ToInt(),
			S:        txData.S.ToInt(),
		})
	}
	block2 := geth_types.NewBlock(blockTrace2.Header, transactions2, nil, nil, trie.NewStackTrie(nil))
	blockWithWithdrawTrieRoot2 = &types.BlockWithWithdrawTrieRoot{
		Block:            block2,
		WithdrawTrieRoot: common.Hash{},
	}
	batchData2 = types.NewBatchData(parentBatch2, []*types.BlockWithWithdrawTrieRoot{blockWithWithdrawTrieRoot2}, nil)

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

}
