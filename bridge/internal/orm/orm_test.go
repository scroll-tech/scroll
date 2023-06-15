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
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
	"scroll-tech/common/docker"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

var (
	base *docker.App

	db         *gorm.DB
	l2BlockOrm *L2Block
	chunkOrm   *Chunk
	batchOrm   *Batch

	wrappedBlock1 *bridgeTypes.WrappedBlock
	wrappedBlock2 *bridgeTypes.WrappedBlock
	chunk1        *bridgeTypes.Chunk
	chunk2        *bridgeTypes.Chunk
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
	wrappedBlock1 = &bridgeTypes.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace, wrappedBlock1); err != nil {
		t.Fatalf("failed to unmarshal block trace: %v", err)
	}

	templateBlockTrace, err = os.ReadFile("../../../common/testdata/blockTrace_03.json")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	wrappedBlock2 = &bridgeTypes.WrappedBlock{}
	if err = json.Unmarshal(templateBlockTrace, wrappedBlock2); err != nil {
		t.Fatalf("failed to unmarshal block trace: %v", err)
	}

	chunk1 = &bridgeTypes.Chunk{Blocks: []*bridgeTypes.WrappedBlock{wrappedBlock1}}
	chunkHashBytes1, err := chunk1.Hash()
	assert.NoError(t, err)
	chunkHash1 = hex.EncodeToString(chunkHashBytes1)

	chunk2 = &bridgeTypes.Chunk{Blocks: []*bridgeTypes.WrappedBlock{wrappedBlock2}}
	chunkHashBytes2, err := chunk2.Hash()
	assert.NoError(t, err)
	chunkHash2 = hex.EncodeToString(chunkHashBytes2)
}

func TestL2BlockOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	err = l2BlockOrm.InsertL2Blocks([]*bridgeTypes.WrappedBlock{wrappedBlock1, wrappedBlock2})
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

	blocks, err = l2BlockOrm.GetUnchunkedBlocks()
	assert.NoError(t, err)
	assert.Len(t, blocks, 1)
	assert.Equal(t, wrappedBlock2, blocks[0])
	blockInfos, err := l2BlockOrm.GetL2Blocks(map[string]interface{}{"number": 2}, nil, 0)
	assert.NoError(t, err)
	assert.Len(t, blockInfos, 1)
	assert.Equal(t, "test hash", blockInfos[0].ChunkHash)
}

func TestChunkOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	err = l2BlockOrm.InsertL2Blocks([]*bridgeTypes.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

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
	err = chunkOrm.UpdateBatchHashForChunks([]string{chunkHash2}, "", chunkOrm.db)
	assert.NoError(t, err)
	chunks, err = chunkOrm.GetUnbatchedChunks(context.Background())
	assert.Len(t, chunks, 0)
}

func TestBatchOrm(t *testing.T) {
	sqlDB, err := db.DB()
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(sqlDB))

	err = l2BlockOrm.InsertL2Blocks([]*bridgeTypes.WrappedBlock{wrappedBlock1, wrappedBlock2})
	assert.NoError(t, err)

	err = chunkOrm.InsertChunk(context.Background(), chunk1, l2BlockOrm)
	assert.NoError(t, err)

	err = chunkOrm.InsertChunk(context.Background(), chunk2, l2BlockOrm)
	assert.NoError(t, err)

	err = batchOrm.InsertBatch(context.Background(), []*bridgeTypes.Chunk{chunk1}, chunkOrm)
	assert.NoError(t, err)

	err = batchOrm.InsertBatch(context.Background(), []*bridgeTypes.Chunk{chunk2}, chunkOrm)
	assert.NoError(t, err)

	count, err := batchOrm.GetBatchCount(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	latestBatch, err := batchOrm.GetLatestBatch(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), latestBatch.Index)

	pendingBatches, err := batchOrm.GetPendingBatches(context.Background(), 100)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pendingBatches))

	batchHeader1, err := batchOrm.GetBatchHeader(context.Background(), 0, chunkOrm, l2BlockOrm)
	assert.NoError(t, err)
	batchHash1 := batchHeader1.Hash().Hex()

	batchHeader2, err := batchOrm.GetBatchHeader(context.Background(), 1, chunkOrm, l2BlockOrm)
	assert.NoError(t, err)
	batchHash2 := batchHeader2.Hash().Hex()

	rollupStatus, err := batchOrm.GetRollupStatusByHashList(context.Background(), []string{batchHash1, batchHash2})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(rollupStatus))
	assert.Equal(t, types.RollupPending, rollupStatus[0])
	assert.Equal(t, types.RollupPending, rollupStatus[1])

	err = batchOrm.UpdateBatch(context.Background(), batchHash1, map[string]interface{}{
		"rollup_status":  types.RollupCommitted,
		"proving_status": types.ProvingTaskSkipped,
	})
	assert.NoError(t, err)

	err = batchOrm.UpdateBatch(context.Background(), batchHash2, map[string]interface{}{
		"rollup_status":  types.RollupCommitted,
		"proving_status": types.ProvingTaskFailed,
	})
	assert.NoError(t, err)

	count, err = batchOrm.UpdateSkippedBatches(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	count, err = batchOrm.UpdateSkippedBatches(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	latestBatch, err = batchOrm.GetLatestBatchByRollupStatus([]types.RollupStatus{types.RollupFinalizationSkipped})
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), latestBatch.Index)

	proof := &message.AggProof{
		Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = batchOrm.UpdateProofByHash(context.Background(), batchHash1, proof, 1200)
	assert.NoError(t, err)
	err = batchOrm.UpdateBatch(context.Background(), batchHash1, map[string]interface{}{
		"proving_status": types.ProvingTaskVerified,
	})
	assert.NoError(t, err)

	dbProof, err := batchOrm.GetVerifiedProofByHash(context.Background(), batchHash1)
	assert.NoError(t, err)
	assert.Equal(t, proof, dbProof)
}
