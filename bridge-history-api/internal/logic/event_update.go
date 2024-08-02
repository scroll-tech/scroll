package logic

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/orm"
	btypes "scroll-tech/bridge-history-api/internal/types"
	"scroll-tech/bridge-history-api/internal/utils"
)

// EventUpdateLogic the logic of insert/update the database
type EventUpdateLogic struct {
	db                         *gorm.DB
	crossMessageOrm            *orm.CrossMessage
	batchEventOrm              *orm.BatchEvent
	bridgeBatchDepositEventOrm *orm.BridgeBatchDepositEvent

	eventUpdateLogicL1FinalizeBatchEventL2BlockUpdateHeight prometheus.Gauge
	eventUpdateLogicL2MessageNonceUpdateHeight              prometheus.Gauge
}

// NewEventUpdateLogic creates a EventUpdateLogic instance
func NewEventUpdateLogic(db *gorm.DB, isL1 bool) *EventUpdateLogic {
	b := &EventUpdateLogic{
		db:                         db,
		crossMessageOrm:            orm.NewCrossMessage(db),
		batchEventOrm:              orm.NewBatchEvent(db),
		bridgeBatchDepositEventOrm: orm.NewBridgeBatchDepositEvent(db),
	}

	if !isL1 {
		reg := prometheus.DefaultRegisterer
		b.eventUpdateLogicL1FinalizeBatchEventL2BlockUpdateHeight = promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "event_update_logic_L1_finalize_batch_event_L2_block_update_height",
			Help: "L2 block height of the latest L1 batch event that has been finalized and updated in the message_table.",
		})
		b.eventUpdateLogicL2MessageNonceUpdateHeight = promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "event_update_logic_L2_message_nonce_update_height",
			Help: "L2 message nonce height in the latest L1 batch event that has been finalized and updated in the message_table.",
		})
	}

	return b
}

// GetL1SyncHeight gets the l1 sync height from db
func (b *EventUpdateLogic) GetL1SyncHeight(ctx context.Context) (uint64, uint64, uint64, error) {
	messageSyncedHeight, err := b.crossMessageOrm.GetMessageSyncedHeightInDB(ctx, btypes.MessageTypeL1SentMessage)
	if err != nil {
		log.Error("failed to get L1 cross message synced height", "error", err)
		return 0, 0, 0, err
	}

	batchSyncedHeight, err := b.batchEventOrm.GetBatchEventSyncedHeightInDB(ctx)
	if err != nil {
		log.Error("failed to get L1 batch event synced height", "error", err)
		return 0, 0, 0, err
	}

	bridgeBatchDepositSyncedHeight, err := b.bridgeBatchDepositEventOrm.GetMessageL1SyncedHeightInDB(ctx)
	if err != nil {
		log.Error("failed to get l1 bridge batch deposit synced height", "error", err)
		return 0, 0, 0, err
	}

	return messageSyncedHeight, batchSyncedHeight, bridgeBatchDepositSyncedHeight, nil
}

// GetL2MessageSyncedHeightInDB gets L2 messages synced height
func (b *EventUpdateLogic) GetL2MessageSyncedHeightInDB(ctx context.Context) (uint64, uint64, error) {
	l2SentMessageSyncedHeight, err := b.crossMessageOrm.GetMessageSyncedHeightInDB(ctx, btypes.MessageTypeL2SentMessage)
	if err != nil {
		log.Error("failed to get L2 cross message processed height", "err", err)
		return 0, 0, err
	}

	l2BridgeBatchDepositSyncHeight, err := b.bridgeBatchDepositEventOrm.GetMessageL2SyncedHeightInDB(ctx)
	if err != nil {
		log.Error("failed to get bridge batch deposit processed height", "err", err)
		return 0, 0, err
	}
	return l2SentMessageSyncedHeight, l2BridgeBatchDepositSyncHeight, nil
}

// L1InsertOrUpdate inserts or updates l1 messages
func (b *EventUpdateLogic) L1InsertOrUpdate(ctx context.Context, l1FetcherResult *L1FilterResult) error {
	if err := b.crossMessageOrm.InsertOrUpdateL1Messages(ctx, l1FetcherResult.DepositMessages); err != nil {
		log.Error("failed to insert L1 deposit messages", "err", err)
		return err
	}

	if err := b.crossMessageOrm.InsertOrUpdateL1RelayedMessagesOfL2Withdrawals(ctx, l1FetcherResult.RelayedMessages); err != nil {
		log.Error("failed to update L1 relayed messages of L2 withdrawals", "err", err)
		return err
	}

	if err := b.batchEventOrm.InsertOrUpdateBatchEvents(ctx, l1FetcherResult.BatchEvents); err != nil {
		log.Error("failed to insert or update batch events", "err", err)
		return err
	}

	if err := b.crossMessageOrm.UpdateL1MessageQueueEventsInfo(ctx, l1FetcherResult.MessageQueueEvents); err != nil {
		log.Error("failed to insert L1 message queue events", "err", err)
		return err
	}

	if err := b.crossMessageOrm.InsertFailedL1GatewayTxs(ctx, l1FetcherResult.RevertedTxs); err != nil {
		log.Error("failed to insert failed L1 gateway transactions", "err", err)
		return err
	}

	if err := b.bridgeBatchDepositEventOrm.InsertOrUpdateL1BridgeBatchDepositEvent(ctx, l1FetcherResult.BridgeBatchDepositEvents); err != nil {
		log.Error("failed to insert L1 bridge batch deposit transactions", "err", err)
		return err
	}

	return nil
}

func (b *EventUpdateLogic) updateL2WithdrawMessageInfos(ctx context.Context, batchIndex, startBlock, endBlock uint64) error {
	if startBlock > endBlock {
		log.Warn("start block is greater than end block", "start", startBlock, "end", endBlock)
		return nil
	}

	l2WithdrawMessages, err := b.crossMessageOrm.GetL2WithdrawalsByBlockRange(ctx, startBlock, endBlock)
	if err != nil {
		log.Error("failed to get L2 withdrawals by batch index", "batch index", batchIndex, "err", err)
		return err
	}

	if len(l2WithdrawMessages) == 0 {
		return nil
	}

	withdrawTrie := utils.NewWithdrawTrie()
	lastMessage, err := b.crossMessageOrm.GetL2LatestFinalizedWithdrawal(ctx)
	if err != nil {
		log.Error("failed to get latest L2 finalized sent message event", "err", err)
		return err
	}

	if lastMessage != nil {
		withdrawTrie.Initialize(lastMessage.MessageNonce, common.HexToHash(lastMessage.MessageHash), lastMessage.MerkleProof)
	}

	if withdrawTrie.NextMessageNonce != l2WithdrawMessages[0].MessageNonce {
		log.Error("nonce mismatch", "expected next message nonce", withdrawTrie.NextMessageNonce, "actual next message nonce", l2WithdrawMessages[0].MessageNonce)
		return errors.New("nonce mismatch")
	}

	messageHashes := make([]common.Hash, len(l2WithdrawMessages))
	for i, message := range l2WithdrawMessages {
		messageHashes[i] = common.HexToHash(message.MessageHash)
	}

	proofs := withdrawTrie.AppendMessages(messageHashes)

	for i, message := range l2WithdrawMessages {
		message.MerkleProof = proofs[i]
		message.RollupStatus = int(btypes.RollupStatusTypeFinalized)
		message.BatchIndex = batchIndex
	}

	if dbErr := b.crossMessageOrm.UpdateBatchIndexRollupStatusMerkleProofOfL2Messages(ctx, l2WithdrawMessages); dbErr != nil {
		log.Error("failed to update batch index and rollup status and merkle proof of L2 messages", "err", dbErr)
		return dbErr
	}

	b.eventUpdateLogicL2MessageNonceUpdateHeight.Set(float64(withdrawTrie.NextMessageNonce - 1))
	return nil
}

// UpdateL2WithdrawMessageProofs updates L2 withdrawal message proofs.
func (b *EventUpdateLogic) UpdateL2WithdrawMessageProofs(ctx context.Context, height uint64) error {
	lastUpdatedFinalizedBlockHeight, err := b.batchEventOrm.GetLastUpdatedFinalizedBlockHeight(ctx)
	if err != nil {
		log.Error("failed to get last updated finalized block height", "error", err)
		return err
	}

	finalizedBatches, err := b.batchEventOrm.GetUnupdatedFinalizedBatchesLEBlockHeight(ctx, height)
	if err != nil {
		log.Error("failed to get unupdated finalized batches >= block height", "error", err)
		return err
	}

	for _, finalizedBatch := range finalizedBatches {
		log.Info("update finalized batch or bundle info of L2 withdrawals", "index", finalizedBatch.BatchIndex, "lastUpdatedFinalizedBlockHeight", lastUpdatedFinalizedBlockHeight, "start", finalizedBatch.StartBlockNumber, "end", finalizedBatch.EndBlockNumber)
		// This method is compatible with both "finalize by batch" and "finalize by bundle" modes:
		// - In "finalize by batch" mode, each batch emits a FinalizedBatch event.
		// - In "finalize by bundle" mode, all batches in the bundle emit only one FinalizedBatch event, using the last batch's index and hash.
		//
		// The method updates two types of information in L2 withdrawal messages:
		// 1. Withdraw proof generation:
		//    - finalize by batch: Generates proofs for each batch.
		//    - finalize by bundle: Generates proofs for the entire bundle at once.
		// 2. Batch index updating:
		//    - finalize by batch: Updates the batch index for withdrawal messages in each processed batch.
		//    - finalize by bundle: Updates the batch index for all withdrawal messages in the bundle, using the index of the last batch in the bundle.
		if updateErr := b.updateL2WithdrawMessageInfos(ctx, finalizedBatch.BatchIndex, lastUpdatedFinalizedBlockHeight+1, finalizedBatch.EndBlockNumber); updateErr != nil {
			log.Error("failed to update L2 withdraw message infos", "index", finalizedBatch.BatchIndex, "lastUpdatedFinalizedBlockHeight", lastUpdatedFinalizedBlockHeight, "start", finalizedBatch.StartBlockNumber, "end", finalizedBatch.EndBlockNumber, "error", updateErr)
			return updateErr
		}
		if dbErr := b.batchEventOrm.UpdateBatchEventStatus(ctx, finalizedBatch.BatchIndex); dbErr != nil {
			log.Error("failed to update batch event status as updated", "index", finalizedBatch.BatchIndex, "lastUpdatedFinalizedBlockHeight", lastUpdatedFinalizedBlockHeight, "start", finalizedBatch.StartBlockNumber, "end", finalizedBatch.EndBlockNumber, "error", dbErr)
			return dbErr
		}
		lastUpdatedFinalizedBlockHeight = finalizedBatch.EndBlockNumber
		b.eventUpdateLogicL1FinalizeBatchEventL2BlockUpdateHeight.Set(float64(finalizedBatch.EndBlockNumber))
	}
	return nil
}

// UpdateL2BridgeBatchDepositEvent update l2 bridge batch deposit status
func (b *EventUpdateLogic) UpdateL2BridgeBatchDepositEvent(ctx context.Context, l2BatchDistributes []*orm.BridgeBatchDepositEvent) error {
	distributeFailedMap := make(map[uint64][]string)
	for _, l2BatchDistribute := range l2BatchDistributes {
		if btypes.TxStatusType(l2BatchDistribute.TxStatus) == btypes.TxStatusBridgeBatchDistributeFailed {
			distributeFailedMap[l2BatchDistribute.BatchIndex] = append(distributeFailedMap[l2BatchDistribute.BatchIndex], l2BatchDistribute.Sender)
		}

		if err := b.bridgeBatchDepositEventOrm.UpdateBatchEventStatus(ctx, l2BatchDistribute); err != nil {
			log.Error("failed to update L1 bridge batch distribute event", "batchIndex", l2BatchDistribute.BatchIndex, "err", err)
			return err
		}
	}

	for batchIndex, distributeFailedSenders := range distributeFailedMap {
		if err := b.bridgeBatchDepositEventOrm.UpdateDistributeFailedStatus(ctx, batchIndex, distributeFailedSenders); err != nil {
			log.Error("failed to update L1 bridge batch distribute failed event", "batchIndex", batchIndex, "failed senders", distributeFailedSenders, "err", err)
			return err
		}
	}

	return nil
}

// L2InsertOrUpdate inserts or updates L2 messages
func (b *EventUpdateLogic) L2InsertOrUpdate(ctx context.Context, l2FetcherResult *L2FilterResult) error {
	if err := b.crossMessageOrm.InsertOrUpdateL2Messages(ctx, l2FetcherResult.WithdrawMessages); err != nil {
		log.Error("failed to insert L2 withdrawal messages", "err", err)
		return err
	}

	if err := b.crossMessageOrm.InsertOrUpdateL2RelayedMessagesOfL1Deposits(ctx, l2FetcherResult.RelayedMessages); err != nil {
		log.Error("failed to update L2 relayed messages of L1 deposits", "err", err)
		return err
	}

	if err := b.crossMessageOrm.InsertFailedL2GatewayTxs(ctx, l2FetcherResult.OtherRevertedTxs); err != nil {
		log.Error("failed to insert failed L2 gateway transactions", "err", err)
		return err
	}
	return nil
}
