package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scroll-tech/bridge-history-api/internal/types"

	btypes "scroll-tech/bridge-history-api/internal/types"
)

// MessageQueueEvent struct represents the details of a batch event.
type MessageQueueEvent struct {
	EventType  btypes.MessageQueueEventType
	QueueIndex uint64

	// Track replay tx hash and refund tx hash.
	TxHash common.Hash

	// QueueTransaction only in replayMessage, to track which message is replayed.
	MessageHash common.Hash
}

// CrossMessage represents a cross message.
type CrossMessage struct {
	db *gorm.DB `gorm:"column:-"`

	ID             uint64     `json:"id" gorm:"column:id;primary_key"`
	MessageType    int        `json:"message_type" gorm:"column:message_type"`
	RollupStatus   int        `json:"rollup_status" gorm:"column:rollup_status"`
	TxStatus       int        `json:"tx_status" gorm:"column:tx_status"`
	TokenType      int        `json:"token_type" gorm:"column:token_type"`
	Sender         string     `json:"sender" gorm:"column:sender"`
	Receiver       string     `json:"receiver" gorm:"column:receiver"`
	MessageHash    string     `json:"message_hash" gorm:"column:message_hash"`
	L1TxHash       string     `json:"l1_tx_hash" gorm:"column:l1_tx_hash"` // initial tx hash, if MessageType is MessageTypeL1SentMessage.
	L1ReplayTxHash string     `json:"l1_replay_tx_hash" gorm:"column:l1_replay_tx_hash"`
	L1RefundTxHash string     `json:"l1_refund_tx_hash" gorm:"column:l1_refund_tx_hash"`
	L2TxHash       string     `json:"l2_tx_hash" gorm:"column:l2_tx_hash"` // initial tx hash, if MessageType is MessageTypeL2SentMessage.
	L1BlockNumber  uint64     `json:"l1_block_number" gorm:"column:l1_block_number"`
	L2BlockNumber  uint64     `json:"l2_block_number" gorm:"column:l2_block_number"`
	L1TokenAddress string     `json:"l1_token_address" gorm:"column:l1_token_address"`
	L2TokenAddress string     `json:"l2_token_address" gorm:"column:l2_token_address"`
	TokenIDs       string     `json:"token_ids" gorm:"column:token_ids"`
	TokenAmounts   string     `json:"token_amounts" gorm:"column:token_amounts"`
	BlockTimestamp uint64     `json:"block_timestamp" gorm:"column:block_timestamp"`
	MessageFrom    string     `json:"message_from" gorm:"column:message_from"`
	MessageTo      string     `json:"message_to" gorm:"column:message_to"`
	MessageValue   string     `json:"message_value" gorm:"column:message_value"`
	MessageNonce   uint64     `json:"message_nonce" gorm:"column:message_nonce"`
	MessageData    string     `json:"message_data" gorm:"column:message_data"`
	MerkleProof    []byte     `json:"merkle_proof" gorm:"column:merkle_proof"`
	BatchIndex     uint64     `json:"batch_index" gorm:"column:batch_index"`
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt      *time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

// TableName returns the table name for the CrossMessage model.
func (*CrossMessage) TableName() string {
	return "cross_message_v2"
}

// NewCrossMessage returns a new instance of CrossMessage.
func NewCrossMessage(db *gorm.DB) *CrossMessage {
	return &CrossMessage{db: db}
}

// GetMessageSyncedHeightInDB returns the latest synced cross message height from the database for a given message type.
func (c *CrossMessage) GetMessageSyncedHeightInDB(ctx context.Context, messageType btypes.MessageType) (uint64, error) {
	var message CrossMessage
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("message_type = ?", messageType)
	switch {
	case messageType == btypes.MessageTypeL1SentMessage:
		db = db.Order("l1_block_number desc")
	case messageType == btypes.MessageTypeL2SentMessage:
		db = db.Order("l2_block_number desc")
	}
	if err := db.First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get latest processed height, type: %v, error: %w", messageType, err)
	}
	switch {
	case messageType == btypes.MessageTypeL1SentMessage:
		return message.L1BlockNumber, nil
	case messageType == btypes.MessageTypeL2SentMessage:
		return message.L2BlockNumber, nil
	default:
		return 0, fmt.Errorf("invalid message type: %v", messageType)
	}
}

// GetL2LatestFinalizedWithdrawal returns the latest finalized L2 withdrawal from the database.
func (c *CrossMessage) GetL2LatestFinalizedWithdrawal(ctx context.Context) (*CrossMessage, error) {
	var message CrossMessage
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("message_type = ?", btypes.MessageTypeL2SentMessage)
	db = db.Where("rollup_status = ?", btypes.RollupStatusTypeFinalized)
	db = db.Order("message_nonce desc")
	if err := db.First(&message).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest L2 finalized sent message event, error: %w", err)
	}
	return &message, nil
}

// GetL2WithdrawalsByBlockRange returns the L2 withdrawals by block range from the database.
func (c *CrossMessage) GetL2WithdrawalsByBlockRange(ctx context.Context, startBlock, endBlock uint64) ([]*CrossMessage, error) {
	var messages []*CrossMessage
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("l2_block_number >= ?", startBlock)
	db = db.Where("l2_block_number <= ?", endBlock)
	db = db.Where("tx_status != ?", types.TxStatusTypeSentTxReverted)
	db = db.Where("message_type = ?", btypes.MessageTypeL2SentMessage)
	db = db.Order("message_nonce asc")
	if err := db.Find(&messages).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get L2 withdrawals by block range, error: %v", err)
	}
	return messages, nil
}

// GetMessagesByTxHashes retrieves all cross messages from the database that match the provided transaction hashes.
func (c *CrossMessage) GetMessagesByTxHashes(ctx context.Context, txHashes []string) ([]*CrossMessage, error) {
	var messages []*CrossMessage
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("l1_tx_hash in (?) or l2_tx_hash in (?)", txHashes, txHashes)
	if err := db.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get L2 messages by tx hashes, tx hashes: %v, error: %w", txHashes, err)
	}
	return messages, nil
}

// GetL2UnclaimedWithdrawalsByAddress retrieves all L2 unclaimed withdrawal messages for a given sender address.
func (c *CrossMessage) GetL2UnclaimedWithdrawalsByAddress(ctx context.Context, sender string) ([]*CrossMessage, error) {
	var messages []*CrossMessage
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("message_type = ?", btypes.MessageTypeL2SentMessage)
	db = db.Where("tx_status = ?", types.TxStatusTypeSent)
	db = db.Where("sender = ?", sender)
	db = db.Order("block_timestamp desc")
	db = db.Limit(500)
	if err := db.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get L2 claimable withdrawal messages by sender address, sender: %v, error: %w", sender, err)
	}
	return messages, nil
}

// GetL2WithdrawalsByAddress retrieves all L2 claimable withdrawal messages for a given sender address.
func (c *CrossMessage) GetL2WithdrawalsByAddress(ctx context.Context, sender string) ([]*CrossMessage, error) {
	var messages []*CrossMessage
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("message_type = ?", btypes.MessageTypeL2SentMessage)
	db = db.Where("sender = ?", sender)
	db = db.Order("block_timestamp desc")
	db = db.Limit(500)
	if err := db.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get L2 withdrawal messages by sender address, sender: %v, error: %w", sender, err)
	}
	return messages, nil
}

// GetTxsByAddress retrieves all txs for a given sender address.
func (c *CrossMessage) GetTxsByAddress(ctx context.Context, sender string) ([]*CrossMessage, error) {
	var messages []*CrossMessage
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("sender = ?", sender)
	db = db.Order("block_timestamp desc")
	db = db.Limit(500)
	if err := db.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get all txs by sender address, sender: %v, error: %w", sender, err)
	}
	return messages, nil
}

// UpdateL1MessageQueueEventsInfo updates the information about L1 message queue events in the database.
func (c *CrossMessage) UpdateL1MessageQueueEventsInfo(ctx context.Context, l1MessageQueueEvents []*MessageQueueEvent) error {
	// update tx statuses.
	for _, l1MessageQueueEvent := range l1MessageQueueEvents {
		db := c.db
		db = db.WithContext(ctx)
		db = db.Model(&CrossMessage{})
		txStatusUpdateFields := make(map[string]interface{})
		switch l1MessageQueueEvent.EventType {
		case btypes.MessageQueueEventTypeQueueTransaction:
			continue
		case btypes.MessageQueueEventTypeDequeueTransaction:
			// do not over-write terminal statuses.
			db = db.Where("tx_status != ?", types.TxStatusTypeRelayed)
			db = db.Where("tx_status != ?", types.TxStatusTypeDropped)
			db = db.Where("message_nonce = ?", l1MessageQueueEvent.QueueIndex)
			db = db.Where("message_type = ?", btypes.MessageTypeL1SentMessage)
			txStatusUpdateFields["tx_status"] = types.TxStatusTypeSkipped
		case btypes.MessageQueueEventTypeDropTransaction:
			// do not over-write terminal statuses.
			db = db.Where("tx_status != ?", types.TxStatusTypeRelayed)
			db = db.Where("tx_status != ?", types.TxStatusTypeDropped)
			db = db.Where("message_nonce = ?", l1MessageQueueEvent.QueueIndex)
			db = db.Where("message_type = ?", btypes.MessageTypeL1SentMessage)
			txStatusUpdateFields["tx_status"] = types.TxStatusTypeDropped
		case btypes.MessageQueueEventTypeResetDequeuedTransaction:
			db = db.Where("tx_status = ?", types.TxStatusTypeSkipped)
			// reset skipped messages that the nonce is greater than or equal to the queue index.
			db = db.Where("message_nonce >= ?", l1MessageQueueEvent.QueueIndex)
			db = db.Where("message_type = ?", btypes.MessageTypeL1SentMessage)
			txStatusUpdateFields["tx_status"] = types.TxStatusTypeSent
		}
		if err := db.Updates(txStatusUpdateFields).Error; err != nil {
			return fmt.Errorf("failed to update tx statuses of L1 message queue events, update fields: %v, error: %w", txStatusUpdateFields, err)
		}
	}

	// update tx hashes of replay and refund.
	for _, l1MessageQueueEvent := range l1MessageQueueEvents {
		db := c.db
		db = db.WithContext(ctx)
		db = db.Model(&CrossMessage{})
		txHashUpdateFields := make(map[string]interface{})
		switch l1MessageQueueEvent.EventType {
		case btypes.MessageQueueEventTypeDequeueTransaction, btypes.MessageQueueEventTypeResetDequeuedTransaction:
			continue
		case btypes.MessageQueueEventTypeQueueTransaction:
			// only replayMessages or enforced txs (whose message hashes would not be found), sendMessages have been filtered out.
			// replayMessage case:
			// First SentMessage in L1: https://sepolia.etherscan.io/tx/0xbee4b631312448fcc2caac86e4dccf0a2ae0a88acd6c5fd8764d39d746e472eb
			// Transaction reverted in L2: https://sepolia.scrollscan.com/tx/0xde6ef307a7da255888aad7a4c40a6b8c886e46a8a05883070bbf18b736cbfb8c
			// replayMessage: https://sepolia.etherscan.io/tx/0xa5392891232bb32d98fcdbaca0d91b4d22ef2755380d07d982eebd47b147ce28
			//
			// Note: update l1_tx_hash if the user calls replayMessage, cannot use queue index here,
			// because in replayMessage, queue index != message nonce.
			// Ref: https://github.com/scroll-tech/scroll/blob/v4.3.44/contracts/src/L1/L1ScrollMessenger.sol#L187-L190
			db = db.Where("message_hash = ?", l1MessageQueueEvent.MessageHash.String())
			txHashUpdateFields["l1_replay_tx_hash"] = l1MessageQueueEvent.TxHash.String()
		case btypes.MessageQueueEventTypeDropTransaction:
			db = db.Where("message_nonce = ?", l1MessageQueueEvent.QueueIndex)
			db = db.Where("message_type = ?", btypes.MessageTypeL1SentMessage)
			txHashUpdateFields["l1_refund_tx_hash"] = l1MessageQueueEvent.TxHash.String()
		}
		if err := db.Updates(txHashUpdateFields).Error; err != nil {
			return fmt.Errorf("failed to update tx hashes of replay and refund in L1 message queue events info, update fields: %v, error: %w", txHashUpdateFields, err)
		}
	}
	return nil
}

// UpdateBatchStatusOfL2Withdrawals updates batch status of L2 withdrawals.
func (c *CrossMessage) UpdateBatchStatusOfL2Withdrawals(ctx context.Context, startBlockNumber, endBlockNumber, batchIndex uint64) error {
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Where("message_type = ?", btypes.MessageTypeL2SentMessage)
	db = db.Where("l2_block_number >= ?", startBlockNumber)
	db = db.Where("l2_block_number <= ?", endBlockNumber)
	updateFields := make(map[string]interface{})
	updateFields["batch_index"] = batchIndex
	updateFields["rollup_status"] = btypes.RollupStatusTypeFinalized
	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("failed to update batch status of L2 sent messages, start: %v, end: %v, index: %v, error: %w", startBlockNumber, endBlockNumber, batchIndex, err)
	}
	return nil
}

// UpdateBatchIndexRollupStatusMerkleProofOfL2Messages updates the batch_index, rollup_status, and merkle_proof fields for a list of L2 cross messages.
func (c *CrossMessage) UpdateBatchIndexRollupStatusMerkleProofOfL2Messages(ctx context.Context, messages []*CrossMessage) error {
	if len(messages) == 0 {
		return nil
	}
	for _, message := range messages {
		updateFields := map[string]interface{}{
			"batch_index":   message.BatchIndex,
			"rollup_status": message.RollupStatus,
			"merkle_proof":  message.MerkleProof,
		}
		db := c.db.WithContext(ctx)
		db = db.Model(&CrossMessage{})
		db = db.Where("message_hash = ?", message.MessageHash)
		if err := db.Updates(updateFields).Error; err != nil {
			return fmt.Errorf("failed to update L2 message with message_hash %s, error: %w", message.MessageHash, err)
		}
	}
	return nil
}

// InsertOrUpdateL1Messages inserts or updates a list of L1 cross messages into the database.
func (c *CrossMessage) InsertOrUpdateL1Messages(ctx context.Context, messages []*CrossMessage) error {
	if len(messages) == 0 {
		return nil
	}
	db := c.db
	db = db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	// 'tx_status' column is not explicitly assigned during the update to prevent a later status from being overwritten back to "sent".
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"sender", "receiver", "token_type", "l1_block_number", "l1_tx_hash", "l1_token_address", "l2_token_address", "token_ids", "token_amounts", "message_type", "block_timestamp", "message_nonce"}),
	})
	if err := db.Create(messages).Error; err != nil {
		return fmt.Errorf("failed to insert message, error: %w", err)
	}
	return nil
}

// InsertOrUpdateL2Messages inserts or updates a list of L2 cross messages into the database.
func (c *CrossMessage) InsertOrUpdateL2Messages(ctx context.Context, messages []*CrossMessage) error {
	if len(messages) == 0 {
		return nil
	}
	db := c.db
	db = db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	// 'tx_status' column is not explicitly assigned during the update to prevent a later status from being overwritten back to "sent".
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"sender", "receiver", "token_type", "l2_block_number", "l2_tx_hash", "l1_token_address", "l2_token_address", "token_ids", "token_amounts", "message_type", "block_timestamp", "message_from", "message_to", "message_value", "message_data", "message_nonce"}),
	})
	if err := db.Create(messages).Error; err != nil {
		return fmt.Errorf("failed to insert message, error: %w", err)
	}
	return nil
}

// InsertFailedL2GatewayTxs inserts a list of transactions that failed to interact with the L2 gateways into the database.
// To resolve unique index confliction, L2 tx hash is used as the MessageHash.
// The OnConflict clause is used to prevent inserting same failed transactions multiple times.
func (c *CrossMessage) InsertFailedL2GatewayTxs(ctx context.Context, messages []*CrossMessage) error {
	if len(messages) == 0 {
		return nil
	}

	for _, message := range messages {
		message.MessageHash = message.L2TxHash
	}

	db := c.db
	db = db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_hash"}},
		DoNothing: true,
	})

	if err := db.Create(&messages).Error; err != nil {
		return fmt.Errorf("failed to insert failed gateway router txs, error: %w", err)
	}
	return nil
}

// InsertFailedL1GatewayTxs inserts a list of transactions that failed to interact with the L1 gateways into the database.
// To resolve unique index confliction, L1 tx hash is used as the MessageHash.
// The OnConflict clause is used to prevent inserting same failed transactions multiple times.
func (c *CrossMessage) InsertFailedL1GatewayTxs(ctx context.Context, messages []*CrossMessage) error {
	if len(messages) == 0 {
		return nil
	}

	for _, message := range messages {
		message.MessageHash = message.L1TxHash
	}

	db := c.db
	db = db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_hash"}},
		DoNothing: true,
	})

	if err := db.Create(&messages).Error; err != nil {
		return fmt.Errorf("failed to insert failed gateway router txs, error: %w", err)
	}
	return nil
}

// InsertOrUpdateL2RelayedMessagesOfL1Deposits inserts or updates the database with a list of L2 relayed messages related to L1 deposits.
func (c *CrossMessage) InsertOrUpdateL2RelayedMessagesOfL1Deposits(ctx context.Context, l2RelayedMessages []*CrossMessage) error {
	if len(l2RelayedMessages) == 0 {
		return nil
	}
	// Deduplicate messages, for each message_hash, retaining message with the highest block number.
	// This is necessary as a single message, like a FailedRelayedMessage or a reverted relayed transaction,
	// may be relayed multiple times within certain block ranges, potentially leading to the error:
	// "ERROR: ON CONFLICT DO UPDATE command cannot affect row a second time (SQLSTATE 21000)".
	// This happens if we attempt to insert multiple records with the same message_hash in a single db.Create operation.
	// For example, see these transactions where the same message was relayed twice within certain block ranges:
	// Reverted tx 1: https://sepolia.scrollscan.com/tx/0xcd6979277c3bc747445273a5e58ef1e9692fbe101d88cfefbbb69d3aef3193c0
	// Reverted tx 2: https://sepolia.scrollscan.com/tx/0x43e28ed7cb71107c18c5d8ebbdb4a1d9cac73e60391d14d41e92985028faa337
	// Another example:
	// FailedRelayedMessage 1: https://sepolia.scrollscan.com/tx/0xfadb147fb211e5096446c5cac3ae0a8a705d2ece6c47c65135c8874f84638f17
	// FailedRelayedMessage 2: https://sepolia.scrollscan.com/tx/0x6cb149b61afd07bf2e17561a59ebebde41e343b6610290c97515b2f862160b42
	mergedL2RelayedMessages := make(map[string]*CrossMessage)
	for _, message := range l2RelayedMessages {
		if existing, found := mergedL2RelayedMessages[message.MessageHash]; found {
			if types.TxStatusType(message.TxStatus) == types.TxStatusTypeRelayed || message.L2BlockNumber > existing.L2BlockNumber {
				mergedL2RelayedMessages[message.MessageHash] = message
			}
		} else {
			mergedL2RelayedMessages[message.MessageHash] = message
		}
	}
	uniqueL2RelayedMessages := make([]*CrossMessage, 0, len(mergedL2RelayedMessages))
	for _, msg := range mergedL2RelayedMessages {
		uniqueL2RelayedMessages = append(uniqueL2RelayedMessages, msg)
	}
	// Do not update tx status of successfully relayed messages,
	// because if a message is handled, the later relayed message tx would be reverted.
	// ref: https://github.com/scroll-tech/scroll/blob/v4.3.44/contracts/src/L2/L2ScrollMessenger.sol#L102
	// e.g.,
	// Successfully relayed: https://sepolia.scrollscan.com/tx/0x4eb7cb07ba76956259c0079819a34a146f8a93dd891dc94812e9b3d66b056ec7#eventlog
	// Reverted tx 1 (Reason: Message was already successfully executed): https://sepolia.scrollscan.com/tx/0x1973cafa14eb40734df30da7bfd4d9aceb53f8f26e09d96198c16d0e2e4a95fd
	// Reverted tx 2 (Reason: Message was already successfully executed): https://sepolia.scrollscan.com/tx/0x02fc3a28684a590aead2482022f56281539085bd3d273ac8dedc1ceccb2bc554
	db := c.db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"message_type", "l2_block_number", "l2_tx_hash", "tx_status"}),
		Where: clause.Where{
			Exprs: []clause.Expression{
				clause.And(
					// do not over-write terminal statuses.
					clause.Neq{Column: "cross_message_v2.tx_status", Value: types.TxStatusTypeRelayed},
					clause.Neq{Column: "cross_message_v2.tx_status", Value: types.TxStatusTypeDropped},
				),
			},
		},
	})
	if err := db.Create(uniqueL2RelayedMessages).Error; err != nil {
		return fmt.Errorf("failed to update L2 reverted relayed message of L1 deposit, error: %w", err)
	}
	return nil
}

// InsertOrUpdateL1RelayedMessagesOfL2Withdrawals inserts or updates the database with a list of L1 relayed messages related to L2 withdrawals.
func (c *CrossMessage) InsertOrUpdateL1RelayedMessagesOfL2Withdrawals(ctx context.Context, l1RelayedMessages []*CrossMessage) error {
	if len(l1RelayedMessages) == 0 {
		return nil
	}
	// Deduplicate messages, for each message_hash, retaining message with the highest block number.
	// This is necessary as a single message, like a FailedRelayedMessage or a reverted relayed transaction,
	// may be relayed multiple times within certain block ranges, potentially leading to the error:
	// "ERROR: ON CONFLICT DO UPDATE command cannot affect row a second time (SQLSTATE 21000)".
	// This happens if we attempt to insert multiple records with the same message_hash in a single db.Create operation.
	// For example, see these transactions where the same message was relayed twice within certain block ranges:
	// FailedRelayedMessage 1: https://sepolia.etherscan.io/tx/0x28b3212cda6ca0f3790f362a780257bbe2b37417ccf75a4eca6c3a08294c8f1b#eventlog
	// FailedRelayedMessage 2: https://sepolia.etherscan.io/tx/0xc8a8254825dd2cab5caef58cfd8d88c077ceadadc78f2340214a86cf8ab88543#eventlog
	// Another example (relayed success, then relayed again):
	// Relay Message, and success: https://sepolia.etherscan.io/tx/0xcfdf2f5446719e3e123a8aa06e4d6b3809c3850a13adf875755c8b1e423aa448#eventlog
	// Relay Message again, and reverted: https://sepolia.etherscan.io/tx/0xb1fcae7546f3de4cfd0b4d679f4075adb4eb69578b12e2b5673f5f24b1836578
	mergedL1RelayedMessages := make(map[string]*CrossMessage)
	for _, message := range l1RelayedMessages {
		if existing, found := mergedL1RelayedMessages[message.MessageHash]; found {
			if types.TxStatusType(message.TxStatus) == types.TxStatusTypeRelayed || message.L1BlockNumber > existing.L1BlockNumber {
				mergedL1RelayedMessages[message.MessageHash] = message
			}
		} else {
			mergedL1RelayedMessages[message.MessageHash] = message
		}
	}
	uniqueL1RelayedMessages := make([]*CrossMessage, 0, len(mergedL1RelayedMessages))
	for _, msg := range mergedL1RelayedMessages {
		uniqueL1RelayedMessages = append(uniqueL1RelayedMessages, msg)
	}
	db := c.db
	db = db.WithContext(ctx)
	db = db.Model(&CrossMessage{})
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"message_type", "l1_block_number", "l1_tx_hash", "tx_status"}),
		Where: clause.Where{
			Exprs: []clause.Expression{
				clause.And(
					// do not over-write terminal statuses.
					clause.Neq{Column: "cross_message_v2.tx_status", Value: types.TxStatusTypeRelayed},
					clause.Neq{Column: "cross_message_v2.tx_status", Value: types.TxStatusTypeDropped},
				),
			},
		},
	})
	if err := db.Create(uniqueL1RelayedMessages).Error; err != nil {
		return fmt.Errorf("failed to update L1 relayed message of L2 withdrawal, error: %w", err)
	}
	return nil
}
