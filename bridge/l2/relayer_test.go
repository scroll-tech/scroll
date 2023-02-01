package l2

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/bigint"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
)

var (
	templateL2Message = []*orm.L2Message{
		{
			Nonce:      1,
			Height:     bigint.NewInt(1),
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

	relayer, err := NewLayer2Relayer(context.Background(), db, cfg.L2Config.RelayerConfig)
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
	relayer, err := NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	err = db.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	traces := []*types.BlockTrace{
		{
			Header: &types.Header{
				Number: templateL2Message[0].Height.BigInt(),
			},
		},
		{
			Header: &types.Header{
				Number: big.NewInt(0).Add(templateL2Message[0].Height.BigInt(), big.NewInt(1)),
			},
		},
	}
	err = db.InsertBlockTraces(traces)
	assert.NoError(t, err)

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: templateL2Message[0].Height},
		&orm.BlockInfo{Number: bigint.NewInt(templateL2Message[0].Height.Int64() + 1)},
		"0f", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = db.SetBatchIDForBlocksInDBTx(dbTx, []*bigint.BigInt{templateL2Message[0].Height, bigint.NewInt(templateL2Message[0].Height.Int64() + 1)}, batchID)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	err = db.UpdateRollupStatus(context.Background(), batchID, orm.RollupFinalized)
	assert.NoError(t, err)

	var wg = sync.WaitGroup{}
	wg.Add(1)
	relayer.ProcessSavedEvents(&wg)
	wg.Wait()

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
	relayer, err := NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	// this blockresult has number of 0x4, need to change it to match the testcase
	// In this testcase scenario, db will store two blocks with height 0x4 and 0x3
	var traces []*types.BlockTrace

	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	blockTrace := &types.BlockTrace{}
	err = json.Unmarshal(templateBlockTrace, blockTrace)
	assert.NoError(t, err)
	traces = append(traces, blockTrace)
	templateBlockTrace, err = os.ReadFile("../../common/testdata/blockTrace_03.json")
	assert.NoError(t, err)
	blockTrace = &types.BlockTrace{}
	err = json.Unmarshal(templateBlockTrace, blockTrace)
	assert.NoError(t, err)
	traces = append(traces, blockTrace)

	err = db.InsertBlockTraces(traces)
	assert.NoError(t, err)

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{Number: bigint.NewBigInt(traces[0].Header.Number)},
		&orm.BlockInfo{Number: bigint.NewBigInt(traces[1].Header.Number)},
		"ff", 1, 194676) // parentHash & totalTxNum & totalL2Gas don't really matter here
	assert.NoError(t, err)
	err = db.SetBatchIDForBlocksInDBTx(dbTx, []*bigint.BigInt{
		bigint.NewBigInt(traces[0].Header.Number),
		bigint.NewBigInt(traces[1].Header.Number)}, batchID)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	// err = db.UpdateRollupStatus(context.Background(), batchID, orm.RollupPending)
	// assert.NoError(t, err)

	var wg = sync.WaitGroup{}
	wg.Add(1)
	relayer.ProcessPendingBatches(&wg)
	wg.Wait()

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
	relayer, err := NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx, &orm.BlockInfo{Number: bigint.NewInt(0)}, &orm.BlockInfo{Number: bigint.NewInt(0)}, "0", 1, 194676) // startBlock & endBlock & parentHash & totalTxNum & totalL2Gas don't really matter here
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

	var wg = sync.WaitGroup{}
	wg.Add(1)
	relayer.ProcessCommittedBatches(&wg)
	wg.Wait()

	status, err := db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupFinalizing, status)
}

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

	createBatch := func(rollupStatus orm.RollupStatus, provingStatus orm.ProvingStatus) string {
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
		createBatch(orm.RollupCommitted, orm.ProvingTaskSkipped),
		createBatch(orm.RollupCommitted, orm.ProvingTaskFailed),
	}

	notSkipped := []string{
		createBatch(orm.RollupPending, orm.ProvingTaskSkipped),
		createBatch(orm.RollupCommitting, orm.ProvingTaskSkipped),
		createBatch(orm.RollupFinalizing, orm.ProvingTaskSkipped),
		createBatch(orm.RollupFinalized, orm.ProvingTaskSkipped),
		createBatch(orm.RollupPending, orm.ProvingTaskFailed),
		createBatch(orm.RollupCommitting, orm.ProvingTaskFailed),
		createBatch(orm.RollupFinalizing, orm.ProvingTaskFailed),
		createBatch(orm.RollupFinalized, orm.ProvingTaskFailed),
		createBatch(orm.RollupCommitted, orm.ProvingTaskVerified),
	}

	var wg = sync.WaitGroup{}
	wg.Add(1)
	relayer.ProcessCommittedBatches(&wg)
	wg.Wait()

	for _, id := range skipped {
		status, err := db.GetRollupStatus(id)
		assert.NoError(t, err)
		assert.Equal(t, orm.RollupFinalizationSkipped, status)
	}

	for _, id := range notSkipped {
		status, err := db.GetRollupStatus(id)
		assert.NoError(t, err)
		assert.NotEqual(t, orm.RollupFinalizationSkipped, status)
	}
}
