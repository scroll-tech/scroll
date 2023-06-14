package orm

import (
	"context"
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

	db           *gorm.DB
	batchOrm     *Batch
	l1MessageOrm *L1Message
	l2MessageOrm *L2Message
	l1BlockOrm   *L1Block
	l2BlockOrm   *L2Block

	wrappedBlock1 *types.WrappedBlock
	wrappedBlock2 *types.WrappedBlock
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
	l1MessageOrm = NewL1Message(db)
	l2MessageOrm = NewL2Message(db)
	l1BlockOrm = NewL1Block(db)
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
}

func TestL2Block(t *testing.T) {
	testBlocks := []*types.WrappedBlock{wrappedBlock1, wrappedBlock2}
	err := l2BlockOrm.InsertL2Blocks(testBlocks)
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

	err = l2BlockOrm.UpdateChunkHashForL2Blocks([]uint64{2}, "test_hash")
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
	assert.Equal(t, "test_hash", blockInfos[0].ChunkHash)

	err = l2BlockOrm.UpdateBatchIndexForL2Blocks([]uint64{2, 3}, 1)
	assert.NoError(t, err)
	blockInfos, err = l2BlockOrm.GetL2Blocks(map[string]interface{}{"batch_index": 1}, nil, 0)
	assert.NoError(t, err)
	assert.Len(t, blockInfos, 2)
}
