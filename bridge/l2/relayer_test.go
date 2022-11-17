package l2_test

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/bridge/l2"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
)

var (
	templateL2Message = []*orm.L2Message{
		{
			Nonce:      1,
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "100",
			Fee:        "100",
			GasLimit:   11529940,
			Deadline:   uint64(time.Now().Unix()),
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash0",
		},
	}
)

func testCreateNewRelayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, cfg.L2Config.ProofGenerationFreq, cfg.L2Config.SkippedOpcodes, int64(cfg.L2Config.Confirmations), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}

func testL2RelayerProcessSaveEvents(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, l2Cfg.ProofGenerationFreq, l2Cfg.SkippedOpcodes, int64(l2Cfg.Confirmations), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	err = db.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	results := []*types.BlockResult{
		{
			BlockTrace: &types.BlockTrace{
				Number: (*hexutil.Big)(big.NewInt(int64(templateL2Message[0].Height))),
				Hash:   common.HexToHash("00"),
			},
		},
		{
			BlockTrace: &types.BlockTrace{
				Number: (*hexutil.Big)(big.NewInt(int64(templateL2Message[0].Height + 1))),
				Hash:   common.HexToHash("01"),
			},
		},
	}
	err = db.InsertBlockTraces(context.Background(), results)
	assert.NoError(t, err)

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: templateL2Message[0].Height},
		&orm.BlockInfo{Number: templateL2Message[0].Height + 1},
		"0f", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = db.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		templateL2Message[0].Height,
		templateL2Message[0].Height + 1}, batchID)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	err = db.UpdateRollupStatus(context.Background(), batchID, orm.RollupFinalized)
	assert.NoError(t, err)

	relayer.ProcessSavedEvents()

	msg, err := db.GetL2MessageByNonce(templateL2Message[0].Nonce)
	assert.NoError(t, err)
	assert.Equal(t, orm.MsgSubmitted, msg.Status)
}

func testL2RelayerProcessPendingBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, l2Cfg.ProofGenerationFreq, l2Cfg.SkippedOpcodes, int64(l2Cfg.Confirmations), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	// this blockresult has number of 0x4, need to change it to match the testcase
	// In this testcase scenario, db will store two blocks with height 0x4 and 0x3
	var results []*types.BlockResult

	templateBlockResult, err := os.ReadFile("../../common/testdata/blockResult_relayer_parent.json")
	assert.NoError(t, err)
	blockResult := &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(t, err)
	results = append(results, blockResult)
	templateBlockResult, err = os.ReadFile("../../common/testdata/blockResult_relayer.json")
	assert.NoError(t, err)
	blockResult = &types.BlockResult{}
	err = json.Unmarshal(templateBlockResult, blockResult)
	assert.NoError(t, err)
	results = append(results, blockResult)

	err = db.InsertBlockTraces(context.Background(), results)
	assert.NoError(t, err)

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: results[0].BlockTrace.Number.ToInt().Uint64()},
		&orm.BlockInfo{Number: results[1].BlockTrace.Number.ToInt().Uint64()},
		"ff", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = db.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		results[0].BlockTrace.Number.ToInt().Uint64(),
		results[1].BlockTrace.Number.ToInt().Uint64()}, batchID)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	// err = db.UpdateRollupStatus(context.Background(), batchID, orm.RollupPending)
	// assert.NoError(t, err)

	relayer.ProcessPendingBatches()

	// Check if Rollup Result is changed successfully
	status, err := db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupCommitting, status)
}

func testL2RelayerProcessCommittedBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := l2.NewLayer2Relayer(context.Background(), l2Cli, l2Cfg.ProofGenerationFreq, l2Cfg.SkippedOpcodes, int64(l2Cfg.Confirmations), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx, &orm.BlockInfo{}, &orm.BlockInfo{}, "0", 1, 194676) // startBlock & endBlock & parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	err = db.UpdateRollupStatus(context.Background(), batchID, orm.RollupCommitted)
	assert.NoError(t, err)

	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	err = db.UpdateProofByID(context.Background(), batchID, tProof, tInstanceCommitments, 100)
	assert.NoError(t, err)
	err = db.UpdateProvingStatus(batchID, orm.ProvingTaskVerified)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()

	status, err := db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupFinalizing, status)
}
