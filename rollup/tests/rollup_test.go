package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/controller/relayer"
	"scroll-tech/rollup/internal/controller/watcher"
	"scroll-tech/rollup/internal/orm"
)

func testCommitAndFinalizeGenesisBatch(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	prepareContracts(t)

	l2Cfg := rollupApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, true, relayer.ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)
	assert.NotNil(t, l2Relayer)

	genesisChunkHash := common.HexToHash("0x00e076380b00a3749816fcc9a2a576b529952ef463222a54544d21b7d434dfe1")
	chunkOrm := orm.NewChunk(db)
	dbChunk, err := chunkOrm.GetChunksInRange(context.Background(), 0, 0)
	assert.NoError(t, err)
	assert.Len(t, dbChunk, 1)
	assert.Equal(t, genesisChunkHash.String(), dbChunk[0].Hash)
	assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(dbChunk[0].ProvingStatus))

	genesisBatchHash := common.HexToHash("0x2d214b024f5337d83a5681f88575ab225f345ec2e4e3ce53cf4dc4b0cb5c96b1")
	batchOrm := orm.NewBatch(db)
	batch, err := batchOrm.GetBatchByIndex(context.Background(), 0)
	assert.NoError(t, err)
	assert.Equal(t, genesisBatchHash.String(), batch.Hash)
	assert.Equal(t, types.ProvingTaskVerified, types.ProvingStatus(batch.ProvingStatus))
	assert.Equal(t, types.RollupFinalized, types.RollupStatus(batch.RollupStatus))
}

func testCommitBatchAndFinalizeBatch(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	prepareContracts(t)

	// Create L2Relayer
	l2Cfg := rollupApp.Config.L2Config
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig, false, relayer.ServiceTypeL2RollupRelayer, nil)
	assert.NoError(t, err)

	// Create L1Watcher
	l1Cfg := rollupApp.Config.L1Config
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, l1Cfg.Confirmations, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db, nil)

	// add some blocks to db
	var wrappedBlocks []*types.WrappedBlock
	for i := 0; i < 10; i++ {
		header := gethTypes.Header{
			Number:     big.NewInt(int64(i)),
			ParentHash: common.Hash{},
			Difficulty: big.NewInt(0),
			BaseFee:    big.NewInt(0),
		}
		wrappedBlocks = append(wrappedBlocks, &types.WrappedBlock{
			Header:         &header,
			Transactions:   nil,
			WithdrawRoot:   common.Hash{},
			RowConsumption: &gethTypes.RowConsumption{},
		})
	}

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(context.Background(), wrappedBlocks)
	assert.NoError(t, err)

	cp := watcher.NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             100,
		MaxTxNumPerChunk:                10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MaxRowConsumptionPerChunk:       1048319,
		ChunkTimeoutSec:                 300,
	}, &params.ChainConfig{}, db, nil)
	cp.TryProposeChunk()

	batchOrm := orm.NewBatch(db)
	unbatchedChunkIndex, err := batchOrm.GetFirstUnbatchedChunkIndex(context.Background())
	assert.NoError(t, err)

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), unbatchedChunkIndex, 0)
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)

	bp := watcher.NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunkNumPerBatch:             10,
		MaxL1CommitGasPerBatch:          50000000000,
		MaxL1CommitCalldataSizePerBatch: 1000000,
		BatchTimeoutSec:                 300,
	}, &params.ChainConfig{}, db, nil)
	bp.TryProposeBatch()

	l2Relayer.ProcessPendingBatches()

	batch, err := batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, batch)
	batchHash := batch.Hash
	assert.NotEmpty(t, batch.CommitTxHash)
	assert.Equal(t, types.RollupCommitting, types.RollupStatus(batch.RollupStatus))

	success := utils.TryTimes(30, func() bool {
		var receipt *gethTypes.Receipt
		receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.CommitTxHash))
		return err == nil && receipt.Status == 1
	})
	assert.True(t, success)

	// fetch rollup events
	success = utils.TryTimes(30, func() bool {
		err = l1Watcher.FetchContractEvent()
		assert.NoError(t, err)
		var statuses []types.RollupStatus
		statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash})
		return err == nil && len(statuses) == 1 && types.RollupCommitted == statuses[0]
	})
	assert.True(t, success)

	// add dummy proof
	proof := &message.BatchProof{
		Proof: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = batchOrm.UpdateProofByHash(context.Background(), batchHash, proof, 100)
	assert.NoError(t, err)
	err = batchOrm.UpdateProvingStatus(context.Background(), batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()

	statuses, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalizing, statuses[0])

	batch, err = batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, batch)
	assert.NotEmpty(t, batch.FinalizeTxHash)

	success = utils.TryTimes(30, func() bool {
		var receipt *gethTypes.Receipt
		receipt, err = l1Client.TransactionReceipt(context.Background(), common.HexToHash(batch.FinalizeTxHash))
		return err == nil && receipt.Status == 1
	})
	assert.True(t, success)

	// fetch rollup events
	success = utils.TryTimes(30, func() bool {
		err = l1Watcher.FetchContractEvent()
		assert.NoError(t, err)
		var statuses []types.RollupStatus
		statuses, err = batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash})
		return err == nil && len(statuses) == 1 && types.RollupFinalized == statuses[0]
	})
	assert.True(t, success)
}
