package watcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/database"
	"scroll-tech/common/types"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
)

// TODO: Add unit tests that the limits are enforced correctly.
func testChunkProposer(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	l2BlockOrm := orm.NewL2Block(db)
	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             100,
		MaxTxNumPerChunk:                10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MaxRowConsumptionPerChunk:       1048319,
		ChunkTimeoutSec:                 300,
	}, db, nil)
	cp.TryProposeChunk()

	expectedChunk := &types.Chunk{
		Blocks: []*types.WrappedBlock{wrappedBlock1, wrappedBlock2},
	}
	expectedHash, err := expectedChunk.Hash(0)
	assert.NoError(t, err)

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)
	assert.Equal(t, expectedHash.Hex(), chunks[0].Hash)
}

func testChunkProposerRowConsumption(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	l2BlockOrm := orm.NewL2Block(db)
	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxBlockNumPerChunk:             100,
		MaxTxNumPerChunk:                10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MaxRowConsumptionPerChunk:       0, // !
		ChunkTimeoutSec:                 300,
	}, db, nil)
	cp.TryProposeChunk()

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetChunksGEIndex(context.Background(), 0, 0)
	assert.NoError(t, err)
	assert.Len(t, chunks, 0)
}
