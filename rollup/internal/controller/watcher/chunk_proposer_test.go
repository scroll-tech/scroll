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

func testChunkProposerLimitsCodecV4(t *testing.T) {
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
			}, encoding.CodecV4, &params.ChainConfig{LondonBlock: big.NewInt(0), BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64), DarwinV2Time: new(uint64)}, db, nil)
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

func testChunkProposerBlobSizeLimitCodecV4(t *testing.T) {
	codecVersions := []encoding.CodecVersion{encoding.CodecV4}
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
		if codecVersion == encoding.CodecV4 {
			chainConfig = &params.ChainConfig{LondonBlock: big.NewInt(0), BernoulliBlock: big.NewInt(0), CurieBlock: big.NewInt(0), DarwinTime: new(uint64), DarwinV2Time: new(uint64)}
		} else {
			assert.Fail(t, "unsupported codec version, expected CodecV4")
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
		}, encoding.CodecV4, chainConfig, db, nil)

		for i := 0; i < 2; i++ {
			cp.TryProposeChunk()
		}

		chunkOrm := orm.NewChunk(db)
		chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
		assert.NoError(t, err)

		var expectedNumChunks int = 2
		var numBlocksMultiplier uint64
		if codecVersion == encoding.CodecV4 {
			numBlocksMultiplier = 255
		} else {
			assert.Fail(t, "unsupported codec version, expected CodecV4")
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
