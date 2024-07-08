package watcher

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common/math"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

func testChunkProposerCodecv0Limits(t *testing.T) {
	tests := []struct {
		name                       string
		maxBlockNum                uint64
		maxTxNum                   uint64
		maxL1CommitGas             uint64
		maxL1CommitCalldataSize    uint64
		maxRowConsumption          uint64
		chunkTimeoutSec            uint64
		expectedChunksLen          int
		expectedBlocksInFirstChunk int // only be checked when expectedChunksLen > 0
	}{
		{
			name:                    "NoLimitReached",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "Timeout",
			maxBlockNum:                100,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            0,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 2,
		},
		{
			name:                    "MaxTxNumPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                0,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitGasPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          0,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitCalldataSizePerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 0,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxRowConsumptionPerChunkIs0",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       0,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "MaxBlockNumPerChunkIs1",
			maxBlockNum:                1,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxTxNumPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   2,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitGasPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             7250,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitCalldataSizePerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    298,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxRowConsumptionPerChunkIs1",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1,
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

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             tt.maxBlockNum,
				MaxTxNumPerChunk:                tt.maxTxNum,
				MaxL1CommitGasPerChunk:          tt.maxL1CommitGas,
				MaxL1CommitCalldataSizePerChunk: tt.maxL1CommitCalldataSize,
				MaxRowConsumptionPerChunk:       tt.maxRowConsumption,
				ChunkTimeoutSec:                 tt.chunkTimeoutSec,
				GasCostIncreaseMultiplier:       1.2,
			}, &params.ChainConfig{}, db, nil)
			cp.TryProposeChunk()

			chunkOrm := orm.NewChunk(db)
			chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
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

func testChunkProposerCodecv1Limits(t *testing.T) {
	tests := []struct {
		name                       string
		maxBlockNum                uint64
		maxTxNum                   uint64
		maxL1CommitGas             uint64
		maxL1CommitCalldataSize    uint64
		maxRowConsumption          uint64
		chunkTimeoutSec            uint64
		expectedChunksLen          int
		expectedBlocksInFirstChunk int // only be checked when expectedChunksLen > 0
	}{
		{
			name:                    "NoLimitReached",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "Timeout",
			maxBlockNum:                100,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            0,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 2,
		},
		{
			name:                    "MaxTxNumPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                0,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitGasPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          0,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitCalldataSizePerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 0,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxRowConsumptionPerChunkIs0",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       0,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "MaxBlockNumPerChunkIs1",
			maxBlockNum:                1,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxTxNumPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   2,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitGasPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             2500,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitCalldataSizePerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    60,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxRowConsumptionPerChunkIs1",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1,
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

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             tt.maxBlockNum,
				MaxTxNumPerChunk:                tt.maxTxNum,
				MaxL1CommitGasPerChunk:          tt.maxL1CommitGas,
				MaxL1CommitCalldataSizePerChunk: tt.maxL1CommitCalldataSize,
				MaxRowConsumptionPerChunk:       tt.maxRowConsumption,
				ChunkTimeoutSec:                 tt.chunkTimeoutSec,
				GasCostIncreaseMultiplier:       1.2,
			}, &params.ChainConfig{BernoulliBlock: big.NewInt(0)}, db, nil)
			cp.TryProposeChunk()

			chunkOrm := orm.NewChunk(db)
			chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
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

func testChunkProposerCodecv2Limits(t *testing.T) {
	tests := []struct {
		name                       string
		maxBlockNum                uint64
		maxTxNum                   uint64
		maxL1CommitGas             uint64
		maxL1CommitCalldataSize    uint64
		maxRowConsumption          uint64
		chunkTimeoutSec            uint64
		expectedChunksLen          int
		expectedBlocksInFirstChunk int // only be checked when expectedChunksLen > 0
	}{
		{
			name:                    "NoLimitReached",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "Timeout",
			maxBlockNum:                100,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            0,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 2,
		},
		{
			name:                    "MaxTxNumPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                0,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitGasPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          0,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitCalldataSizePerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 0,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxRowConsumptionPerChunkIs0",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       0,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "MaxBlockNumPerChunkIs1",
			maxBlockNum:                1,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxTxNumPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   2,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitGasPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             2500,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitCalldataSizePerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    60,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxRowConsumptionPerChunkIs1",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1,
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

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             tt.maxBlockNum,
				MaxTxNumPerChunk:                tt.maxTxNum,
				MaxL1CommitGasPerChunk:          tt.maxL1CommitGas,
				MaxL1CommitCalldataSizePerChunk: tt.maxL1CommitCalldataSize,
				MaxRowConsumptionPerChunk:       tt.maxRowConsumption,
				ChunkTimeoutSec:                 tt.chunkTimeoutSec,
				GasCostIncreaseMultiplier:       1.2,
				MaxUncompressedBatchBytesSize:   math.MaxUint64,
			}, &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0)}, db, nil)
			cp.TryProposeChunk()

			chunkOrm := orm.NewChunk(db)
			chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
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

func testChunkProposerCodecv3Limits(t *testing.T) {
	tests := []struct {
		name                       string
		maxBlockNum                uint64
		maxTxNum                   uint64
		maxL1CommitGas             uint64
		maxL1CommitCalldataSize    uint64
		maxRowConsumption          uint64
		chunkTimeoutSec            uint64
		expectedChunksLen          int
		expectedBlocksInFirstChunk int // only be checked when expectedChunksLen > 0
	}{
		{
			name:                    "NoLimitReached",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "Timeout",
			maxBlockNum:                100,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            0,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 2,
		},
		{
			name:                    "MaxTxNumPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                0,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitGasPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          0,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitCalldataSizePerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 0,
			maxRowConsumption:       1000000,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxRowConsumptionPerChunkIs0",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       0,
			chunkTimeoutSec:         1000000000000,
			expectedChunksLen:       0,
		},
		{
			name:                       "MaxBlockNumPerChunkIs1",
			maxBlockNum:                1,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxTxNumPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   2,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitGasPerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             62500,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxL1CommitCalldataSizePerChunkIsFirstBlock",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    60,
			maxRowConsumption:          1000000,
			chunkTimeoutSec:            1000000000000,
			expectedChunksLen:          1,
			expectedBlocksInFirstChunk: 1,
		},
		{
			name:                       "MaxRowConsumptionPerChunkIs1",
			maxBlockNum:                10,
			maxTxNum:                   10000,
			maxL1CommitGas:             50000000000,
			maxL1CommitCalldataSize:    1000000,
			maxRowConsumption:          1,
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

			cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
				MaxBlockNumPerChunk:             tt.maxBlockNum,
				MaxTxNumPerChunk:                tt.maxTxNum,
				MaxL1CommitGasPerChunk:          tt.maxL1CommitGas,
				MaxL1CommitCalldataSizePerChunk: tt.maxL1CommitCalldataSize,
				MaxRowConsumptionPerChunk:       tt.maxRowConsumption,
				ChunkTimeoutSec:                 tt.chunkTimeoutSec,
				GasCostIncreaseMultiplier:       1.2,
				MaxUncompressedBatchBytesSize:   math.MaxUint64,
			}, &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64)}, db, nil)
			cp.TryProposeChunk()

			chunkOrm := orm.NewChunk(db)
			chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
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

func testChunkProposerBlobSizeLimit(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV0, encoding.CodecV1, encoding.CodecV2, encoding.CodecV3}
	for _, codecVersion := range codecVersions {
		db := setupDB(t)
		block := readBlockFromJSON(t, "../../../testdata/blockTrace_03.json")
		for i := int64(0); i < 510; i++ {
			l2BlockOrm := orm.NewL2Block(db)
			block.Header.Number = big.NewInt(i + 1)
			err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block})
			assert.NoError(t, err)
		}

		var chainConfig *params.ChainConfig
		if codecVersion == encoding.CodecV0 { // will never hit blob size limit
			chainConfig = &params.ChainConfig{}
		} else if codecVersion == encoding.CodecV1 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0)}
		} else if codecVersion == encoding.CodecV2 {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0)}
		} else {
			chainConfig = &params.ChainConfig{BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64)}
		}

		cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
			MaxBlockNumPerChunk:             255,
			MaxTxNumPerChunk:                math.MaxUint64,
			MaxL1CommitGasPerChunk:          math.MaxUint64,
			MaxL1CommitCalldataSizePerChunk: math.MaxUint64,
			MaxRowConsumptionPerChunk:       math.MaxUint64,
			ChunkTimeoutSec:                 math.MaxUint32,
			GasCostIncreaseMultiplier:       1,
			MaxUncompressedBatchBytesSize:   math.MaxUint64,
		}, chainConfig, db, nil)

		for i := 0; i < 2; i++ {
			cp.TryProposeChunk()
		}

		chunkOrm := orm.NewChunk(db)
		chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
		assert.NoError(t, err)

		var expectedNumChunks int = 2
		var numBlocksMultiplier uint64
		if codecVersion == encoding.CodecV0 {
			numBlocksMultiplier = 255
		} else if codecVersion == encoding.CodecV1 {
			numBlocksMultiplier = 22
		} else if codecVersion == encoding.CodecV2 {
			numBlocksMultiplier = 255
		} else {
			numBlocksMultiplier = 255
		}
		assert.Len(t, chunks, expectedNumChunks)

		for i, chunk := range chunks {
			expected := numBlocksMultiplier * (uint64(i) + 1)
			if expected > 2000 {
				expected = 2000
			}
			assert.Equal(t, expected, chunk.EndBlockNumber)
		}
		database.CloseDB(db)
	}
}

func testChunkProposerRespectHardforks(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	block := readBlockFromJSON(t, "../../../testdata/blockTrace_02.json")
	for i := int64(1); i <= 20; i++ {
		l2BlockOrm := orm.NewL2Block(db)
		block.Header.Number = big.NewInt(i)
		block.Header.Time = uint64(i)
		err := l2BlockOrm.InsertL2Blocks(context.Background(), []*encoding.Block{block})
		assert.NoError(t, err)
	}

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             math.MaxUint64,
		MaxTxNumPerChunk:                math.MaxUint64,
		MaxL1CommitGasPerChunk:          math.MaxUint64,
		MaxL1CommitCalldataSizePerChunk: math.MaxUint64,
		MaxRowConsumptionPerChunk:       math.MaxUint64,
		ChunkTimeoutSec:                 0,
		GasCostIncreaseMultiplier:       1,
		MaxUncompressedBatchBytesSize:   math.MaxUint64,
	}, &params.ChainConfig{
		BernoulliBlock: big.NewInt(1),
		CurieBlock:     big.NewInt(2),
		DarwinTime:     func() *uint64 { t := uint64(4); return &t }(),
	}, db, nil)

	for i := 0; i < 5; i++ {
		cp.TryProposeChunk()
	}

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
	assert.NoError(t, err)

	assert.Len(t, chunks, 4)
	expectedEndBlockNumbers := []uint64{1, 2, 3, 20}
	for i, chunk := range chunks {
		assert.Equal(t, expectedEndBlockNumbers[i], chunk.EndBlockNumber)
	}
}
