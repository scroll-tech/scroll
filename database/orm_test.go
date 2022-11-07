package database_test

import (
	"context"
	"database/sql"
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
	templateBatch = []*orm.BlockBatch{
		{
			ID:             "1",
			RollupStatus:   orm.RollupPending,
			CommitTxHash:   sql.NullString{Valid: false},
			FinalizeTxHash: sql.NullString{Valid: false},
		},
		{
			ID:             "2",
			RollupStatus:   orm.RollupFinalized,
			CommitTxHash:   sql.NullString{String: "Committed Hash", Valid: true},
			FinalizeTxHash: sql.NullString{String: "Finalized Hash", Valid: true},
		},
	}
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

	// Compare content
	data1, err := json.Marshal(res[0])
	assert.NoError(t, err)
	data2, err := json.Marshal(blockResult)
	assert.NoError(t, err)
	// check content
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
	t.Skip()

	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	// blocks := []uint64{uint64(templateBatch[0].Number), uint64(templateBatch[1].Number)}
	// err = ormBatch.InsertPendingBatches(context.Background(), blocks)
	// assert.NoError(t, err)

	// err = ormBatch.UpdateFinalizeTxHash(context.Background(), templateBatch[0].ID, templateBatch[0].FinalizeTxHash)
	// assert.NoError(t, err)

	err = ormBatch.UpdateRollupStatus(context.Background(), templateBatch[0].ID, orm.RollupPending)
	assert.NoError(t, err)

	err = ormBatch.UpdateFinalizeTxHashAndRollupStatus(context.Background(), templateBatch[1].ID, templateBatch[1].FinalizeTxHash.String, templateBatch[1].RollupStatus)
	assert.NoError(t, err)

	results, err := ormBatch.GetPendingBatches()
	assert.NoError(t, err)
	assert.Equal(t, len(results), 1)
	assert.Equal(t, templateBatch[0].ID, results[0])

	result, err := ormBatch.GetLatestFinalizedBatch()
	assert.NoError(t, err)
	assert.Equal(t, len(results), 1)
	assert.Equal(t, templateBatch[1].ID, result.ID)

	// // Update trace and check result.
	// err = ormBlock.UpdateBlockStatus(blockResult.BlockTrace.Number.ToInt().Uint64(), orm.BlockVerified)
	// assert.NoError(t, err)
	// res, err = ormBlock.GetBlockResults(map[string]interface{}{
	// 	"status": orm.BlockVerified,
	// })
	// assert.NoError(t, err)

	// // Update proof by hashs
	// assert.NoError(t, ormBlock.UpdateProofByID(context.Background(), blockResult.BlockTrace.Number.ToInt().Uint64(), []byte{1}, []byte{2}, 1200))
}
