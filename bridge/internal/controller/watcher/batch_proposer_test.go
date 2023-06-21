package watcher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
)

func testBatchProposer(t *testing.T) {
	db := setupDB(t)
	defer utils.CloseDB(db)

	l2BlockOrm := orm.NewL2Block(db)
	err := l2BlockOrm.InsertL2Blocks([]*bridgeTypes.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxL2TxGasPerChunk:              1000000000,
		MaxL2TxNumPerChunk:              10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MinL1CommitCalldataSizePerChunk: 0,
		ChunkTimeoutSec:                 300,
	}, db)
	cp.TryProposeChunk()

	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunkNumPerBatch:             10,
		MaxL1CommitGasPerBatch:          50000000000,
		MaxL1CommitCalldataSizePerBatch: 1000000,
		MinChunkNumPerBatch:             1,
		BatchTimeoutSec:                 300,
	}, db)
	bp.TryProposeBatch()

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetUnbatchedChunks(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, chunks)

	batchOrm := orm.NewBatch(db)
	batch, err := batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), batch.StartChunkIndex)
	assert.Equal(t, uint64(0), batch.EndChunkIndex)
}
