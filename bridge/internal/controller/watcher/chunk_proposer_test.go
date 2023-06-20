package watcher

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
)

func testChunkProposer(t *testing.T) {
	db := setupDB(t)
	defer utils.CloseDB(db)

	l2BlockOrm := orm.NewL2Block(db)
	err := l2BlockOrm.InsertL2Blocks([]*bridgeTypes.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	p := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxL2TxGasPerChunk:              1000000000,
		MaxL2TxNumPerChunk:              10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MinL1CommitCalldataSizePerChunk: 0,
		ChunkTimeoutSec:                 300,
	}, db)
	p.TryProposeChunk()

	expectedChunk := &bridgeTypes.Chunk{
		Blocks: []*bridgeTypes.WrappedBlock{wrappedBlock1, wrappedBlock2},
	}
	hashBytes, err := expectedChunk.Hash(0)
	assert.NoError(t, err)
	expectedHash := hex.EncodeToString(hashBytes)

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetUnbatchedChunks(context.Background())
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)
	assert.Equal(t, expectedHash, chunks[0].Hash)
}
