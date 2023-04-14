package tests

import (
	"context"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types"

	"scroll-tech/bridge/relayer"
	"scroll-tech/bridge/watcher"

	"scroll-tech/database"
	"scroll-tech/database/migrate"
)

func testRelayL2MessageSucceed(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	prepareContracts(t)

	l2Cfg := cfg.L2Config

	// Create L2Watcher
	confirmations := rpc.LatestBlockNumber
	l2Watcher := watcher.NewL2WatcherClient(context.Background(), l2Client, confirmations, l2Cfg.L2MessengerAddress, l2Cfg.L2MessageQueueAddress, l2Cfg.WithdrawTrieRootSlot, db)

	// Create L2Relayer
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Create L1Watcher
	l1Cfg := cfg.L1Config
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db)

	// send message through l2 messenger contract
	nonce, err := l2MessengerInstance.MessageNonce(&bind.CallOpts{})
	assert.NoError(t, err)
	sendTx, err := l2MessengerInstance.SendMessage(l2Auth, l1Auth.From, big.NewInt(0), common.Hex2Bytes("00112233"), big.NewInt(0))
	assert.NoError(t, err)
	sendReceipt, err := bind.WaitMined(context.Background(), l2Client, sendTx)
	assert.NoError(t, err)
	if sendReceipt.Status != geth_types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// l2 watch process events
	l2Watcher.FetchContractEvent()

	// check db status
	msg, err := db.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, msg.Status, types.MsgPending)
	assert.Equal(t, msg.Sender, l2Auth.From.String())
	assert.Equal(t, msg.Target, l1Auth.From.String())

	// add fake blocks
	traces := []*types.WrappedBlock{
		{
			Header: &geth_types.Header{
				Number:     sendReceipt.BlockNumber,
				ParentHash: common.Hash{},
				Difficulty: big.NewInt(0),
				BaseFee:    big.NewInt(0),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
	}
	assert.NoError(t, db.InsertWrappedBlocks(traces))

	parentBatch := &types.BlockBatch{
		Index: 0,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData := types.NewBatchData(parentBatch, []*types.WrappedBlock{
		traces[0],
	}, cfg.L2Config.BatchProposerConfig.PublicInputConfig)
	batchHash := batchData.Hash().String()

	// add fake batch
	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	assert.NoError(t, db.NewBatchInDBTx(dbTx, batchData))
	var blockIDs = make([]uint64, len(batchData.Batch.Blocks))
	for i, block := range batchData.Batch.Blocks {
		blockIDs[i] = block.BlockNumber
	}
	err = db.SetBatchHashForL2BlocksInDBTx(dbTx, blockIDs, batchHash)
	assert.NoError(t, err)
	assert.NoError(t, dbTx.Commit())

	// add dummy proof
	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	err = db.UpdateProofByHash(context.Background(), batchHash, tProof, tInstanceCommitments, 100)
	assert.NoError(t, err)
	err = db.UpdateProvingStatus(batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process pending batch and check status
	l2Relayer.SendCommitTx([]*types.BatchData{batchData})
	status, err := db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitting, status)
	commitTxHash, err := db.GetCommitTxHash(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, true, commitTxHash.Valid)
	commitTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(commitTxHash.String))
	assert.NoError(t, err)
	commitTxReceipt, err := bind.WaitMined(context.Background(), l1Client, commitTx)
	assert.NoError(t, err)
	assert.Equal(t, len(commitTxReceipt.Logs), 1)

	// fetch CommitBatch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupCommitted, status)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()
	status, err = db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalizing, status)
	finalizeTxHash, err := db.GetFinalizeTxHash(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, true, finalizeTxHash.Valid)
	finalizeTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(finalizeTxHash.String))
	assert.NoError(t, err)
	finalizeTxReceipt, err := bind.WaitMined(context.Background(), l1Client, finalizeTx)
	assert.NoError(t, err)
	assert.Equal(t, len(finalizeTxReceipt.Logs), 1)

	// fetch FinalizeBatch events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchHash)
	assert.NoError(t, err)
	assert.Equal(t, types.RollupFinalized, status)

	// process l2 messages
	l2Relayer.ProcessSavedEvents()
	msg, err = db.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, msg.Status, types.MsgSubmitted)
	relayTxHash, err := db.GetRelayL2MessageTxHash(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, true, relayTxHash.Valid)
	relayTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(relayTxHash.String))
	assert.NoError(t, err)
	relayTxReceipt, err := bind.WaitMined(context.Background(), l1Client, relayTx)
	assert.NoError(t, err)
	assert.Equal(t, len(relayTxReceipt.Logs), 1)

	// fetch message relayed events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	msg, err = db.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, msg.Status, types.MsgConfirmed)
}
