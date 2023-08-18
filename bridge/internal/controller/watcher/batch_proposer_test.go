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
func testBatchProposer(t *testing.T) {
	db := setupDB(t)
	defer database.CloseDB(db)

	l2BlockOrm := orm.NewL2Block(db)
	err := l2BlockOrm.InsertL2Blocks(context.Background(), []*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	cp := NewChunkProposer(context.Background(), &config.ChunkProposerConfig{
		MaxTxGasPerChunk:                1000000000,
		MaxL2TxNumPerChunk:              10000,
		MaxL1CommitGasPerChunk:          50000000000,
		MaxL1CommitCalldataSizePerChunk: 1000000,
		MaxRowConsumptionPerChunk:       1048319,
		ChunkTimeoutSec:                 0,
	}, db, nil)
	cp.TryProposeChunk()

	bp := NewBatchProposer(context.Background(), &config.BatchProposerConfig{
		MaxChunkNumPerBatch:             10,
		MaxL1CommitGasPerBatch:          50000000000,
		MaxL1CommitCalldataSizePerBatch: 1000000,
		BatchTimeoutSec:                 0,
	}, db, nil)
	bp.TryProposeBatch()

	chunkOrm := orm.NewChunk(db)
	chunks, err := chunkOrm.GetUnbatchedChunks(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, chunks)

	batchOrm := orm.NewBatch(db)
	// get all batches.
	batches, err := batchOrm.GetBatches(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, batches, 1)
	assert.Equal(t, uint64(0), batches[0].StartChunkIndex)
	assert.Equal(t, uint64(0), batches[0].EndChunkIndex)
	assert.Equal(t, types.RollupPending, types.RollupStatus(batches[0].RollupStatus))
	assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(batches[0].ProvingStatus))

	dbChunks, err := chunkOrm.GetChunksInRange(context.Background(), 0, 0)
	assert.NoError(t, err)
	assert.Len(t, batches, 1)
	assert.Equal(t, batches[0].Hash, dbChunks[0].BatchHash)
	assert.Equal(t, types.ProvingTaskUnassigned, types.ProvingStatus(dbChunks[0].ProvingStatus))

	blockOrm := orm.NewL2Block(db)
	blocks, err := blockOrm.GetL2Blocks(context.Background(), map[string]interface{}{}, []string{}, 0)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, dbChunks[0].Hash, blocks[0].ChunkHash)
	assert.Equal(t, dbChunks[0].Hash, blocks[1].ChunkHash)
}
