package l2

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"strconv"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

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

func testCreateNewRelayer(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
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
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
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
	_, err = db.InsertL2BlockTraces(traces)
	assert.NoError(t, err)

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData1))
	batchHash := batchData1.Hash().Hex()
	assert.NoError(t, db.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{1}, batchHash))
	assert.NoError(t, dbTx.Commit())

	err = db.UpdateRollupStatus(context.Background(), batchHash, types.RollupFinalized)
	assert.NoError(t, err)

	relayer.ProcessSavedEvents()

	msg, err := db.GetL2MessageByNonce(templateL2Message[0].Nonce)
	assert.NoError(t, err)
	assert.Equal(t, types.MsgSubmitted, msg.Status)
}

func testL2RelayerProcessCommittedBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData1))
	batchHash := batchData1.Hash().Hex()
	err = dbTx.Commit()
	assert.NoError(t, err)

	err = db.UpdateRollupStatus(context.Background(), batchHash, types.RollupCommitted)
	assert.NoError(t, err)

	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	err = db.UpdateProofByHash(context.Background(), batchHash, tProof, tInstanceCommitments, 100)
	assert.NoError(t, err)
	err = db.UpdateProvingStatus(batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	relayer.ProcessCommittedBatches()

	status, err := db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalizing, status)
}

func testL2RelayerSkipBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	l2Cfg := cfg.L2Config
	relayer, err := NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer relayer.Stop()

	createBatch := func(rollupStatus types.RollupStatus, provingStatus types.ProvingStatus, index uint64) string {
		dbTx, err := db.Beginx()
		assert.NoError(t, err)
		batchData := genBatchData(t, index)
		assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData))
		batchHash := batchData.Hash().Hex()
		err = dbTx.Commit()
		assert.NoError(t, err)

		err = db.UpdateRollupStatus(context.Background(), batchHash, rollupStatus)
		assert.NoError(t, err)

		tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
		tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
		err = db.UpdateProofByHash(context.Background(), batchHash, tProof, tInstanceCommitments, 100)
		assert.NoError(t, err)
		err = db.UpdateProvingStatus(batchHash, provingStatus)
		assert.NoError(t, err)

		return batchHash
	}

	skipped := []string{
		createBatch(types.RollupCommitted, types.ProvingTaskSkipped, 1),
		createBatch(types.RollupCommitted, types.ProvingTaskFailed, 2),
	}

	notSkipped := []string{
		createBatch(types.RollupPending, types.ProvingTaskSkipped, 3),
		createBatch(types.RollupCommitting, types.ProvingTaskSkipped, 4),
		createBatch(types.RollupFinalizing, types.ProvingTaskSkipped, 5),
		createBatch(types.RollupFinalized, types.ProvingTaskSkipped, 6),
		createBatch(types.RollupPending, types.ProvingTaskFailed, 7),
		createBatch(types.RollupCommitting, types.ProvingTaskFailed, 8),
		createBatch(types.RollupFinalizing, types.ProvingTaskFailed, 9),
		createBatch(types.RollupFinalized, types.ProvingTaskFailed, 10),
		createBatch(types.RollupCommitted, types.ProvingTaskVerified, 11),
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

func genBatchData(t *testing.T, index uint64) *types.BatchData {
	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	// unmarshal blockTrace
	blockTrace := &geth_types.BlockTrace{}
	err = json.Unmarshal(templateBlockTrace, blockTrace)
	assert.NoError(t, err)
	blockTrace.Header.ParentHash = common.HexToHash("0x" + strconv.FormatUint(index+1, 16))
	parentBatch := &types.BlockBatch{
		Index: index,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	return types.NewBatchData(parentBatch, []*geth_types.BlockTrace{blockTrace}, nil)
}
