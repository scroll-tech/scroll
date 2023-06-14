package orm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/bridge/internal/config"
	"scroll-tech/bridge/internal/orm/migrate"
	"scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
	"scroll-tech/common/docker"
)

var (
	base *docker.App

	db         *gorm.DB
	l2BlockOrm *L2Block
	chunkOrm   *Chunk
	batchOrm   *Batch

	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock
	chunk1        *types.Chunk
	chunk2        *types.Chunk
	chunkHash1    string
	chunkHash2    string
)

func TestMain(m *testing.M) {
	setupEnv(&testing.T{})
	os.Exit(m.Run())
}

func setupEnv(t *testing.T) {
	base = docker.NewDockerApp()
	base.RunDBImage(t)
	var err error
	db, err = utils.InitDB(
		&config.DBConfig{
			DSN:        base.DBConfig.DSN,
			DriverName: base.DBConfig.DriverName,
			MaxOpenNum: base.DBConfig.MaxOpenNum,
			MaxIdleNum: base.DBConfig.MaxIdleNum,
		},
	)
	assert.NoError(t, err)
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	batchOrm = NewBatch(db)
	chunkOrm = NewChunk(db)
	l2BlockOrm = NewL2Block(db)

	templateBlockTrace, err := os.ReadFile("../../../common/testdata/blockTrace_02.json")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	wrappedBlock1 = &types.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace, wrappedBlock1); err != nil {
		t.Fatalf("failed to unmarshal block trace: %v", err)
	}

	templateBlockTrace, err = os.ReadFile("../../../common/testdata/blockTrace_03.json")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	wrappedBlock2 = &types.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace, wrappedBlock2); err != nil {
		t.Fatalf("failed to unmarshal block trace: %v", err)
	}

	chunk1 = &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock1}}
	chunkHashBytes1, err := chunk1.Hash()
	assert.NoError(t, err)
	chunkHash1 = hex.EncodeToString(chunkHashBytes1)

	chunk2 = &types.Chunk{Blocks: []*types.WrappedBlock{wrappedBlock2}}
	chunkHashBytes2, err := chunk2.Hash()
	assert.NoError(t, err)
	chunkHash2 = hex.EncodeToString(chunkHashBytes2)
}

func TestL2BlockOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	err = l2BlockOrm.InsertL2Blocks([]*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	if err != nil {
		t.Fatalf("failed to insert blocks: %v", err)
	}

	height, err := l2BlockOrm.GetL2BlocksLatestHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), height)

	blocks, err := l2BlockOrm.GetUnchunkedBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, wrappedBlock1, blocks[0])
	assert.Equal(t, wrappedBlock2, blocks[1])

	blocks, err = l2BlockOrm.RangeGetL2Blocks(context.Background(), 2, 3)
	assert.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, wrappedBlock1, blocks[0])
	assert.Equal(t, wrappedBlock2, blocks[1])

	err = l2BlockOrm.UpdateChunkHashForL2Blocks([]uint64{2}, "test hash")
	assert.NoError(t, err)
	err = l2BlockOrm.UpdateBatchIndexForL2Blocks([]uint64{2, 3}, 1)
	assert.NoError(t, err)

	blocks, err = l2BlockOrm.GetUnchunkedBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, wrappedBlock2, blocks[0])
	blockInfos, err := l2BlockOrm.GetL2Blocks(map[string]interface{}{"number": 2}, nil, 0)
	assert.NoError(t, err)
	assert.Len(t, blockInfos, 1)
	assert.Equal(t, "test hash", blockInfos[0].ChunkHash)

	err = l2BlockOrm.UpdateBatchIndexForL2Blocks([]uint64{2, 3}, 1)
	assert.NoError(t, err)
	blockInfos, err = l2BlockOrm.GetL2Blocks(map[string]interface{}{"batch_index": 1}, nil, 0)
	assert.NoError(t, err)
	assert.Len(t, blockInfos, 2)
}

func TestChunkOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	err = l2BlockOrm.InsertL2Blocks([]*types.WrappedBlock{wrappedBlock1, wrappedBlock2})
	if err != nil {
		t.Fatalf("failed to insert blocks: %v", err)
	}

	err = chunkOrm.InsertChunk(context.Background(), chunk1, l2BlockOrm)
	assert.NoError(t, err)
	blocks, err := l2BlockOrm.GetUnchunkedBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, wrappedBlock2, blocks[0])

	err = chunkOrm.InsertChunk(context.Background(), chunk2, l2BlockOrm)
	blocks, err = l2BlockOrm.GetUnchunkedBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 0)

	chunks, err := chunkOrm.GetUnbatchedChunks(context.Background())
	assert.NoError(t, err)
	assert.Len(t, chunks, 2)
	assert.Equal(t, chunkHash1, chunks[0].Hash)
	assert.Equal(t, chunkHash2, chunks[1].Hash)

	chunks, err = chunkOrm.RangeGetChunks(context.Background(), 0, 1)
	assert.NoError(t, err)
	assert.Len(t, chunks, 2)
	assert.Equal(t, chunkHash1, chunks[0].Hash)
	assert.Equal(t, chunkHash2, chunks[1].Hash)

	err = chunkOrm.UpdateChunk(context.Background(), chunkHash1, map[string]interface{}{"batch_hash": "hash"})
	assert.NoError(t, err)
	chunks, err = chunkOrm.GetUnbatchedChunks(context.Background())
	assert.NoError(t, err)
	assert.Len(t, chunks, 1)
	err = chunkOrm.UpdateBatchHashForChunks([]string{chunkHash1}, "", chunkOrm.db)
	assert.NoError(t, err)
	chunks, err = chunkOrm.GetUnbatchedChunks(context.Background())
	assert.Len(t, chunks, 2)
}

func TestBatchOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	err = batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk1}, chunkOrm, l2BlockOrm)
	assert.NoError(t, err)

	err = batchOrm.InsertBatch(context.Background(), []*types.Chunk{chunk2}, chunkOrm, l2BlockOrm)
	assert.NoError(t, err)

	count, err := batchOrm.GetBatchCount(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
