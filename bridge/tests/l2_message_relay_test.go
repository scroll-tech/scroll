package tests

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/bridge/internal/controller/relayer"
	"scroll-tech/bridge/internal/controller/watcher"
	"scroll-tech/bridge/internal/orm"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/bridge/internal/utils"
)

func testRelayL2MessageSucceed(t *testing.T) {
	db := setupDB(t)
	defer utils.CloseDB(db)

	prepareContracts(t)

	l2Cfg := bridgeApp.Config.L2Config

	// Create L2Watcher
	confirmations := rpc.LatestBlockNumber
	l2Watcher := watcher.NewL2WatcherClient(context.Background(), l2Client, confirmations, l2Cfg.L2MessengerAddress, l2Cfg.L2MessageQueueAddress, l2Cfg.WithdrawTrieRootSlot, db)

	// Create L2Relayer
	l2Relayer, err := relayer.NewLayer2Relayer(context.Background(), l2Client, db, l2Cfg.RelayerConfig)
	assert.NoError(t, err)

	// Create L1Watcher
	l1Cfg := bridgeApp.Config.L1Config
	l1Watcher := watcher.NewL1WatcherClient(context.Background(), l1Client, 0, confirmations, l1Cfg.L1MessengerAddress, l1Cfg.L1MessageQueueAddress, l1Cfg.ScrollChainContractAddress, db)

	// send message through l2 messenger contract
	nonce, err := l2MessengerInstance.MessageNonce(&bind.CallOpts{})
	assert.NoError(t, err)
	sendTx, err := l2MessengerInstance.SendMessage(l2Auth, l1Auth.From, big.NewInt(0), common.Hex2Bytes("00112233"), big.NewInt(0))
	assert.NoError(t, err)
	sendReceipt, err := bind.WaitMined(context.Background(), l2Client, sendTx)
	assert.NoError(t, err)
	if sendReceipt.Status != gethTypes.ReceiptStatusSuccessful || err != nil {
		t.Fatalf("Call failed")
	}

	// l2 watch process events
	l2Watcher.FetchContractEvent()
	l2MessageOrm := orm.NewL2Message(db)
	blockTraceOrm := orm.NewBlockTrace(db)
	blockBatchOrm := orm.NewBlockBatch(db)

	// check db status
	msg, err := l2MessageOrm.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, types.MsgStatus(msg.Status), types.MsgPending)
	assert.Equal(t, msg.Sender, l2Auth.From.String())
	assert.Equal(t, msg.Target, l1Auth.From.String())

	// add fake blocks
	traces := []*bridgeTypes.WrappedBlock{
		{
			Header: &gethTypes.Header{
				Number:     sendReceipt.BlockNumber,
				ParentHash: common.Hash{},
				Difficulty: big.NewInt(0),
				BaseFee:    big.NewInt(0),
			},
			Transactions:     nil,
			WithdrawTrieRoot: common.Hash{},
		},
	}
	assert.NoError(t, blockTraceOrm.InsertWrappedBlocks(traces))

	parentBatch := &bridgeTypes.BatchInfo{
		Index: 0,
		Hash:  "0x0000000000000000000000000000000000000000",
	}
	batchData := bridgeTypes.NewBatchData(parentBatch, []*bridgeTypes.WrappedBlock{traces[0]}, l2Cfg.BatchProposerConfig.PublicInputConfig)
	batchHash := batchData.Hash().String()
	// add fake batch
	err = db.Transaction(func(tx *gorm.DB) error {
		rowsAffected, dbTxErr := blockBatchOrm.InsertBlockBatchByBatchData(tx, batchData)
		if dbTxErr != nil {
			return dbTxErr
		}
		if rowsAffected != 1 {
			dbTxErr = errors.New("the InsertBlockBatchByBatchData affected row is not 1")
			return dbTxErr
		}
		var blockIDs = make([]uint64, len(batchData.Batch.Blocks))
		for i, block := range batchData.Batch.Blocks {
			blockIDs[i] = block.BlockNumber
		}
		dbTxErr = blockTraceOrm.UpdateChunkHashInClosedRange(tx, blockIDs, batchHash)
		if dbTxErr != nil {
			return dbTxErr
		}
		return nil
	})
	assert.NoError(t, err)

	// add dummy proof
	proof := &message.AggProof{
		Proof:     []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
		FinalPair: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
	}
	err = blockBatchOrm.UpdateProofByHash(context.Background(), batchHash, proof, 100)
	assert.NoError(t, err)
	err = blockBatchOrm.UpdateProvingStatus(batchHash, types.ProvingTaskVerified)
	assert.NoError(t, err)

	// process pending batch and check status
	assert.NoError(t, l2Relayer.SendCommitTx([]*bridgeTypes.BatchData{batchData}))

	blockBatches, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batchHash}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockBatches))
	assert.NotEmpty(t, blockBatches[0].CommitTxHash)
	assert.Equal(t, types.RollupCommitting, types.RollupStatus(blockBatches[0].RollupStatus))

	commitTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(blockBatches[0].CommitTxHash))
	assert.NoError(t, err)
	commitTxReceipt, err := bind.WaitMined(context.Background(), l1Client, commitTx)
	assert.NoError(t, err)
	assert.Equal(t, len(commitTxReceipt.Logs), 1)

	// fetch CommitBatch rollup events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	statuses, err := blockBatchOrm.GetRollupStatusByHashList([]string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupCommitted, statuses[0])

	// process committed batch and check status
	l2Relayer.ProcessCommittedBatches()

	blockBatchWithFinalizeTxHash, err := blockBatchOrm.GetBlockBatches(map[string]interface{}{"hash": batchHash}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(blockBatchWithFinalizeTxHash))
	assert.NotEmpty(t, blockBatchWithFinalizeTxHash[0].FinalizeTxHash)
	assert.Equal(t, types.RollupFinalizing, types.RollupStatus(blockBatchWithFinalizeTxHash[0].RollupStatus))

	finalizeTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(blockBatchWithFinalizeTxHash[0].FinalizeTxHash))
	assert.NoError(t, err)
	finalizeTxReceipt, err := bind.WaitMined(context.Background(), l1Client, finalizeTx)
	assert.NoError(t, err)
	assert.Equal(t, len(finalizeTxReceipt.Logs), 1)

	// fetch FinalizeBatch events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	statuses, err = blockBatchOrm.GetRollupStatusByHashList([]string{batchHash})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(statuses))
	assert.Equal(t, types.RollupFinalized, statuses[0])

	// process l2 messages
	l2Relayer.ProcessSavedEvents()

	l2Messages, err := l2MessageOrm.GetL2Messages(map[string]interface{}{"nonce": nonce.Uint64()}, nil, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(l2Messages))
	assert.NotEmpty(t, l2Messages[0].Layer1Hash)
	assert.Equal(t, types.MsgStatus(l2Messages[0].Status), types.MsgSubmitted)

	relayTx, _, err := l1Client.TransactionByHash(context.Background(), common.HexToHash(l2Messages[0].Layer1Hash))
	assert.NoError(t, err)
	relayTxReceipt, err := bind.WaitMined(context.Background(), l1Client, relayTx)
	assert.NoError(t, err)
	assert.Equal(t, len(relayTxReceipt.Logs), 1)

	// fetch message relayed events
	err = l1Watcher.FetchContractEvent()
	assert.NoError(t, err)
	msg, err = l2MessageOrm.GetL2MessageByNonce(nonce.Uint64())
	assert.NoError(t, err)
	assert.Equal(t, types.MsgStatus(msg.Status), types.MsgConfirmed)
}
