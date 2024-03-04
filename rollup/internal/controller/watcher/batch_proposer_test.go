package watcher

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

func testBatchProposerLimits(t *testing.T) {
	tests := []struct {
		name                       string
		maxChunkNum                uint64
		maxL1CommitGas             uint64
		maxL1CommitCalldataSize    uint32
		batchTimeoutSec            uint64
		expectedBatchesLen         int
		expectedChunksInFirstBatch uint64 // only be checked when expectedBatchesLen > 0
	}{
		{
			name:                    "NoLimitReached",
			maxChunkNum:             10,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			batchTimeoutSec:         1000000000000,
			expectedBatchesLen:      0,
		},
		{
			name:                       "Timeout",
			maxChunkNum:                10,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			batchTimeoutSec:            0,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 2,
		},
		{
			name:                    "MaxL1CommitGasPerBatchIs0",
			maxChunkNum:             10,
			maxL1CommitGas:          0,
			maxL1CommitCalldataSize: 1000000,
			batchTimeoutSec:         1000000000000,
			expectedBatchesLen:      0,
		},
		{
			name:                    "MaxL1CommitCalldataSizePerBatchIs0",
			maxChunkNum:             10,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 0,
			batchTimeoutSec:         1000000000000,
			expectedBatchesLen:      0,
		},
		{
			name:                       "MaxChunkNumPerBatchIs1",
			maxChunkNum:                1,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			batchTimeoutSec:            1000000000000,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 1,
		},
		{
			name:                       "MaxL1CommitGasPerBatchIsFirstChunk",
			maxChunkNum:                10,
			maxL1CommitGas:             200000,
			maxL1CommitCalldataSize:    1000000,
			batchTimeoutSec:            1000000000000,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 1,
		},
		{
			name:                       "MaxL1CommitCalldataSizePerBatchIsFirstChunk",
			maxChunkNum:                10,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    298,
			batchTimeoutSec:            1000000000000,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			defer database.CloseDB(db)

			l2BlockOrm := orm.NewL2Block(db)
			err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
			assert.NoError(t, err)

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             1,
				MaxTxNumPerChunk:                10000,
				MaxL1CommitGasPerChunk:          50000000000,
				MaxL1CommitCalldataSizePerChunk: 1000000,
				MaxRowConsumptionPerChunk:       1000000,
				ChunkTimeoutSec:                 300,
				GasCostIncreaseMultiplier:       1.2,
			}, &params.ChainConfig{}, db, nil)
			cp.TryProposeChunk() // chunk1 contains block1
			cp.TryProposeChunk() // chunk2 contains block2

			chunkOrm := orm.NewChunk(db)
			chunks, err := chunkOrm.GetChunksInRange(context.Background(), 0, 1)
			assert.NoError(t, err)
			assert.Equal(t, uint64(6042), chunks[0].TotalL1CommitGas)
			assert.Equal(t, uint32(298), chunks[0].TotalL1CommitCalldataSize)
			assert.Equal(t, uint64(94586), chunks[1].TotalL1CommitGas)
			assert.Equal(t, uint32(5735), chunks[1].TotalL1CommitCalldataSize)

			bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
				MaxChunkNumPerBatch:             tt.maxChunkNum,
				MaxL1CommitGasPerBatch:          tt.maxL1CommitGas,
				MaxL1CommitCalldataSizePerBatch: tt.maxL1CommitCalldataSize,
				BatchTimeoutSec:                 tt.batchTimeoutSec,
				GasCostIncreaseMultiplier:       1.2,
			}, db, nil)
			bp.TryProposeBatch()

			batchOrm := orm.NewBatch(db)
			batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
			assert.NoError(t, err)
			assert.Len(t, batches, tt.expectedBatchesLen)
			if tt.expectedBatchesLen > 0 {
				assert.Equal(t, uint64(0), batches[0].StartChunkIndex)
				assert.Equal(t, tt.expectedChunksInFirstBatch-1, batches[0].EndChunkIndex)
				assert.Equal(t, types.RollupPending, types.RollupStatus(batches[0].RollupStatus))
				assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(batches[0].ProvingStatus))

				dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), 0, tt.expectedChunksInFirstBatch-1)
				assert.NoError(t, err)
				assert.Len(t, dbChunks, int(tt.expectedChunksInFirstBatch))
				for _, chunk := range dbChunks {
					assert.Equal(t, batches[0].Hash, chunk.BatchHash)
					assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(chunk.ProvingStatus))
				}
			}
		})
	}
}

func testBatchCommitGasAndCalldataSizeEstimation(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	l2BlockOrm := orm.NewL2Block(db)
	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             1,
		MaxTxNumPerChunk:                10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MaxRowConsumptionPerChunk:       1000000,
		ChunkTimeoutSec:                 300,
		GasCostIncreaseMultiplier:       1.2,
	}, &params.ChainConfig{}, db, nil)
	cp.TryProposeChunk() // chunk1 contains block1
	cp.TryProposeChunk() // chunk2 contains block2

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetChunksInRange(context.Background(), 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6042), chunks[0].TotalL1CommitGas)
	assert.Equal(t, uint32(298), chunks[0].TotalL1CommitCalldataSize)
	assert.Equal(t, uint64(94586), chunks[1].TotalL1CommitGas)
	assert.Equal(t, uint32(5735), chunks[1].TotalL1CommitCalldataSize)

	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunkNumPerBatch:             10,
		MaxL1CommitGasPerBatch:          50000000000,
		MaxL1CommitCalldataSizePerBatch: 1000000,
		BatchTimeoutSec:                 0,
		GasCostIncreaseMultiplier:       1.2,
	}, db, nil)
	bp.TryProposeBatch()

	batchOrm := orm.NewBatch(db)
	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, batches, 1)
	assert.Equal(t, uint64(0), batches[0].StartChunkIndex)
	assert.Equal(t, uint64(1), batches[0].EndChunkIndex)
	assert.Equal(t, types.RollupPending, types.RollupStatus(batches[0].RollupStatus))
	assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(batches[0].ProvingStatus))

	dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), 0, 1)
	assert.NoError(t, err)
	assert.Len(t, dbChunks, 2)
	for _, chunk := range dbChunks {
		assert.Equal(t, batches[0].Hash, chunk.BatchHash)
		assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(chunk.ProvingStatus))
	}

	assert.Equal(t, uint64(254562), batches[0].TotalL1CommitGas)
	assert.Equal(t, uint32(6033), batches[0].TotalL1CommitCalldataSize)
}
