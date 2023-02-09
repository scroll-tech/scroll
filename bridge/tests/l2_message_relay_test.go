package tests

import (
	"context"
	"math/big"
	"scroll-tech/database"
	"scroll-tech/database/migrate"
	"scroll-tech/database/orm"
	"testing"

	"scroll-tech/bridge/l1"
	"scroll-tech/bridge/l2"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func testRelayL2MessageSucceed(t *testing.T) {
	// Create db handler and reset db.
	db, err := database.NewOrmFactory(cfg.DBConfig)
	assert.NoError(t, err)
	assert.NoError(t, migrate.ResetDB(db.GetDB().DB))
	defer db.Close()

	prepareContracts(t)

	// Create L2Relayer
	l2Cfg := cfg.L2Config
	l2Relayer, err := l2.NewLayer2Relayer(context.Background(), db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)
	defer l2Relayer.Stop()

	// Create L2Watcher
	l2Watcher := l2.NewL2WatcherClient(context.Background(), l2Client, 0, l2Cfg.BatchProposerConfig, l2Cfg.L2MessengerAddress, db)

	// Create L1Watcher
	l1Cfg := cfg.L1Config
	l1Watcher := l1.NewWatcher(context.Background(), l1Client, 0, 0, l1Cfg.L1MessengerAddress, l1Cfg.RollupContractAddress, db)

	// send message through l2 messenger contract
	nonce, err := l2MessengerInstance.MessageNonce(&bind.CallOpts{})
	assert.NoError(t, err)
	sendTx, err := l2MessengerInstance.SendMessage(l2Auth, l1Auth.From, big.NewInt(0), common.Hex2Bytes("00112233"), big.NewInt(0))
	assert.NoError(t, err)
	sendReceipt, err := bind.WaitMined(context.Background(), l2Client, sendTx)
	assert.NoError(t, err)
	if sendReceipt.Status != types.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// l2 watch process events
	l2Watcher.FetchContractEvent(sendReceipt.BlockNumber.Uint64())

	// check db status
	msg, err := db.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, msg.Status, orm.MsgPending)
	assert.Equal(t, msg.Sender, l2Auth.From.String())
	assert.Equal(t, msg.Target, l1Auth.From.String())

	// add fake blocks
	traces := []*types.BlockTrace{
		{
			Header: &types.Header{
				Number:     sendReceipt.BlockNumber,
				ParentHash: common.Hash{},
				Difficulty: big.NewInt(0),
				BaseFee:    big.NewInt(0),
			},
			StorageTrace: &types.StorageTrace{},
		},
	}
	err = db.InsertBlockTraces(traces)
	assert.NoError(t, err)

	// add fake batch
	dbTx, err := db.Beginx()
	assert.NoError(t, err)
	batchID, err := db.NewBatchInDBTx(dbTx,
		&orm.BlockInfo{
			Number:     traces[0].Header.Number.Uint64(),
			Hash:       traces[0].Header.Hash().String(),
			ParentHash: traces[0].Header.ParentHash.String(),
		},
		&orm.BlockInfo{
			Number:     traces[0].Header.Number.Uint64(),
			Hash:       traces[0].Header.Hash().String(),
			ParentHash: traces[0].Header.ParentHash.String(),
		},
		traces[0].Header.ParentHash.String(), 1, 194676)
	assert.NoError(t, err)
	err = db.SetBatchIDForBlocksInDBTx(dbTx, []uint64{
		traces[0].Header.Number.Uint64(),
		traces[0].Header.Number.Uint64()}, batchID)
	assert.NoError(t, err)
	err = dbTx.Commit()
	assert.NoError(t, err)

	// add dummy proof
	tProof := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	tInstanceCommitments := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	err = db.UpdateProofByID(context.Background(), batchID, tProof, tInstanceCommitments, 100)
	assert.NoError(t, err)
	err = db.UpdateProvingStatus(batchID, orm.ProvingTaskVerified)
	assert.NoError(t, err)

	// process pending batch and check status
	l2Relayer.ProcessPendingBatches()
	status, err := db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupCommitting, status)
	commitTxHash, err := db.GetCommitTxHash(batchID)
	assert.NoError(t, err)
	assert.Equal(t, true, commitTxHash.Valid)
	commitTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(commitTxHash.String))
	assert.NoError(t, err)
	commitTxReceipt, err := bind.WaitMined(context.Background(), l1Client, commitTx)
	assert.NoError(t, err)
	assert.Equal(t, len(commitTxReceipt.Logs), 1)

	// fetch CommitBatch rollup events
	err = l1Watcher.FetchContractEvent(commitTxReceipt.BlockNumber.Uint64())
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupCommitted, status)

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()
	status, err = db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupFinalizing, status)
	finalizeTxHash, err := db.GetFinalizeTxHash(batchID)
	assert.NoError(t, err)
	assert.Equal(t, true, finalizeTxHash.Valid)
	finalizeTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(finalizeTxHash.String))
	assert.NoError(t, err)
	finalizeTxReceipt, err := bind.WaitMined(context.Background(), l1Client, finalizeTx)
	assert.NoError(t, err)
	assert.Equal(t, len(finalizeTxReceipt.Logs), 1)

	// fetch FinalizeBatch events
	err = l1Watcher.FetchContractEvent(finalizeTxReceipt.BlockNumber.Uint64())
	assert.NoError(t, err)
	status, err = db.GetRollupStatus(batchID)
	assert.NoError(t, err)
	assert.Equal(t, orm.RollupFinalized, status)

	// process l2 messages
	l2Relayer.ProcessSavedEvents()
	msg, err = db.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, msg.Status, orm.MsgSubmitted)
	relayTxHash, err := db.GetRelayL2MessageTxHash(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, true, relayTxHash.Valid)
	relayTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(relayTxHash.String))
	assert.NoError(t, err)
	relayTxReceipt, err := bind.WaitMined(context.Background(), l1Client, relayTx)
	assert.NoError(t, err)
	assert.Equal(t, len(relayTxReceipt.Logs), 1)

	// fetch message relayed events
	err = l1Watcher.FetchContractEvent(relayTxReceipt.BlockNumber.Uint64())
	assert.NoError(t, err)
	msg, err = db.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, msg.Status, orm.MsgConfirmed)
}
