package relayer

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
	"scroll-tech/common/utils"

	"scroll-tech/bridge/sender"

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
	assert.NotNil(t, relayer)
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

	err = db.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)

	traces := []*types.WrappedBlock{
		{
			Header: &geth_types.Header{
				Number: big.NewInt(int64(templateL2Message[0].Height)),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
		{
			Header: &geth_types.Header{
				Number: big.NewInt(int64(templateL2Message[0].Height + 1)),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
	}
	assert.NoError(t, db.InsertWrappedBlocks(traces))

	parentBatch1 := &types.BlockBatch{
		Index:     0,
		Hash:      common.Hash{}.Hex(),
		StateRoot: common.Hash{}.Hex(),
	}
	batchData1 := types.NewBatchData(parentBatch1, []*types.WrappedBlock{wrappedBlock1}, nil)
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

	parentBatch1 := &types.BlockBatch{
		Index:     0,
		Hash:      common.Hash{}.Hex(),
		StateRoot: common.Hash{}.Hex(),
	}
	batchData1 := types.NewBatchData(parentBatch1, []*types.WrappedBlock{wrappedBlock1}, nil)
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

func testL2RelayerMsgConfirm(t *testing.T) {
	// Set up the database and defer closing it.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert test data.
	assert.NoError(t, db.SaveL2Messages(context.Background(), []*types.L2Message{
		{MsgHash: "msg-1", Nonce: 0}, {MsgHash: "msg-2", Nonce: 1},
	}))

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	l2Relayer.processingMessage.Store("msg-1", "msg-1")
	l2Relayer.messageSender.SendConfirmation(&sender.Confirmation{
		ID:           "msg-1",
		IsSuccessful: true,
	})
	l2Relayer.processingMessage.Store("msg-2", "msg-2")
	l2Relayer.messageSender.SendConfirmation(&sender.Confirmation{
		ID:           "msg-2",
		IsSuccessful: false,
	})

	// Check the database for the updated status using TryTimes.
	utils.TryTimes(5, func() bool {
		msg1, err1 := db.GetL2MessageByMsgHash("msg-1")
		msg2, err2 := db.GetL2MessageByMsgHash("msg-2")
		return err1 == nil && msg1.Status == types.MsgConfirmed &&
			err2 == nil && msg2.Status == types.MsgRelayFailed
	})
}

func testL2RelayerRollupConfirm(t *testing.T) {
	// Set up the database and defer closing it.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert test data.
	batches := make([]*types.BatchData, 6)
	for i := 0; i < 6; i++ {
		batches[i] = genBatchData(t, uint64(i))
	}

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	for _, batch := range batches {
		assert.NoError(t, db.NewBatchInDBTx(dbTx, batch))
	}
	assert.NoError(t, dbTx.Commit())

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	processingKeys := []string{"committed-1", "committed-2", "finalized-1", "finalized-2"}
	isSuccessful := []bool{true, false, true, false}

	for i, key := range processingKeys[:2] {
		batchHashes := []string{batches[i*2].Hash().Hex(), batches[i*2+1].Hash().Hex()}
		l2Relayer.processingBatchesCommitment.Store(key, batchHashes)
		l2Relayer.messageSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i],
		})
	}

	for i, key := range processingKeys[2:] {
		batchHash := batches[i+4].Hash().Hex()
		l2Relayer.processingFinalization.Store(key, batchHash)
		l2Relayer.rollupSender.SendConfirmation(&sender.Confirmation{
			ID:           key,
			IsSuccessful: isSuccessful[i+2],
			TxHash:       common.HexToHash("0x56789abcdef1234"),
		})
	}

	// Check the database for the updated status using TryTimes.
	utils.TryTimes(5, func() bool {
		expectedStatuses := []types.RollupStatus{
			types.RollupCommitted,
			types.RollupCommitted,
			types.RollupCommitFailed,
			types.RollupCommitFailed,
			types.RollupFinalized,
			types.RollupFinalizeFailed,
		}

		for i, batch := range batches[:6] {
			batchInDB, err := db.GetBlockBatches(map[string]interface{}{"hash": batch.Hash().Hex()})
			if err != nil || len(batchInDB) != 1 || batchInDB[0].RollupStatus != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
}

func testL2RelayerGasOracleConfirm(t *testing.T) {
	// Set up the database and defer closing it.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	// Insert test data.
	batches := make([]*types.BatchData, 2)
	for i := 0; i < 2; i++ {
		batches[i] = genBatchData(t, uint64(i))
	}

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	for _, batch := range batches {
		assert.NoError(t, db.NewBatchInDBTx(dbTx, batch))
	}
	assert.NoError(t, dbTx.Commit())

	// Create and set up the Layer2 Relayer.
	l2Cfg := cfg.L2Config
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l2Relayer, err := NewLayer2Relayer(ctx, l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Simulate message confirmations.
	isSuccessful := []bool{true, false}
	for i, batch := range batches {
		l2Relayer.gasOracleSender.SendConfirmation(&sender.Confirmation{
			ID:           batch.Hash().Hex(),
			IsSuccessful: isSuccessful[i],
		})
	}

	// Check the database for the updated status using TryTimes.
	utils.TryTimes(5, func() bool {
		expectedStatuses := []types.GasOracleStatus{types.GasOracleImported, types.GasOracleFailed}
		for i, batch := range batches {
			gasOracle, err := db.GetBlockBatches(map[string]interface{}{"hash": batch.Hash().Hex()})
			if err != nil || len(gasOracle) != 1 || gasOracle[0].OracleStatus != expectedStatuses[i] {
				return false
			}
		}
		return true
	})
}

func genBatchData(t *testing.T, index uint64) *types.BatchData {
	templateBlockTrace, err := os.ReadFile("../../common/testdata/blockTrace_02.json")
	assert.NoError(t, err)
	// unmarshal blockTrace
	wrappedBlock := &types.WrappedBlock{}
	err = json.Unmarshal(templateBlockTrace, wrappedBlock)
	assert.NoError(t, err)
	wrappedBlock.Header.ParentHash = common.HexToHash("0x" + strconv.FormatUint(index+1, 16))
	parentBatch := &types.BlockBatch{
		Index: index,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	return types.NewBatchData(parentBatch, []*types.WrappedBlock{wrappedBlock}, nil)
}
