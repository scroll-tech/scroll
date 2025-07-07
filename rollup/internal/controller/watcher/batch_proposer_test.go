package watcher

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/internal/utils"
)

func testBatchProposerLimitsCodecV7(t *testing.T) {
	tests := []struct {
		name                       string
		batchTimeoutSec            uint64
		expectedBatchesLen         int
		expectedChunksInFirstBatch uint64 // only be checked when expectedBatchesLen > 0
	}{
		{
			name:               "NoLimitReached",
			batchTimeoutSec:    1000000000000,
			expectedBatchesLen: 0,
		},
		{
			name:                       "Timeout",
			batchTimeoutSec:            0,
			expectedBatchesLen:         1,
			expectedChunksInFirstBatch: 2,
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
			}
			chunk := &encoding.Chunk{
				Blocks: []*encoding.Block{block},
			}
			chunkOrm := orm.NewChunk(db)
			_, err := chunkOrm.InsertChunk(context.Background(), chunk, encoding.CodecV0, utils.ChunkMetrics{})
			assert.NoError(t, err)
			batch := &encoding.Batch{
				Index:                      0,
				TotalL1MessagePoppedBefore: 0,
				ParentBatchHash:            common.Hash{},
				Chunks:                     []*encoding.Chunk{chunk},
			}
			batchOrm := orm.NewBatch(db)
			_, err = batchOrm.InsertBatch(context.Background(), batch, encoding.CodecV0, utils.BatchMetrics{})
			assert.NoError(t, err)

			l2BlockOrm := orm.NewL2Block(db)
			err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
			assert.NoError(t, err)

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:           1,
				MaxL2GasPerChunk:              20000000,
				ChunkTimeoutSec:               300,
				MaxUncompressedBatchBytesSize: math.MaxUint64,
			}, encoding.CodecV7, &params.ChainConfig{
				LondonBlock:    big.NewInt(0),
				BernoulliBlock: big.NewInt(0),
				CurieBlock:     big.NewInt(0),
				DarwinTime:     new(uint64),
				DarwinV2Time:   new(uint64),
				EuclidTime:     new(uint64),
				EuclidV2Time:   new(uint64),
			}, db, nil)
			cp.TryProposeChunk() // chunk1 contains block1
			cp.TryProposeChunk() // chunk2 contains block2

			bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
				MaxChunksPerBatch:             math.MaxInt32,
				BatchTimeoutSec:               tt.batchTimeoutSec,
				MaxUncompressedBatchBytesSize: math.MaxUint64,
			}, encoding.CodecV7, &params.ChainConfig{
				LondonBlock:    big.NewInt(0),
				BernoulliBlock: big.NewInt(0),
				CurieBlock:     big.NewInt(0),
				DarwinTime:     new(uint64),
				DarwinV2Time:   new(uint64),
				EuclidTime:     new(uint64),
				EuclidV2Time:   new(uint64),
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

func testBatchProposerBlobSizeLimitCodecV7(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	// Add genesis batch.
	block := &encoding.Block{
		Header: &gethTypes.Header{
			Number: big.NewInt(0),
		},
	}
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block},
	}
	chunkOrm := orm.NewChunk(db)
	_, err := chunkOrm.InsertChunk(context.Background(), chunk, encoding.CodecV0, utils.ChunkMetrics{})
	assert.NoError(t, err)
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}
	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), batch, encoding.CodecV0, utils.BatchMetrics{})
	assert.NoError(t, err)

	chainConfig := &params.ChainConfig{LondonBlock: big.NewInt(0), BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64), DarwinV2Time: new(uint64), EuclidTime: new(uint64), EuclidV2Time: new(uint64)}

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:           math.MaxUint64,
		MaxL2GasPerChunk:              math.MaxUint64,
		ChunkTimeoutSec:               0,
		MaxUncompressedBatchBytesSize: math.MaxUint64,
	}, encoding.CodecV7, chainConfig, db, nil)

	blockHeight := uint64(0)
	block = readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")
	for total := int64(0); total < 90; total++ {
		for i := int64(0); i < 30; i++ {
			blockHeight++
			l2BlockOrm := orm.NewL2Block(db)
			block.Header.Number = new(big.Int).SetUint64(blockHeight)
			block.Header.Time = blockHeight
			err = l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block})
			assert.NoError(t, err)
		}
		cp.TryProposeChunk()
	}

	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunksPerBatch:             math.MaxInt32,
		BatchTimeoutSec:               math.MaxUint32,
		MaxUncompressedBatchBytesSize: math.MaxUint64,
	}, encoding.CodecV7, chainConfig, db, nil)

	for i := 0; i < 2; i++ {
		bp.TryProposeBatch()
	}

	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
	batches = batches[1:]
	assert.NoError(t, err)

	var expectedNumBatches int = 1
	var numChunksMultiplier uint64 = 64
	assert.Len(t, batches, expectedNumBatches)

	for i, batch := range batches {
		assert.Equal(t, numChunksMultiplier*(uint64(i)+1), batch.EndChunkIndex)
	}
}

func testBatchProposerMaxChunkNumPerBatchLimitCodecV7(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	// Add genesis batch.
	block := &encoding.Block{
		Header: &gethTypes.Header{
			Number: big.NewInt(0),
		},
	}
	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{block},
	}
	chunkOrm := orm.NewChunk(db)
	_, err := chunkOrm.InsertChunk(context.Background(), chunk, encoding.CodecV0, utils.ChunkMetrics{})
	assert.NoError(t, err)
	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}
	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), batch, encoding.CodecV0, utils.BatchMetrics{})
	assert.NoError(t, err)

	var expectedChunkNum uint64 = 45
	chainConfig := &params.ChainConfig{LondonBlock: big.NewInt(0), BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64), DarwinV2Time: new(uint64), EuclidTime: new(uint64), EuclidV2Time: new(uint64)}

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:           math.MaxUint64,
		MaxL2GasPerChunk:              math.MaxUint64,
		ChunkTimeoutSec:               0,
		MaxUncompressedBatchBytesSize: math.MaxUint64,
	}, encoding.CodecV7, chainConfig, db, nil)

	block = readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")
	for blockHeight := uint64(1); blockHeight <= 60; blockHeight++ {
		block.Header.Number = new(big.Int).SetUint64(blockHeight)
		block.Header.Time = blockHeight
		err = orm.NewL2Block(db).InsertL2Blocks(context.Background(), []*encoding.Block{block})
		assert.NoError(t, err)
		cp.TryProposeChunk()
	}

	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunksPerBatch:             45,
		BatchTimeoutSec:               math.MaxUint32,
		MaxUncompressedBatchBytesSize: math.MaxUint64,
	}, encoding.CodecV7, chainConfig, db, nil)
	bp.TryProposeBatch()

	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, batches, 2)
	dbBatch := batches[1]

	assert.Equal(t, expectedChunkNum, dbBatch.EndChunkIndex)
}

func testBatchProposerUncompressedBatchBytesLimitCodecV8(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	// Add genesis batch
	genesisBlock := &encoding.Block{
		Header: &gethTypes.Header{
			Number: big.NewInt(0),
		},
	}
	genesisChunk := &encoding.Chunk{
		Blocks: []*encoding.Block{genesisBlock},
	}
	chunkOrm := orm.NewChunk(db)
	_, err := chunkOrm.InsertChunk(context.Background(), genesisChunk, encoding.CodecV0, utils.ChunkMetrics{})
	assert.NoError(t, err)

	genesisBatch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{genesisChunk},
	}
	batchOrm := orm.NewBatch(db)
	_, err = batchOrm.InsertBatch(context.Background(), genesisBatch, encoding.CodecV0, utils.BatchMetrics{})
	assert.NoError(t, err)

	// Create blocks with large calldata
	block := readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")

	// Create large calldata (3KiB per block)
	largeCalldata := make([]byte, 3*1024) // 3KiB calldata
	for i := range largeCalldata {
		largeCalldata[i] = byte(i % 256)
	}

	// Modify the block to have a transaction with large calldata
	block.Transactions[0].Data = "0x" + common.Bytes2Hex(largeCalldata)

	chainConfig := &params.ChainConfig{
		LondonBlock:    big.NewInt(0),
		BernoulliBlock: big.NewInt(0),
		CurieBlock:     big.NewInt(0),
		DarwinTime:     new(uint64),
		DarwinV2Time:   new(uint64),
		EuclidTime:     new(uint64),
		EuclidV2Time:   new(uint64),
		FeynmanTime:    new(uint64),
	}

	// Create chunk proposer with no uncompressed batch bytes limit for chunks
	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:           1, // One block per chunk
		MaxL2GasPerChunk:              math.MaxUint64,
		ChunkTimeoutSec:               math.MaxUint32,
		MaxUncompressedBatchBytesSize: math.MaxUint64,
	}, encoding.CodecV8, chainConfig, db, nil)

	// Insert 2 blocks with large calldata and create 2 chunks
	l2BlockOrm := orm.NewL2Block(db)
	for i := uint64(1); i <= 2; i++ {
		blockCopy := *block
		blockCopy.Header = &gethTypes.Header{}
		*blockCopy.Header = *block.Header
		blockCopy.Header.Number = new(big.Int).SetUint64(i)
		blockCopy.Header.Time = i
		err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{&blockCopy})
		assert.NoError(t, err)

		cp.TryProposeChunk() // Each call creates one chunk with one block
	}

	// Create batch proposer with 4KiB uncompressed batch bytes limit
	// Each chunk is ~3KiB, so 1 chunk (~3KiB) should fit, but 2 chunks (~6KiB) should exceed limit
	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunksPerBatch:             math.MaxInt32,  // No chunk count limit
		BatchTimeoutSec:               math.MaxUint32, // No timeout limit
		MaxUncompressedBatchBytesSize: 4 * 1024,       // 4KiB limit
	}, encoding.CodecV8, chainConfig, db, nil)

	bp.TryProposeBatch()

	// Check that a batch was created
	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, batches, 2) // genesis batch + 1 new batch

	// Verify that the batch contains only 1 chunk (not 2) due to uncompressed batch bytes limit
	newBatch := batches[1] // Skip genesis batch
	assert.Equal(t, uint64(1), newBatch.StartChunkIndex)
	assert.Equal(t, uint64(1), newBatch.EndChunkIndex) // Should only include chunk 1

	// Verify that the second chunk is still available for next batch
	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 2, 0)
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)                 // Second chunk should still be available
	assert.Equal(t, "", chunks[0].BatchHash) // Should not be assigned to any batch yet
}
