package watcher

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"
	"scroll-tech/common/types/encoding"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

func testBatchProposerCodecv0Limits(t *testing.T) {
	tests := []struct {
		name                       string
		maxChunkNum                uint64
		maxL1CommitGas             uint64
		maxL1CommitCalldataSize    uint64
		batchTimeoutSec            uint64
		forkBlock                  *big.Int
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
		{
			name:                       "ForkBlockReached",
			maxChunkNum:                10,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			batchTimeoutSec:            1000000000000,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 1,
			forkBlock:                  big.NewInt(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			defer database.CloseDB(db)

			// Add genesis batch.
			block := &encoding.Block{
				Header: &gethTypes.Header{
					Number: big.NewInt(0),
				},
				RowConsumption: &gethTypes.RowConsumption{},
			}
			chunk := &encoding.Chunk{
				Blocks: []*encoding.Block{block},
			}
			chunkOrm := orm.NewChunk(db)
			_, err := chunkOrm.InsertChunk(context.Background(), chunk, encoding.CodecV0)
			assert.NoError(t, err)
			batch := &encoding.Batch{
				Index:                      0,
				TotalL1MessagePoppedBefore: 0,
				ParentBatchHash:            common.Hash{},
				Chunks:                     []*encoding.Chunk{chunk},
			}
			batchOrm := orm.NewBatch(db)
			_, err = batchOrm.InsertBatch(context.Background(), batch, encoding.CodecV0)
			assert.NoError(t, err)

			l2BlockOrm := orm.NewL2Block(db)
			err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
			assert.NoError(t, err)

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             1,
				MaxTxNumPerChunk:                10000,
				MaxL1CommitGasPerChunk:          50000000000,
				MaxL1CommitCalldataSizePerChunk: 1000000,
				MaxRowConsumptionPerChunk:       1000000,
				ChunkTimeoutSec:                 300,
				GasCostIncreaseMultiplier:       1.2,
			}, &params.ChainConfig{
				HomesteadBlock: tt.forkBlock,
			}, db, nil)
			cp.TryProposeChunk() // chunk1 contains block1
			cp.TryProposeChunk() // chunk2 contains block2

			chunks, err := chunkOrm.GetChunksInRange(context.Background(), 1, 2)
			assert.NoError(t, err)
			assert.Equal(t, uint64(6042), chunks[0].TotalL1CommitGas)
			assert.Equal(t, uint64(298), chunks[0].TotalL1CommitCalldataSize)
			assert.Equal(t, uint64(94618), chunks[1].TotalL1CommitGas)
			assert.Equal(t, uint64(5737), chunks[1].TotalL1CommitCalldataSize)

			bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
				MaxChunkNumPerBatch:             tt.maxChunkNum,
				MaxL1CommitGasPerBatch:          tt.maxL1CommitGas,
				MaxL1CommitCalldataSizePerBatch: tt.maxL1CommitCalldataSize,
				BatchTimeoutSec:                 tt.batchTimeoutSec,
				GasCostIncreaseMultiplier:       1.2,
			}, &params.ChainConfig{
				HomesteadBlock: tt.forkBlock,
			}, db, nil)
			bp.TryProposeBatch()

			batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
			assert.NoError(t, err)
			assert.Len(t, batches, tt.expectedBatchesLen+1)
			batches = batches[1:]
			if tt.expectedBatchesLen > 0 {
				assert.Equal(t, uint64(1), batches[0].StartChunkIndex)
				assert.Equal(t, tt.expectedChunksInFirstBatch, batches[0].EndChunkIndex)
				assert.Equal(t, types.RollupPending, types.RollupStatus(batches[0].RollupStatus))
				assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(batches[0].ProvingStatus))

				dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), 1, tt.expectedChunksInFirstBatch)
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

func testBatchProposerCodecv1Limits(t *testing.T) {
	tests := []struct {
		name                       string
		maxChunkNum                uint64
		batchTimeoutSec            uint64
		forkBlock                  *big.Int
		expectedBatchesLen         int
		expectedChunksInFirstBatch uint64 // only be checked when expectedBatchesLen > 0
	}{
		{
			name:               "NoLimitReached",
			maxChunkNum:        10,
			batchTimeoutSec:    1000000000000,
			expectedBatchesLen: 0,
		},
		{
			name:                       "Timeout",
			maxChunkNum:                10,
			batchTimeoutSec:            0,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 2,
		},
		{
			name:                       "MaxChunkNumPerBatchIs1",
			maxChunkNum:                1,
			batchTimeoutSec:            1000000000000,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 1,
		},
		{
			name:                       "ForkBlockReached",
			maxChunkNum:                10,
			batchTimeoutSec:            1000000000000,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 1,
			forkBlock:                  big.NewInt(3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			defer database.CloseDB(db)

			// Add genesis batch.
			block := &encoding.Block{
				Header: &gethTypes.Header{
					Number: big.NewInt(0),
				},
				RowConsumption: &gethTypes.RowConsumption{},
			}
			chunk := &encoding.Chunk{
				Blocks: []*encoding.Block{block},
			}
			chunkOrm := orm.NewChunk(db)
			_, err := chunkOrm.InsertChunk(context.Background(), chunk, encoding.CodecV1)
			assert.NoError(t, err)
			batch := &encoding.Batch{
				Index:                      0,
				TotalL1MessagePoppedBefore: 0,
				ParentBatchHash:            common.Hash{},
				Chunks:                     []*encoding.Chunk{chunk},
			}
			batchOrm := orm.NewBatch(db)
			_, err = batchOrm.InsertBatch(context.Background(), batch, encoding.CodecV1)
			assert.NoError(t, err)

			l2BlockOrm := orm.NewL2Block(db)
			err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
			assert.NoError(t, err)

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             1,
				MaxTxNumPerChunk:                10000,
				MaxL1CommitGasPerChunk:          1,
				MaxL1CommitCalldataSizePerChunk: 1,
				MaxRowConsumptionPerChunk:       1000000,
				ChunkTimeoutSec:                 300,
				GasCostIncreaseMultiplier:       1.2,
			}, &params.ChainConfig{
				BernoulliBlock: big.NewInt(0), HomesteadBlock: tt.forkBlock,
			}, db, nil)
			cp.TryProposeChunk() // chunk1 contains block1
			cp.TryProposeChunk() // chunk2 contains block2

			chunks, err := chunkOrm.GetChunksInRange(context.Background(), 1, 2)
			assert.NoError(t, err)
			assert.Equal(t, uint64(0), chunks[0].TotalL1CommitGas)
			assert.Equal(t, uint64(0), chunks[0].TotalL1CommitCalldataSize)
			assert.Equal(t, uint64(0), chunks[1].TotalL1CommitGas)
			assert.Equal(t, uint64(0), chunks[1].TotalL1CommitCalldataSize)

			bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
				MaxChunkNumPerBatch:             tt.maxChunkNum,
				MaxL1CommitGasPerBatch:          1,
				MaxL1CommitCalldataSizePerBatch: 1,
				BatchTimeoutSec:                 tt.batchTimeoutSec,
				GasCostIncreaseMultiplier:       1.2,
			}, &params.ChainConfig{
				BernoulliBlock: big.NewInt(0), HomesteadBlock: tt.forkBlock,
			}, db, nil)
			bp.TryProposeBatch()

			batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
			assert.NoError(t, err)
			assert.Len(t, batches, tt.expectedBatchesLen+1)
			batches = batches[1:]
			if tt.expectedBatchesLen > 0 {
				assert.Equal(t, uint64(1), batches[0].StartChunkIndex)
				assert.Equal(t, tt.expectedChunksInFirstBatch, batches[0].EndChunkIndex)
				assert.Equal(t, types.RollupPending, types.RollupStatus(batches[0].RollupStatus))
				assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(batches[0].ProvingStatus))

				dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), 1, tt.expectedChunksInFirstBatch)
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

	// Add genesis batch.
	block := &encoding.Block{
		Header: &gethTypes.Header{
			Number: big.NewInt(0),
		},
		RowConsumption: &gethTypes.RowConsumption{},
	}
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block},
	}
	chunkOrm := orm.NewChunk(db)
	_, err := chunkOrm.InsertChunk(context.Background(), chunk, encoding.CodecV0)
	assert.NoError(t, err)
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}
	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), batch, encoding.CodecV0)
	assert.NoError(t, err)

	l2BlockOrm := orm.NewL2Block(db)
	err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
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

	chunks, err := chunkOrm.GetChunksInRange(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, uint64(6042), chunks[0].TotalL1CommitGas)
	assert.Equal(t, uint64(298), chunks[0].TotalL1CommitCalldataSize)
	assert.Equal(t, uint64(94618), chunks[1].TotalL1CommitGas)
	assert.Equal(t, uint64(5737), chunks[1].TotalL1CommitCalldataSize)

	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunkNumPerBatch:             10,
		MaxL1CommitGasPerBatch:          50000000000,
		MaxL1CommitCalldataSizePerBatch: 1000000,
		BatchTimeoutSec:                 0,
		GasCostIncreaseMultiplier:       1.2,
	}, &params.ChainConfig{}, db, nil)
	bp.TryProposeBatch()

	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, batches, 2)
	batches = batches[1:]
	assert.Equal(t, uint64(1), batches[0].StartChunkIndex)
	assert.Equal(t, uint64(2), batches[0].EndChunkIndex)
	assert.Equal(t, types.RollupPending, types.RollupStatus(batches[0].RollupStatus))
	assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(batches[0].ProvingStatus))

	dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Len(t, dbChunks, 2)
	for _, chunk := range dbChunks {
		assert.Equal(t, batches[0].Hash, chunk.BatchHash)
		assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(chunk.ProvingStatus))
	}

	assert.Equal(t, uint64(258383), batches[0].TotalL1CommitGas)
	assert.Equal(t, uint64(6035), batches[0].TotalL1CommitCalldataSize)
}

func testBatchProposerBlobSizeLimit(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	// Add genesis batch.
	block := &encoding.Block{
		Header: &gethTypes.Header{
			Number: big.NewInt(0),
		},
		RowConsumption: &gethTypes.RowConsumption{},
	}
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block},
	}
	chunkOrm := orm.NewChunk(db)
	_, err := chunkOrm.InsertChunk(context.Background(), chunk, encoding.CodecV1)
	assert.NoError(t, err)
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}
	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), batch, encoding.CodecV1)
	assert.NoError(t, err)

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             math.MaxUint64,
		MaxTxNumPerChunk:                math.MaxUint64,
		MaxL1CommitGasPerChunk:          1,
		MaxL1CommitCalldataSizePerChunk: 1,
		MaxRowConsumptionPerChunk:       math.MaxUint64,
		ChunkTimeoutSec:                 math.MaxUint64,
		GasCostIncreaseMultiplier:       1,
	}, &params.ChainConfig{BernoulliBlock: big.NewInt(0)}, db, nil)

	block = readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")
	for total := int64(0); total < 7; total++ {
		for i := int64(0); i < 10; i++ {
			l2BlockOrm := orm.NewL2Block(db)
			block.Header.Number = big.NewInt(total*10 + i + 1)
			err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block})
			assert.NoError(t, err)
		}
		cp.TryProposeChunk()
	}

	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunkNumPerBatch:             math.MaxUint64,
		MaxL1CommitGasPerBatch:          1,
		MaxL1CommitCalldataSizePerBatch: 1,
		BatchTimeoutSec:                 math.MaxUint64,
		GasCostIncreaseMultiplier:       1,
	}, &params.ChainConfig{BernoulliBlock: big.NewInt(0)}, db, nil)

	for i := 0; i < 10; i++ {
		bp.TryProposeBatch()
	}

	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
	batches = batches[1:]
	assert.NoError(t, err)
	assert.Len(t, batches, 4)
	for i, batch := range batches {
		expected := uint64(2 * (i + 1))
		if expected > 7 {
			expected = 7
		}
		assert.Equal(t, expected, batch.EndChunkIndex)
	}
}
