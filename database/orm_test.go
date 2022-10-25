package database_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"

	docker_db "scroll-tech/database/docker"

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
	sqlxdb      *sqlx.DB
	ormTrace    orm.BlockResultOrm
	blockResult *types.BlockResult
	ormLayer1   orm.Layer1MessageOrm
	ormLayer2   orm.Layer2MessageOrm
	img         docker.ImgInstance
	ormRollup   orm.RollupResultOrm
)

func initEnv(t *testing.T) error {
	img = docker_db.NewImgDB(t, "postgres", "123456", "test", 5444)
	assert.NoError(t, img.Start())
	factory, err := database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        img.Endpoint(),
	})
	if err != nil {
		return err
	}
	db := factory.GetDB()
	sqlxdb = db
	ormTrace = orm.NewBlockResultOrm(db)
	ormLayer1 = orm.NewLayer1MessageOrm(db)
	ormLayer2 = orm.NewLayer2MessageOrm(db)
	ormRollup = orm.NewRollupResultOrm(db)

	// init db
	version := int64(0)
	err = migrate.Rollback(db.DB, &version)
	if err != nil {
		log.Error("failed to rollback in test db", "err", err)
		return err
	}

	err = migrate.Migrate(db.DB)
	if err != nil {
		log.Error("migrate failed in test db", "err", err)
		return err
	}

	templateBlockResult, err := os.ReadFile("../common/testdata/blockResult_orm.json")
	if err != nil {
		return err
	}

	// unmarshal blockResult
	if blockResult == nil {
		blockResult = &types.BlockResult{}
		return json.Unmarshal(templateBlockResult, blockResult)
	}
	return nil
}

func TestOrm_BlockResults(t *testing.T) {
	assert.NoError(t, initEnv(t))
	defer img.Stop()

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
	// TODO: fix this
	assert.NoError(t, ormTrace.UpdateProofByID(context.Background(), blockResult.BlockTrace.Number.ToInt().Uint64(), []byte{1}, []byte{2}, 1200))

	// Update trace and check result.
	err = ormTrace.UpdateBlockStatus(blockResult.BlockTrace.Number.ToInt().Uint64(), orm.BlockVerified)
	assert.NoError(t, err)
	res, err = ormTrace.GetBlockResults(map[string]interface{}{
		"status": orm.BlockVerified,
	})
	assert.NoError(t, err)
	assert.Equal(t, true, len(res) == 1 && res[0].BlockTrace.Hash.String() == blockResult.BlockTrace.Hash.String())

	defer sqlxdb.Close()
}

func TestOrm_Layer1Message(t *testing.T) {
	assert.NoError(t, initEnv(t))
	defer img.Stop()

	expected := "expect hash"

	// Insert into db
	err := ormLayer1.SaveLayer1Messages(context.Background(), templateLayer1Message)
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

	// todo : we should have a method to verify layer2hash in layer1message
	defer sqlxdb.Close()
}

func TestOrm_Layer2Message(t *testing.T) {
	assert.NoError(t, initEnv(t))
	defer img.Stop()

	expected := "expect hash"

	// Insert into db
	err := ormLayer2.SaveLayer2Messages(context.Background(), templateLayer2Message)
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

	// todo : we should have a method to verify layer1hash in layer1message
	defer sqlxdb.Close()
}

// TestOrm_RollupResult test rollup result table functions
func TestOrm_RollupResult(t *testing.T) {
	assert.NoError(t, initEnv(t))
	defer img.Stop()

	blocks := []uint64{uint64(templateRollup[0].Number), uint64(templateRollup[1].Number)}
	err := ormRollup.InsertPendingBatches(context.Background(), blocks)
	assert.NoError(t, err)

	err = ormRollup.UpdateFinalizeTxHash(context.Background(), uint64(templateRollup[0].Number), templateRollup[0].FinalizeTxHash)
	assert.NoError(t, err)

	err = ormRollup.UpdateRollupStatus(context.Background(), uint64(templateRollup[0].Number), orm.RollupPending)
	assert.NoError(t, err)

	err = ormRollup.UpdateFinalizeTxHashAndStatus(context.Background(), uint64(templateRollup[1].Number), templateRollup[1].FinalizeTxHash, templateRollup[1].Status)
	assert.NoError(t, err)

	results, err := ormRollup.GetPendingBatches()
	assert.NoError(t, err)
	assert.Equal(t, len(results), 1)

	// TODO: fix this
	result, err := ormRollup.GetLatestFinalizedBatch()
	assert.NoError(t, err)
	assert.Equal(t, templateRollup[1].Number, int(result))
}
