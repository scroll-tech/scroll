package l2

import (
	"testing"

	"scroll-tech/common/testdata"

	"scroll-tech/common/docker"
	"scroll-tech/common/types"

	"scroll-tech/bridge/config"
)

var (
	// config
	cfg *config.Config

	base *docker.App

	// block trace
	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock

	// batch data
	batchData1 *types.BatchData
	batchData2 *types.BatchData
)

func init() {
	trace02 := testdata.GetTrace("../../common/testdata/blockTrace_02.json")
	wrappedBlock1 = &types.WrappedBlock{
		Header:           trace02.Header,
		Transactions:     trace02.Transactions,
		WithdrawTrieRoot: trace02.WithdrawTrieRoot,
	}
	batchData1 = types.NewBatchData(&types.BlockBatch{
		Index:     0,
		Hash:      "0x0cc6b102c2924402c14b2e3a19baccc316252bfdc44d9ec62e942d34e39ec729",
		StateRoot: "0x2579122e8f9ec1e862e7d415cef2fb495d7698a8e5f0dddc5651ba4236336e7d",
	}, []*types.WrappedBlock{wrappedBlock1}, nil)

	trace03 := testdata.GetTrace("../../common/testdata/blockTrace_03.json")
	wrappedBlock2 = &types.WrappedBlock{
		Header:           trace03.Header,
		Transactions:     trace03.Transactions,
		WithdrawTrieRoot: trace03.WithdrawTrieRoot,
	}
	batchData2 = types.NewBatchData(&types.BlockBatch{
		Index:     batchData1.Batch.BatchIndex,
		Hash:      batchData1.Hash().Hex(),
		StateRoot: batchData1.Batch.NewStateRoot.String(),
	}, []*types.WrappedBlock{wrappedBlock2}, nil)

}

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()

	// Load config.
	var err error
	cfg, err = config.NewConfig("../config.json")
	if err != nil {
		panic(err)
	}
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	cfg.DBConfig = base.DBConfig

	m.Run()

	base.Free()
}
