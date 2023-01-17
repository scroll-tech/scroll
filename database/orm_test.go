package database_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"scroll-tech/database/cache"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
)

var (
	templateL1Message = []*orm.L1Message{
		{
			Nonce:      1,
			MsgHash:    "msg_hash1",
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			Fee:        "0x19ece",
			GasLimit:   11529940,
			Deadline:   uint64(time.Now().Unix()),
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer1Hash: "hash0",
		},
		{
			Nonce:      2,
			MsgHash:    "msg_hash2",
			Height:     2,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			Fee:        "0x19ece",
			GasLimit:   11529940,
			Deadline:   uint64(time.Now().Unix()),
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer1Hash: "hash1",
		},
	}
	templateL2Message = []*orm.L2Message{
		{
			Nonce:      1,
			MsgHash:    "msg_hash1",
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			Fee:        "0x19ece",
			GasLimit:   11529940,
			Deadline:   uint64(time.Now().Unix()),
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash0",
		},
		{
			Nonce:      2,
			MsgHash:    "msg_hash2",
			Height:     2,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "0x19ece",
			Fee:        "0x19ece",
			GasLimit:   11529940,
			Deadline:   uint64(time.Now().Unix()),
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash1",
		},
	}
	blockTrace *types.BlockTrace

	dbConfig   *database.DBConfig
	dbImg      docker.ImgInstance
	redisImg   docker.ImgInstance
	ormBlock   orm.BlockTraceOrm
	ormLayer1  orm.L1MessageOrm
	ormLayer2  orm.L2MessageOrm
	ormBatch   orm.BlockBatchOrm
	ormSession orm.SessionInfoOrm
)

func setupEnv(t *testing.T) error {
	// Init db config and start db container.
	dbImg = docker.NewTestDBDocker(t, "postgres")
	redisImg = docker.NewTestRedisDocker(t)
	dbConfig = &database.DBConfig{
		DB: &database.PGConfig{
			DriverName: "postgres",
			DSN:        dbImg.Endpoint(),
		},
		RedisConfig: &cache.RedisConfig{
			RedisURL:    redisImg.Endpoint(),
			Expirations: map[string]int64{"trace": 30},
		},
	}

	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	db := factory.GetDB()
	assert.NoError(t, migrate.ResetDB(db.DB))

	// Init several orm handles.
	ormBlock = orm.BlockTraceOrm(factory)
	ormLayer1 = orm.L1MessageOrm(factory)
	ormLayer2 = orm.L2MessageOrm(factory)
	ormBatch = orm.BlockBatchOrm(factory)
	ormSession = orm.SessionInfoOrm(factory)

	templateBlockTrace, err := os.ReadFile("../common/testdata/blockTrace_03.json")
	if err != nil {
		return err
	}
	// unmarshal blockTrace
	blockTrace = &types.BlockTrace{}
	return json.Unmarshal(templateBlockTrace, blockTrace)
}

func freeDB(t *testing.T) {
	if dbImg != nil {
		assert.NoError(t, dbImg.Stop())
	}
	if redisImg != nil {
		assert.NoError(t, redisImg.Stop())
	}
}

// TestOrmFactory run several test cases.
func TestOrmFactory(t *testing.T) {
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}
	defer freeDB(t)

	t.Run("testOrmBlockTraces", testOrmBlockTraces)

	t.Run("testOrmL1Message", testOrmL1Message)

	t.Run("testOrmL2Message", testOrmL2Message)

	t.Run("testOrmBlockBatch", testOrmBlockBatch)

	t.Run("testOrmSessionInfo", testOrmSessionInfo)
}

func testOrmBlockTraces(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	res, err := ormBlock.GetBlockTraces(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 0)

	exist, err := ormBlock.Exist(blockTrace.Header.Number.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, false, exist)

	// Insert into db
	err = ormBlock.InsertBlockTraces([]*types.BlockTrace{blockTrace})
	assert.NoError(t, err)

	res2, err := ormBlock.GetUnbatchedBlocks(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res2) == 1)

	exist, err = ormBlock.Exist(blockTrace.Header.Number.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)

	res, err = ormBlock.GetBlockTraces(map[string]interface{}{
		"hash": blockTrace.Header.Hash().String(),
	})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 1)

	// Compare trace
	data1, err := json.Marshal(res[0])
	assert.NoError(t, err)
	data2, err := json.Marshal(blockTrace)
	assert.NoError(t, err)
	// check trace
	assert.Equal(t, true, string(data1) == string(data2))
}

func testOrmL1Message(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	expected := "expect hash"

	// Insert into db
	err = ormLayer1.SaveL1Messages(context.Background(), templateL1Message)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer1Status(context.Background(), "msg_hash1", orm.MsgConfirmed)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer1Status(context.Background(), "msg_hash2", orm.MsgSubmitted)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer2Hash(context.Background(), "msg_hash2", expected)
	assert.NoError(t, err)

	result, err := ormLayer1.GetL1ProcessedNonce()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	height, err := ormLayer1.GetLayer1LatestWatchedHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), height)

	msg, err := ormLayer1.GetL1MessageByMsgHash("msg_hash2")
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
}

func testOrmL2Message(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	expected := "expect hash"

	// Insert into db
	err = ormLayer2.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer2Status(context.Background(), "msg_hash1", orm.MsgConfirmed)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer2Status(context.Background(), "msg_hash2", orm.MsgSubmitted)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer1Hash(context.Background(), "msg_hash2", expected)
	assert.NoError(t, err)

	result, err := ormLayer2.GetL2ProcessedNonce()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	height, err := ormLayer2.GetLayer2LatestWatchedHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), height)

	msg, err := ormLayer2.GetL2MessageByMsgHash("msg_hash2")
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
	assert.Equal(t, msg.MsgHash, "msg_hash2")
}

// testOrmBlockBatch test rollup result table functions
func testOrmBlockBatch(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	dbTx, err := factory.Beginx()
	assert.NoError(t, err)
	batchID1, err := ormBatch.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: blockTrace.Header.Number.Uint64()},
		&orm.BlockInfo{Number: blockTrace.Header.Number.Uint64() + 1},
		"ff", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = ormBlock.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		blockTrace.Header.Number.Uint64(),
		blockTrace.Header.Number.Uint64() + 1}, batchID1)
	assert.NoError(t, err)
	batchID2, err := ormBatch.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: blockTrace.Header.Number.Uint64() + 2},
		&orm.BlockInfo{Number: blockTrace.Header.Number.Uint64() + 3},
		"ff", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = ormBlock.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		blockTrace.Header.Number.Uint64() + 2,
		blockTrace.Header.Number.Uint64() + 3}, batchID2)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	batches, err := ormBatch.GetBlockBatches(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, int(2), len(batches))

	batcheIDs, err := ormBatch.GetPendingBatches()
	assert.NoError(t, err)
	assert.Equal(t, int(2), len(batcheIDs))
	assert.Equal(t, batchID1, batcheIDs[0])
	assert.Equal(t, batchID2, batcheIDs[1])

	err = ormBatch.UpdateCommitTxHashAndRollupStatus(context.Background(), batchID1, "commit_tx_1", orm.RollupCommitted)
	assert.NoError(t, err)

	batcheIDs, err = ormBatch.GetPendingBatches()
	assert.NoError(t, err)
	assert.Equal(t, int(1), len(batcheIDs))
	assert.Equal(t, batchID2, batcheIDs[0])

	provingStatus, err := ormBatch.GetProvingStatusByID(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.ProvingTaskUnassigned, provingStatus)
	err = ormBatch.UpdateProofByID(context.Background(), batchID1, []byte{1}, []byte{2}, 1200)
	assert.NoError(t, err)
	err = ormBatch.UpdateProvingStatus(batchID1, orm.ProvingTaskVerified)
	assert.NoError(t, err)
	provingStatus, err = ormBatch.GetProvingStatusByID(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.ProvingTaskVerified, provingStatus)

	rollupStatus, err := ormBatch.GetRollupStatus(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupCommitted, rollupStatus)
	err = ormBatch.UpdateFinalizeTxHashAndRollupStatus(context.Background(), batchID1, "finalize_tx_1", orm.RollupFinalized)
	assert.NoError(t, err)
	rollupStatus, err = ormBatch.GetRollupStatus(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupFinalized, rollupStatus)
	result, err := ormBatch.GetLatestFinalizedBatch()
	assert.NoError(t, err)
	assert.Equal(t, batchID1, result.ID)

	status1, err := ormBatch.GetRollupStatus(batchID1)
	assert.NoError(t, err)
	status2, err := ormBatch.GetRollupStatus(batchID2)
	assert.NoError(t, err)
	assert.NotEqual(t, status1, status2)
	statues, err := ormBatch.GetRollupStatusByIDList([]string{batchID1, batchID2, batchID1, batchID2})
	assert.NoError(t, err)
	assert.Equal(t, statues[0], status1)
	assert.Equal(t, statues[1], status2)
	assert.Equal(t, statues[2], status1)
	assert.Equal(t, statues[3], status2)
	statues, err = ormBatch.GetRollupStatusByIDList([]string{batchID2, batchID1, batchID2, batchID1})
	assert.NoError(t, err)
	assert.Equal(t, statues[0], status2)
	assert.Equal(t, statues[1], status1)
	assert.Equal(t, statues[2], status2)
	assert.Equal(t, statues[3], status1)
}

// testOrmSessionInfo test rollup result table functions
func testOrmSessionInfo(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))
	dbTx, err := factory.Beginx()
	assert.NoError(t, err)
	batchID, err := ormBatch.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: blockTrace.Header.Number.Uint64()},
		&orm.BlockInfo{Number: blockTrace.Header.Number.Uint64() + 1},
		"ff", 1, 194676)
	assert.NoError(t, err)
	assert.NoError(t, ormBlock.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		blockTrace.Header.Number.Uint64(),
		blockTrace.Header.Number.Uint64() + 1}, batchID))
	assert.NoError(t, dbTx.Commit())
	assert.NoError(t, ormBatch.UpdateProvingStatus(batchID, orm.ProvingTaskAssigned))

	// empty
	ids, err := ormBatch.GetAssignedBatchIDs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(ids))
	sessionInfos, err := ormSession.GetSessionInfosByIDs(ids)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(sessionInfos))

	sessionInfo := orm.SessionInfo{
		ID: batchID,
		Rollers: map[string]*orm.RollerStatus{
			"0": {
				PublicKey: "0",
				Name:      "roller-0",
				Status:    orm.RollerAssigned,
			},
		},
		StartTimestamp: time.Now().Unix()}

	// insert
	assert.NoError(t, ormSession.SetSessionInfo(&sessionInfo))
	sessionInfos, err = ormSession.GetSessionInfosByIDs(ids)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionInfos))
	assert.Equal(t, sessionInfo, *sessionInfos[0])

	// update
	sessionInfo.Rollers["0"].Status = orm.RollerProofValid
	assert.NoError(t, ormSession.SetSessionInfo(&sessionInfo))
	sessionInfos, err = ormSession.GetSessionInfosByIDs(ids)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(sessionInfos))
	assert.Equal(t, sessionInfo, *sessionInfos[0])

	// delete
	assert.NoError(t, ormBatch.UpdateProvingStatus(batchID, orm.ProvingTaskVerified))
	ids, err = ormBatch.GetAssignedBatchIDs()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(ids))
	sessionInfos, err = ormSession.GetSessionInfosByIDs(ids)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(sessionInfos))
}
