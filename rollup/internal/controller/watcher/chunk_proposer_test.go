package watcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

func testChunkProposerLimits(t *testing.T) {
	tests := []struct {
		name                    string
		maxBlockNum             uint64
		maxTxNum                uint64
		maxL1CommitGas          uint64
		maxL1CommitCalldataSize uint64
		maxRowConsumption       uint64
		expectedChunksLen       int
		expectedBlocksNum       uint64
	}{
		{
			name:                    "Timeout",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			expectedChunksLen:       1,
			expectedBlocksNum:       2,
		},
		{
			name:                    "MaxTxNumPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                0,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitGasPerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          0,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxL1CommitCalldataSizePerChunkIs0",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 0,
			maxRowConsumption:       1000000,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxRowConsumptionPerChunkIs0",
			maxBlockNum:             100,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       0,
			expectedChunksLen:       0,
		},
		{
			name:                    "MaxBlockNumPerChunkIs1",
			maxBlockNum:             1,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			expectedChunksLen:       1,
			expectedBlocksNum:       1,
		},
		{
			name:                    "MaxTxNumPerChunkIsFirstBlock",
			maxBlockNum:             10,
			maxTxNum:                2,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			expectedChunksLen:       1,
			expectedBlocksNum:       1,
		},
		{
			name:                    "MaxL1CommitGasPerChunkIsFirstBlock",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          60,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1000000,
			expectedChunksLen:       1,
			expectedBlocksNum:       1,
		},
		{
			name:                    "MaxL1CommitCalldataSizePerChunkIsFirstBlock",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 298,
			maxRowConsumption:       1000000,
			expectedChunksLen:       1,
			expectedBlocksNum:       1,
		},
		{
			name:                    "MaxRowConsumptionPerChunkIs1",
			maxBlockNum:             10,
			maxTxNum:                10000,
			maxL1CommitGas:          50000000000,
			maxL1CommitCalldataSize: 1000000,
			maxRowConsumption:       1,
			expectedChunksLen:       1,
			expectedBlocksNum:       1,
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
				MaxBlockNumPerChunk:             tt.maxBlockNum,
				MaxTxNumPerChunk:                tt.maxTxNum,
				MaxL1CommitGasPerChunk:          tt.maxL1CommitGas,
				MaxL1CommitCalldataSizePerChunk: tt.maxL1CommitCalldataSize,
				MaxRowConsumptionPerChunk:       tt.maxRowConsumption,
				ChunkTimeoutSec:                 300,
				GasCostIncreaseMultiplier:       1.2,
			}, db, nil)
			cp.TryProposeChunk()

			chunkOrm := orm.NewChunk(db)
			chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
			assert.NoError(t, err)
			assert.Len(t, chunks, tt.expectedChunksLen)

			if len(chunks) > 0 {
				var expectedChunk types.Chunk

				switch tt.expectedBlocksNum {
				case 1:
					expectedChunk.Blocks = []*types.WrappedBlock{wrappedBlock1}
				case 2:
					expectedChunk.Blocks = []*types.WrappedBlock{wrappedBlock1, wrappedBlock2}
				}

				expectedHash, err := expectedChunk.Hash(0)
				assert.NoError(t, err)
				assert.Equal(t, expectedHash.Hex(), chunks[0].Hash)
			}
		})
	}
}
