package database_test

import (
	"context"
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
	templateRollup = []*orm.RollupResult{
		{
			Number:         1,
			Status:         orm.RollupPending,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "",
		},
		{
			Number:         2,
			Status:         orm.RollupFinalized,
			RollupTxHash:   "Rollup Test Hash",
			FinalizeTxHash: "",
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
	ormTrace  orm.BlockResultOrm
	ormLayer1 orm.Layer1MessageOrm
	ormLayer2 orm.Layer2MessageOrm
	ormRollup orm.RollupResultOrm
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
	ormTrace = orm.NewBlockResultOrm(db)
	ormLayer1 = orm.NewLayer1MessageOrm(db)
	ormLayer2 = orm.NewLayer2MessageOrm(db)
	ormRollup = orm.NewRollupResultOrm(db)

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

	t.Run("testOrm_BlockResults", testOrmBlockResults)

	t.Run("testOrmLayer1Message", testOrmLayer1Message)

	t.Run("testOrmLayer2Message", testOrmLayer2Message)

	t.Run("testOrmRollupResult", testOrmRollupResult)
}

func testOrmBlockResults(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	res, err := ormTrace.GetBlockResults(map[string]interface{}{})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 0)

	// Insert into db
	err = ormTrace.InsertBlockResultsWithStatus(context.Background(), []*types.BlockResult{blockResult}, orm.BlockUnassigned)
	assert.NoError(t, err)

	exist, err := ormTrace.Exist(blockResult.BlockTrace.Number.ToInt().Uint64())
	assert.NoError(t, err)
	assert.Equal(t, true, exist)

	res, err = ormTrace.GetBlockResults(map[string]interface{}{
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

	// Update proof by hash
	assert.NoError(t, ormTrace.UpdateProofByNumber(context.Background(), blockResult.BlockTrace.Number.ToInt().Uint64(), []byte{1}, []byte{2}, 1200))

	// Update trace and check result.
	err = ormTrace.UpdateBlockStatus(blockResult.BlockTrace.Number.ToInt().Uint64(), orm.BlockVerified)
	assert.NoError(t, err)
	res, err = ormTrace.GetBlockResults(map[string]interface{}{
		"status": orm.BlockVerified,
	})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 1 && res[0].BlockTrace.Hash.String() == blockResult.BlockTrace.Hash.String())
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

// testOrmRollupResult test rollup result table functions
func testOrmRollupResult(t *testing.T) {
	// Create db handler and reset db.
	factory, err := database.NewOrmFactory(dbConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(factory.GetDB().DB))

	blocks := []uint64{uint64(templateRollup[0].Number), uint64(templateRollup[1].Number)}
	err = ormRollup.InsertPendingBlocks(context.Background(), blocks)
	assert.NoError(t, err)

	err = ormRollup.UpdateFinalizeTxHash(context.Background(), uint64(templateRollup[0].Number), templateRollup[0].FinalizeTxHash)
	assert.NoError(t, err)

	err = ormRollup.UpdateRollupStatus(context.Background(), uint64(templateRollup[0].Number), orm.RollupPending)
	assert.NoError(t, err)

	err = ormRollup.UpdateFinalizeTxHashAndStatus(context.Background(), uint64(templateRollup[1].Number), templateRollup[1].FinalizeTxHash, templateRollup[1].Status)
	assert.NoError(t, err)

	results, err := ormRollup.GetPendingBlocks()
	assert.NoError(t, err)
	assert.Equal(t, len(results), 1)

	result, err := ormRollup.GetLatestFinalizedBlock()
	assert.NoError(t, err)
	assert.Equal(t, templateRollup[1].Number, int(result))
}
