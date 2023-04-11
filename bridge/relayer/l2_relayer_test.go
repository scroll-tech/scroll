package relayer_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"scroll-tech/common/utils"
	"strconv"
	"strings"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"

	"scroll-tech/bridge/relayer"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

var (
	templateL2Message = []*types.L2Message{
		{
			Nonce:      1,
			MsgHash:    "msg_hash1",
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

	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
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
	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
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
		Hash:      common.Hash{}.String(),
		StateRoot: common.Hash{}.String(),
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
	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	parentBatch1 := &types.BlockBatch{
		Index:     0,
		Hash:      common.Hash{}.String(),
		StateRoot: common.Hash{}.String(),
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
	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
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

func testL2CheckSubmittedMessages(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	auth, err := bind.NewKeyedTransactorWithChainID(cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], l2ChainID)
	assert.NoError(t, err)

	signedTx, err := mockTx(auth)
	assert.NoError(t, err)
	err = db.SaveTx(templateL2Message[0].MsgHash, auth.From.String(), types.L2toL1MessageTx, signedTx, "")
	assert.Nil(t, err)
	err = db.SaveL2Messages(context.Background(), templateL2Message)
	assert.NoError(t, err)
	err = db.UpdateLayer2Status(context.Background(), templateL2Message[0].MsgHash, types.MsgSubmitted)
	assert.NoError(t, err)

	cfg.L2Config.RelayerConfig.SenderConfig.Confirmations = 0
	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, cfg.L2Config.RelayerConfig)
	assert.NoError(t, err)
	assert.NotNil(t, relayer)
	err = relayer.CheckSubmittedMessages()
	assert.Nil(t, err)
	relayer.WaitSubmittedMessages()

	var (
		maxNonce uint64
		txMsgs   []*types.ScrollTx
	)
	utils.TryTimes(5, func() bool {
		// check tx is confirmed.
		maxNonce, txMsgs, err = db.GetL2TxMessages(
			map[string]interface{}{"status": types.MsgConfirmed},
			fmt.Sprintf("AND nonce > %d", 0),
			fmt.Sprintf("ORDER BY nonce ASC LIMIT %d", 10),
		)
		return err == nil
	})

	assert.Nil(t, err)
	assert.Equal(t, 1, len(txMsgs))
	assert.Equal(t, templateL2Message[0].Nonce, maxNonce)

	// check tx is on chain.
	_, err = l1Cli.TransactionReceipt(context.Background(), common.HexToHash(txMsgs[0].TxHash.String))
	assert.NoError(t, err)
}

func testL2CheckRollupCommittingBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	var batches []*types.BatchData
	batches = append(batches, types.NewBatchData(&types.BlockBatch{
		Index:     0,
		Hash:      common.Hash{}.String(),
		StateRoot: common.Hash{}.String(),
	}, []*types.WrappedBlock{wrappedBlock1}, nil))
	batches = append(batches, types.NewBatchData(&types.BlockBatch{
		Index:     batches[0].Batch.BatchIndex,
		Hash:      batches[0].Hash().Hex(),
		StateRoot: batches[0].Batch.NewStateRoot.String(),
	}, []*types.WrappedBlock{wrappedBlock2}, nil))

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchHashes := make([]string, len(batches))
	for i, batch := range batches {
		assert.NoError(t, db.NewBatchInDBTx(dbTx, batch))
		batchHash := batch.Hash().Hex()
		batchHashes[i] = batchHash
		assert.NoError(t, db.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{1}, batchHash))
		assert.NoError(t, db.UpdateCommitTxHashAndRollupStatus(context.Background(), batchHash, "", types.RollupCommitting))
	}
	assert.NoError(t, dbTx.Commit())

	l2Cfg := cfg.L2Config
	l2Cfg.RelayerConfig.SenderConfig.Confirmations = 0
	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], l2ChainID)
	assert.NoError(t, err)
	signedTx, err := mockTx(auth)
	assert.NoError(t, err)
	id := "rollup committing tx"
	err = db.SaveTx(id, auth.From.String(), types.RollUpCommitTx, signedTx, strings.Join(batchHashes, ","))
	assert.NoError(t, err)

	assert.NoError(t, relayer.CheckRollupCommittingBatches())
	relayer.WaitRollupCommittingBatches()

	var txMsgs []*types.ScrollTx
	utils.TryTimes(5, func() bool {
		// check tx is confirmed.
		txMsgs, err = db.GetScrollTxs(
			map[string]interface{}{
				"type":    types.RollUpCommitTx,
				"confirm": true,
			},
			"ORDER BY nonce ASC",
		)
		return err == nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(txMsgs))
	assert.Equal(t, "", txMsgs[0].ExtraData.String)
	// check tx is on chain.
	_, err = l1Cli.TransactionReceipt(context.Background(), common.HexToHash(txMsgs[0].TxHash.String))
	assert.NoError(t, err)
}

func testL2CheckRollupFinalizingBatches(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	var batches []*types.BatchData
	batches = append(batches, types.NewBatchData(&types.BlockBatch{
		Index:     0,
		Hash:      common.Hash{}.String(),
		StateRoot: common.Hash{}.String(),
	}, []*types.WrappedBlock{wrappedBlock1}, nil))
	batches = append(batches, types.NewBatchData(&types.BlockBatch{
		Index:     batches[0].Batch.BatchIndex,
		Hash:      batches[0].Hash().Hex(),
		StateRoot: batches[0].Batch.NewStateRoot.String(),
	}, []*types.WrappedBlock{wrappedBlock2}, nil))

	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchHashes := make([]string, len(batches))
	for i, batch := range batches {
		assert.NoError(t, db.NewBatchInDBTx(dbTx, batch))
		batchHash := batch.Hash().Hex()
		batchHashes[i] = batchHash
		assert.NoError(t, db.SetBatchHashForL2BlocksInDBTx(dbTx, []uint64{1}, batchHash))
	}
	assert.NoError(t, dbTx.Commit())
	assert.NoError(t, db.UpdateFinalizeTxHashAndRollupStatus(context.Background(), batchHashes[0], "", types.RollupFinalizing))

	l2Cfg := cfg.L2Config
	l2Cfg.RelayerConfig.SenderConfig.Confirmations = 0
	relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Cli, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	auth, err := bind.NewKeyedTransactorWithChainID(cfg.L2Config.RelayerConfig.MessageSenderPrivateKeys[0], l2ChainID)
	assert.NoError(t, err)

	signedTx, err := mockTx(auth)
	assert.NoError(t, err)
	err = db.SaveTx(batchHashes[0], auth.From.String(), types.RollupFinalizeTx, signedTx, strings.Join(batchHashes, ","))
	assert.NoError(t, err)
	assert.NoError(t, relayer.CheckRollupFinalizingBatches())
	relayer.WaitRollupFinalizingBatches()

	var txMsgs []*types.ScrollTx
	utils.TryTimes(5, func() bool {
		// check tx is confirmed.
		txMsgs, err = db.GetScrollTxs(
			map[string]interface{}{
				"type":    types.RollupFinalizeTx,
				"confirm": true,
			},
			"ORDER BY nonce ASC",
		)
		return err == nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(txMsgs))
	assert.Equal(t, "", txMsgs[0].ExtraData.String)
	// check tx is on chain.
	_, err = l1Cli.TransactionReceipt(context.Background(), common.HexToHash(txMsgs[0].TxHash.String))
	assert.NoError(t, err)
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
