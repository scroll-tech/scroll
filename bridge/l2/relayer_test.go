package l2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

/*
var (
	templateL2Message = []*types.L2Message{
		{
			Nonce:      1,
			Height:     1,
			Sender:     "0x596a746661dbed76a84556111c2872249b070e15",
			Value:      "100",
			Target:     "0x2c73620b223808297ea734d946813f0dd78eb8f7",
			Calldata:   "testdata",
			Layer2Hash: "hash0",
		},
	}
)
*/

func testCreateNewRelayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := NewLayer2Relayer(context.Background(), db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	relayer.Start()
}

/*
func testL2RelayerProcessSaveEvents(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	err = db.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	traces := []*geth_types.BlockTrace{
		{
			Header: &geth_types.Header{
				Number: big.NewInt(int64(templateL2Message[0].Height)),
			},
		},
		{
			Header: &geth_types.Header{
				Number: big.NewInt(int64(templateL2Message[0].Height + 1)),
			},
		},
	}
	err = db.InsertBlockTraces(traces)
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

	err = db.UpdateRollupStatus(context.Background(), batchID, types.RollupFinalized)
	assert.NoError(t, err)

	relayer.ProcessSavedEvents()

	msg, err := db.GetL2MessageByNonce(templateL2Message[0].Nonce)
	assert.NoError(t, err)
	assert.Equal(t, types.MsgSubmitted, msg.Status)
}
*/

/*
func testL2RelayerProcessPendingBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	// this blockresult has number of 0x4, need to change it to match the testcase
	// In this testcase scenario, db will store two blocks with height 0x4 and 0x3
	var traces []*geth_types.BlockTrace

	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	blockTrace := &geth_types.BlockTrace{}
	err = json.Unmarshal(templateBlockTrace, blockTrace)
	assert.NoError(t, err)
	traces = append(traces, blockTrace)
	templateBlockTrace, err = os.ReadFile("../../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	blockTrace = &geth_types.BlockTrace{}
	err = json.Unmarshal(templateBlockTrace, blockTrace)
	assert.NoError(t, err)
	traces = append(traces, blockTrace)

	err = db.InsertBlockTraces(traces)
	assert.NoError(t, err)

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: traces[0].Header.Number.Uint64()},
		&orm.BlockInfo{Number: traces[1].Header.Number.Uint64()},
		"ff", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = db.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		traces[0].Header.Number.Uint64(),
		traces[1].Header.Number.Uint64()}, batchID)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	// err = db.UpdateRollupStatus(context.Background(), batchID, orm.RollupPending)
	// assert.NoError(t, err)

	relayer.ProcessPendingBatches()

	// Check if Rollup Result is changed successfully
	status, err := db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitting, status)
}
*/

/*
func testL2RelayerProcessCommittedBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx, &orm.BlockInfo{}, &types.BlockInfo{}, "0", 1, 194676) // startBlock & endBlock & parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	err = db.UpdateRollupStatus(context.Background(), batchID, types.RollupCommitted)
	assert.NoError(t, err)

	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	err = db.UpdateProofByID(context.Background(), batchID, tProof, tInstanceCommitments, 100)
	assert.NoError(t, err)
	err = db.UpdateProvingStatus(batchID, types.ProvingTaskVerified)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()

	status, err := db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalizing, status)
}
*/

/*
func testL2RelayerSkipBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	createBatch := func(rollupStatus types.RollupStatus, provingStatus types.ProvingStatus) string {
		dbTx, err := db.Beginx()
		assert.NoError(t, err)
		batchID, err := db.NewBatchInDBTx(dbTx, &orm.BlockInfo{}, &orm.BlockInfo{}, "0", 1, 194676) // startBlock & endBlock & parentHash & totalTxNum & totalL2Gas don't really matter here
		assert.NoError(t, err)
		err = dbTx.Commit()
		assert.NoError(t, err)

		err = db.UpdateRollupStatus(context.Background(), batchID, rollupStatus)
		assert.NoError(t, err)

		tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
		tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
		err = db.UpdateProofByID(context.Background(), batchID, tProof, tInstanceCommitments, 100)
		assert.NoError(t, err)
		err = db.UpdateProvingStatus(batchID, provingStatus)
		assert.NoError(t, err)

		return batchID
	}

	skipped := []string{
		createBatch(types.RollupCommitted, types.ProvingTaskSkipped),
		createBatch(types.RollupCommitted, types.ProvingTaskFailed),
	}

	notSkipped := []string{
		createBatch(types.RollupPending, types.ProvingTaskSkipped),
		createBatch(types.RollupCommitting, types.ProvingTaskSkipped),
		createBatch(types.RollupFinalizing, types.ProvingTaskSkipped),
		createBatch(types.RollupFinalized, types.ProvingTaskSkipped),
		createBatch(types.RollupPending, types.ProvingTaskFailed),
		createBatch(types.RollupCommitting, types.ProvingTaskFailed),
		createBatch(types.RollupFinalizing, types.ProvingTaskFailed),
		createBatch(types.RollupFinalized, types.ProvingTaskFailed),
		createBatch(types.RollupCommitted, types.ProvingTaskVerified),
	}

	relayer.ProcessCommittedBatches()

	for _, id := range skipped {
		status, err := db.GetRollupStatus(id)
		assert.NoError(t, err)
		assert.Equal(t, types.RollupFinalizationSkipped, status)
	}

	for _, id := range notSkipped {
		status, err := db.GetRollupStatus(id)
		assert.NoError(t, err)
		assert.NotEqual(t, types.RollupFinalizationSkipped, status)
	}
}
*/
