package watcher

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common/math"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	"scroll-tech/rollup/internal/utils"
)

func testChunkProposerLimitsCodecV7(t *testing.T) {
	tests := []struct {
		name                       string
		maxBlockNum                uint64
		maxL2Gas                   uint64
		chunkTimeoutSec            uint64
		expectedChunksLen          int
		expectedBlocksInFirstChunk int // only be checked when expectedChunksLen > 0
	}{
		{
			name:              "NoLimitReached",
			maxBlockNum:       100,
			maxL2Gas:          20_000_000,
			chunkTimeoutSec:   1000000000000,
			expectedChunksLen: 0,
		},
		{
			name:                       "Timeout",
			maxBlockNum:                100,
			maxL2Gas:                   20_000_000,
			chunkTimeoutSec:            0,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 2,
		},
		{
			name:              "MaxL2GasPerChunkIs0",
			maxBlockNum:       10,
			maxL2Gas:          0,
			chunkTimeoutSec:   1000000000000,
			expectedChunksLen: 0,
		},
		{
			name:                       "MaxBlockNumPerChunkIs1",
			maxBlockNum:                1,
			maxL2Gas:                   20_000_000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			// In this test the second block is not included in the chunk because together
			// with the first block it exceeds the maxL2GasPerChunk limit.
			name:                       "MaxL2GasPerChunkIsSecondBlock",
			maxBlockNum:                10,
			maxL2Gas:                   1_153_000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupDB(t)
			defer database.CloseDB(db)

			l2BlockOrm := orm.NewL2Block(db)
			err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block1, block2})
			assert.NoError(t, err)

			// Add genesis chunk.
			chunkOrm := orm.NewChunk(db)
			_, err = chunkOrm.InsertChunk(context.Background(), &encoding.Chunk{Blocks: []*encoding.Block{{Header: &gethTypes.Header{Number: big.NewInt(0)}}}}, encoding.CodecV0, utils.ChunkMetrics{})
			assert.NoError(t, err)

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk: tt.maxBlockNum,
				MaxL2GasPerChunk:    tt.maxL2Gas,
				ChunkTimeoutSec:     tt.chunkTimeoutSec,
			}, encoding.CodecV7, &params.ChainConfig{LondonBlock: big.NewInt(0), BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64), DarwinV2Time: new(uint64), EuclidTime: new(uint64), EuclidV2Time: new(uint64)}, db, nil)
			cp.TryProposeChunk()

			chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 1, 0)
			assert.NoError(t, err)
			assert.Len(t, chunks, tt.expectedChunksLen)

			if len(chunks) > 0 {
				blockOrm := orm.NewL2Block(db)
				chunkHashes, err := blockOrm.GetChunkHashes(context.Background(), tt.expectedBlocksInFirstChunk)
				assert.NoError(t, err)
				assert.Len(t, chunkHashes, tt.expectedBlocksInFirstChunk)
				firstChunkHash := chunks[0].Hash
				for _, chunkHash := range chunkHashes {
					assert.Equal(t, firstChunkHash, chunkHash)
				}
			}
		})
	}
}

func testChunkProposerBlobSizeLimitCodecV7(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)
	block := readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")
	for i := uint64(0); i < 510; i++ {
		l2BlockOrm := orm.NewL2Block(db)
		block.Header.Number = new(big.Int).SetUint64(i + 1)
		block.Header.Time = i + 1
		err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block})
		assert.NoError(t, err)
	}

	// Add genesis chunk.
	chunkOrm := orm.NewChunk(db)
	_, err := chunkOrm.InsertChunk(context.Background(), &encoding.Chunk{Blocks: []*encoding.Block{{Header: &gethTypes.Header{Number: big.NewInt(0)}}}}, encoding.CodecV0, utils.ChunkMetrics{})
	assert.NoError(t, err)

	chainConfig := &params.ChainConfig{LondonBlock: big.NewInt(0), BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64), DarwinV2Time: new(uint64), EuclidTime: new(uint64), EuclidV2Time: new(uint64)}

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk: 255,
		MaxL2GasPerChunk:    math.MaxUint64,
		ChunkTimeoutSec:     math.MaxUint32,
	}, encoding.CodecV7, chainConfig, db, nil)

	for i := 0; i < 2; i++ {
		cp.TryProposeChunk()
	}

	chunkOrm = orm.NewChunk(db)
	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 1, 0)
	assert.NoError(t, err)

	var expectedNumChunks int = 2
	var numBlocksMultiplier uint64 = 255
	assert.Len(t, chunks, expectedNumChunks)

	for i, chunk := range chunks {
		expected := numBlocksMultiplier * (uint64(i) + 1)
		if expected > 2000 {
			expected = 2000
		}
		assert.Equal(t, expected, chunk.EndBlockNumber)
	}
}
