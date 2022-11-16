package database_test

import (
	"context"
	// "database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

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
	templateLayer1Message = []*orm.Layer1Message{
		{
			Nonce:      1,
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
	templateLayer2Message = []*orm.Layer2Message{
		{
			Nonce:      1,
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
	blockResult *types.BlockResult

	dbConfig  *database.DBConfig
	dbImg     docker.ImgInstance
	ormBlock  orm.BlockResultOrm
	ormLayer1 orm.Layer1MessageOrm
	ormLayer2 orm.Layer2MessageOrm
	ormBatch  orm.BlockBatchOrm
)

func setupEnv(t *testing.T) error {
	// Init db config and start db container.
	dbConfig = &database.DBConfig{DriverName: "postgres"}
	dbImg = docker.NewTestDBDocker(t, dbConfig.DriverName)
	dbConfig.DSN = dbImg.Endpoint()

	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	db := factory.GetDB()
	assert.NoError(t, migrate.ResetDB(db.DB))

	// Init several orm handles.
	ormBlock = orm.NewBlockResultOrm(db)
	ormLayer1 = orm.NewLayer1MessageOrm(db)
	ormLayer2 = orm.NewLayer2MessageOrm(db)
	ormBatch = orm.NewBlockBatchOrm(db)

	templateBlockResult, err := os.ReadFile("../common/testdata/blockResult_orm.json")
	if err != nil {
		return err
	}
	// unmarshal blockResult
	blockResult = &types.BlockResult{}
	return json.Unmarshal(templateBlockResult, blockResult)
}

// TestOrmFactory run several test cases.
func TestOrmFactory(t *testing.T) {
	defer func() {
		if dbImg != nil {
			assert.NoError(t, dbImg.Stop())
		}
	}()
	if err := setupEnv(t); err != nil {
		t.Fatal(err)
	}

	t.Run("testOrmBlockResults", testOrmBlockResults)

	t.Run("testOrmLayer1Message", testOrmLayer1Message)

	t.Run("testOrmLayer2Message", testOrmLayer2Message)

	t.Run("testOrmBlockbatch", testOrmBlockbatch)
}

func testOrmBlockResults(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	res, err := ormBlock.GetBlockResults(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 0)

	// Insert into db
	err = ormBlock.InsertBlockResults(context.Background(), []*types.BlockResult{blockResult})
	assert.NoError(t, err)

	exist, err := ormBlock.Exist(blockResult.BlockTrace.Number.ToInt().Uint64())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)

	res, err = ormBlock.GetBlockResults(map[string]interface{}{
		"hash": blockResult.BlockTrace.Hash.String(),
	})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 1)

	// Compare trace
	data1, err := json.Marshal(res[0])
	assert.NoError(t, err)
	data2, err := json.Marshal(blockResult)
	assert.NoError(t, err)
	// check trace
	assert.Equal(t, true, string(data1) == string(data2))
}

func testOrmLayer1Message(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	expected := "expect hash"

	// Insert into db
	err = ormLayer1.SaveLayer1Messages(context.Background(), templateLayer1Message)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer1Status(context.Background(), "hash0", orm.MsgConfirmed)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer1Status(context.Background(), "hash1", orm.MsgSubmitted)
	assert.NoError(t, err)

	err = ormLayer1.UpdateLayer2Hash(context.Background(), "hash1", expected)
	assert.NoError(t, err)

	result, err := ormLayer1.GetL1ProcessedNonce()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	height, err := ormLayer1.GetLayer1LatestWatchedHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), height)

	msg, err := ormLayer1.GetLayer1MessageByLayer1Hash("hash1")
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
}

func testOrmLayer2Message(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	expected := "expect hash"

	// Insert into db
	err = ormLayer2.SaveLayer2Messages(context.Background(), templateLayer2Message)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer2Status(context.Background(), "hash0", orm.MsgConfirmed)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer2Status(context.Background(), "hash1", orm.MsgSubmitted)
	assert.NoError(t, err)

	err = ormLayer2.UpdateLayer1Hash(context.Background(), "hash1", expected)
	assert.NoError(t, err)

	result, err := ormLayer2.GetL2ProcessedNonce()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	height, err := ormLayer2.GetLayer2LatestWatchedHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), height)

	msg, err := ormLayer2.GetLayer2MessageByLayer2Hash("hash1")
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
}

// testOrmBlockbatch test rollup result table functions
func testOrmBlockbatch(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	dbTx, err := factory.Beginx()
	assert.NoError(t, err)
	batchID1, err := ormBatch.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: blockResult.BlockTrace.Number.ToInt().Uint64()},
		&orm.BlockInfo{Number: blockResult.BlockTrace.Number.ToInt().Uint64() + 1},
		"ff", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = ormBlock.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		blockResult.BlockTrace.Number.ToInt().Uint64(),
		blockResult.BlockTrace.Number.ToInt().Uint64() + 1}, batchID1)
	assert.NoError(t, err)
	batchID2, err := ormBatch.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: blockResult.BlockTrace.Number.ToInt().Uint64() + 2},
		&orm.BlockInfo{Number: blockResult.BlockTrace.Number.ToInt().Uint64() + 3},
		"ff", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = ormBlock.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		blockResult.BlockTrace.Number.ToInt().Uint64() + 2,
		blockResult.BlockTrace.Number.ToInt().Uint64() + 3}, batchID2)
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

	proving_status, err := ormBatch.GetProvingStatusByID(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.ProvingTaskUnassigned, proving_status)
	err = ormBatch.UpdateProofByID(context.Background(), batchID1, []byte{1}, []byte{2}, 1200)
	assert.NoError(t, err)
	err = ormBatch.UpdateProvingStatus(batchID1, orm.ProvingTaskVerified)
	assert.NoError(t, err)
	proving_status, err = ormBatch.GetProvingStatusByID(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.ProvingTaskVerified, proving_status)

	rollup_status, err := ormBatch.GetRollupStatus(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupCommitted, rollup_status)
	err = ormBatch.UpdateFinalizeTxHashAndRollupStatus(context.Background(), batchID1, "finalize_tx_1", orm.RollupFinalized)
	assert.NoError(t, err)
	rollup_status, err = ormBatch.GetRollupStatus(batchID1)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupFinalized, rollup_status)
	result, err := ormBatch.GetLatestFinalizedBatch()
	assert.NoError(t, err)
	assert.Equal(t, batchID1, result.ID)
}
