package logic

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge-history-api/internal/orm"
	"scroll-tech/bridge-history-api/internal/utils"
)

// EventUpdateLogic the logic of insert/update the database
type EventUpdateLogic struct {
	db              *gorm.DB
	crossMessageOrm *orm.CrossMessage
	batchEventOrm   *orm.BatchEvent

	eventUpdateLogicL1FinalizeBatchEventL2BlockUpdateHeight prometheus.Gauge
	eventUpdateLogicL2MessageNonceUpdateHeight              prometheus.Gauge
}

// NewEventUpdateLogic creates a EventUpdateLogic instance
func NewEventUpdateLogic(db *gorm.DB, isL1 bool) *EventUpdateLogic {
	b := &EventUpdateLogic{
		db:              db,
		crossMessageOrm: orm.NewCrossMessage(db),
		batchEventOrm:   orm.NewBatchEvent(db),
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
func (b *EventUpdateLogic) GetL1SyncHeight(ctx context.Context) (uint64, uint64, error) {
	messageSyncedHeight, err := b.crossMessageOrm.GetMessageSyncedHeightInDB(ctx, orm.MessageTypeL1SentMessage)
	if err != nil {
		log.Error("failed to get L1 cross message synced height", "error", err)
		return 0, 0, err
	}

	batchSyncedHeight, err := b.batchEventOrm.GetBatchEventSyncedHeightInDB(ctx)
	if err != nil {
		log.Error("failed to get L1 batch event synced height", "error", err)
		return 0, 0, err
	}

	return messageSyncedHeight, batchSyncedHeight, nil
}

// GetL2MessageSyncedHeightInDB gets L2 messages synced height
func (b *EventUpdateLogic) GetL2MessageSyncedHeightInDB(ctx context.Context) (uint64, error) {
	l2SentMessageSyncedHeight, err := b.crossMessageOrm.GetMessageSyncedHeightInDB(ctx, orm.MessageTypeL2SentMessage)
	if err != nil {
		log.Error("failed to get L2 cross message processed height", "err", err)
		return 0, err
	}
	return l2SentMessageSyncedHeight, nil
}

// L1InsertOrUpdate inserts or updates l1 messages
func (b *EventUpdateLogic) L1InsertOrUpdate(ctx context.Context, l1FetcherResult *L1FilterResult) error {
	if txErr := b.crossMessageOrm.InsertOrUpdateL1Messages(ctx, l1FetcherResult.DepositMessages); txErr != nil {
		log.Error("failed to insert L1 deposit messages", "err", txErr)
		return txErr
	}

	if txErr := b.crossMessageOrm.InsertOrUpdateL1RelayedMessagesOfL2Withdrawals(ctx, l1FetcherResult.RelayedMessages); txErr != nil {
		log.Error("failed to update L1 relayed messages of L2 withdrawals", "err", txErr)
		return txErr
	}

	if txErr := b.batchEventOrm.InsertOrUpdateBatchEvents(ctx, l1FetcherResult.BatchEvents); txErr != nil {
		log.Error("failed to insert or update batch events", "err", txErr)
		return txErr
	}

	if txErr := b.crossMessageOrm.UpdateL1MessageQueueEventsInfo(ctx, l1FetcherResult.MessageQueueEvents); txErr != nil {
		log.Error("failed to insert L1 message queue events", "err", txErr)
		return txErr
	}

	if txErr := b.crossMessageOrm.InsertFailedL1GatewayRouterAndL1MessengerTxs(ctx, l1FetcherResult.RevertedTxs); txErr != nil {
		log.Error("failed to insert failed L1 gateway router and L1 messenger transactions", "err", txErr)
		return txErr
	}
	return nil
}

func (b *EventUpdateLogic) updateL2WithdrawMessageInfos(ctx context.Context, batchIndex, startBlock, endBlock uint64) error {
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
		log.Error("nonce mismatch", "expected next message nonce", withdrawTrie.NextMessageNonce, "actuall next message nonce", l2WithdrawMessages[0].MessageNonce)
		return fmt.Errorf("nonce mismatch")
	}

	messageHashes := make([]common.Hash, len(l2WithdrawMessages))
	for i, message := range l2WithdrawMessages {
		messageHashes[i] = common.HexToHash(message.MessageHash)
	}

	proofs := withdrawTrie.AppendMessages(messageHashes)

	for i, message := range l2WithdrawMessages {
		message.MerkleProof = proofs[i]
		message.RollupStatus = int(orm.RollupStatusTypeFinalized)
		message.BatchIndex = batchIndex
	}

	if dbErr := b.crossMessageOrm.UpdateBatchIndexRollupStatusMerkleProofOfL2Messages(ctx, l2WithdrawMessages); dbErr != nil {
		log.Error("failed to update batch index and rollup status and merkle proof of L2 messages", "err", dbErr)
		return dbErr
	}

	b.eventUpdateLogicL2MessageNonceUpdateHeight.Set(float64(withdrawTrie.NextMessageNonce - 1))
	return nil
}

// UpdateL1BatchIndexAndStatus updates L1 finalized batch index and status
func (b *EventUpdateLogic) UpdateL1BatchIndexAndStatus(ctx context.Context, height uint64) error {
	finalizedBatches, err := b.batchEventOrm.GetFinalizedBatchesLEBlockHeight(ctx, height)
	if err != nil {
		log.Error("failed to get batches >= block height", "error", err)
		return err
	}

	for _, finalizedBatch := range finalizedBatches {
		log.Info("update finalized batch info of L2 withdrawals", "index", finalizedBatch.BatchIndex, "start", finalizedBatch.StartBlockNumber, "end", finalizedBatch.EndBlockNumber)
		if updateErr := b.updateL2WithdrawMessageInfos(ctx, finalizedBatch.BatchIndex, finalizedBatch.StartBlockNumber, finalizedBatch.EndBlockNumber); updateErr != nil {
			log.Error("failed to update L2 withdraw message infos", "index", finalizedBatch.BatchIndex, "start", finalizedBatch.StartBlockNumber, "end", finalizedBatch.EndBlockNumber, "error", updateErr)
			return updateErr
		}
		if dbErr := b.batchEventOrm.UpdateBatchEventStatus(ctx, finalizedBatch.BatchIndex); dbErr != nil {
			log.Error("failed to update batch event status as updated", "index", finalizedBatch.BatchIndex, "start", finalizedBatch.StartBlockNumber, "end", finalizedBatch.EndBlockNumber, "error", dbErr)
			return dbErr
		}
		b.eventUpdateLogicL1FinalizeBatchEventL2BlockUpdateHeight.Set(float64(finalizedBatch.EndBlockNumber))
	}
	return nil
}

// L2InsertOrUpdate inserts or updates L2 messages
func (b *EventUpdateLogic) L2InsertOrUpdate(ctx context.Context, l2FetcherResult *L2FilterResult) error {
	if txErr := b.crossMessageOrm.InsertOrUpdateL2Messages(ctx, l2FetcherResult.WithdrawMessages); txErr != nil {
		log.Error("failed to insert L2 withdrawal messages", "err", txErr)
		return txErr
	}
	if txErr := b.crossMessageOrm.InsertOrUpdateL2RelayedMessagesOfL1Deposits(ctx, l2FetcherResult.RelayedMessages); txErr != nil {
		log.Error("failed to update L2 relayed messages of L1 deposits", "err", txErr)
		return txErr
	}
	if txErr := b.crossMessageOrm.InsertFailedL2GatewayRouterTxs(ctx, l2FetcherResult.OtherRevertedTxs); txErr != nil {
		log.Error("failed to insert failed L2 gateway router transactions", "err", txErr)
		return txErr
	}
	return nil
}
