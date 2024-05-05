package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scroll-tech/bridge-history-api/internal/types"
)

// BridgeBatchDepositEvent represents the bridge batch deposit event.
type BridgeBatchDepositEvent struct {
	db *gorm.DB `gorm:"column:-"`

	ID             uint64     `json:"id" gorm:"column:id;primary_key"`
	TokenType      int        `json:"token_type" gorm:"column:token_type"`
	Sender         string     `json:"sender" gorm:"column:sender"`
	BatchIndex     uint64     `json:"batch_index" gorm:"column:batch_index"`
	TokenAmount    string     `json:"token_amount" gorm:"column:token_amount"`
	Fee            string     `json:"fee" gorm:"column:fee"`
	L1TokenAddress string     `json:"l1_token_address" gorm:"column:l1_token_address"`
	L2TokenAddress string     `json:"l2_token_address" gorm:"column:l2_token_address"`
	L1BlockNumber  uint64     `json:"l1_block_number" gorm:"column:l1_block_number"`
	L2BlockNumber  uint64     `json:"l2_block_number" gorm:"column:l2_block_number"`
	L1TxHash       string     `json:"l1_tx_hash" gorm:"column:l1_tx_hash"`
	L1LogIndex     uint       `json:"l1_log_index" gorm:"l1_log_index"`
	L2TxHash       string     `json:"l2_tx_hash" gorm:"column:l2_tx_hash"`
	L2LogIndex     uint       `json:"l2_log_index" gorm:"l2_log_index"`
	TxStatus       int        `json:"tx_status" gorm:"column:tx_status"`
	BlockTimestamp uint64     `json:"block_timestamp" gorm:"column:block_timestamp"`
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt      *time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

// TableName returns the table name for the BridgeBatchDepositEvent model.
func (*BridgeBatchDepositEvent) TableName() string {
	return "bridge_batch_deposit_events_v2"
}

// NewBridgeBatchDepositEvent returns a new instance of BridgeBatchDepositEvent.
func NewBridgeBatchDepositEvent(db *gorm.DB) *BridgeBatchDepositEvent {
	return &BridgeBatchDepositEvent{db: db}
}

// GetTxsByAddress returns the txs by address
func (c *BridgeBatchDepositEvent) GetTxsByAddress(ctx context.Context, sender string) ([]*BridgeBatchDepositEvent, error) {
	var messages []*BridgeBatchDepositEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Where("sender = ?", sender)
	db = db.Order("block_timestamp desc")
	db = db.Limit(500)
	if err := db.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to get all txs by sender address, sender: %v, error: %w", sender, err)
	}
	return messages, nil
}

// GetMessagesByTxHashes retrieves all BridgeBatchDepositEvent from the database that match the provided transaction hashes.
func (c *BridgeBatchDepositEvent) GetMessagesByTxHashes(ctx context.Context, txHashes []string) ([]*BridgeBatchDepositEvent, error) {
	var messages []*BridgeBatchDepositEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Where("l1_tx_hash in (?)", txHashes)
	if err := db.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("failed to GetMessagesByTxHashes by tx hashes, tx hashes: %v, error: %w", txHashes, err)
	}
	return messages, nil
}

// GetMessageL1SyncedHeightInDB returns the l1 latest bridge batch deposit message height from the database
func (c *BridgeBatchDepositEvent) GetMessageL1SyncedHeightInDB(ctx context.Context) (uint64, error) {
	var message BridgeBatchDepositEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Order("l1_block_number desc")

	err := db.First(&message).Error
	if err != nil && errors.Is(gorm.ErrRecordNotFound, err) {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get l1 latest processed height, error: %w", err)
	}

	return message.L1BlockNumber, nil
}

// GetMessageL2SyncedHeightInDB returns the l2 latest bridge batch deposit message height from the database
func (c *BridgeBatchDepositEvent) GetMessageL2SyncedHeightInDB(ctx context.Context) (uint64, error) {
	var message BridgeBatchDepositEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Order("l2_block_number desc")

	err := db.First(&message).Error
	if err != nil && errors.Is(gorm.ErrRecordNotFound, err) {
		return 0, nil
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get l2 latest processed height, error: %w", err)
	}

	return message.L2BlockNumber, nil
}

// InsertOrUpdateL1BridgeBatchDepositEvent inserts or updates a new L1 BridgeBatchDepositEvent
func (c *BridgeBatchDepositEvent) InsertOrUpdateL1BridgeBatchDepositEvent(ctx context.Context, l1BatchDepositEvents []*BridgeBatchDepositEvent) error {
	if len(l1BatchDepositEvents) == 0 {
		return nil
	}

	db := c.db
	db = db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "l1_tx_hash"}, {Name: "l1_log_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"token_amount", "fee", "l1_block_number", "l1_token_address", "tx_status", "block_timestamp"}),
	})
	if err := db.Create(l1BatchDepositEvents).Error; err != nil {
		return fmt.Errorf("failed to insert message, error: %w", err)
	}
	return nil
}

// InsertOrUpdateL2BridgeBatchDepositEvent inserts or updates a new L2 BridgeBatchDepositEvent
func (c *BridgeBatchDepositEvent) InsertOrUpdateL2BridgeBatchDepositEvent(ctx context.Context, l1BatchDepositEvents []*BridgeBatchDepositEvent) error {
	if len(l1BatchDepositEvents) == 0 {
		return nil
	}

	db := c.db
	db = db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "l1_tx_hash"}, {Name: "l1_log_index"}},
		DoUpdates: clause.AssignmentColumns([]string{"token_amount", "fee", "l1_block_number", "l1_token_address", "tx_status", "block_timestamp"}),
	})
	if err := db.Create(l1BatchDepositEvents).Error; err != nil {
		return fmt.Errorf("failed to insert message, error: %w", err)
	}
	return nil
}

// UpdateBatchEventStatus updates the tx_status of BridgeBatchDepositEvent given batch index
func (c *BridgeBatchDepositEvent) UpdateBatchEventStatus(ctx context.Context, distributeMessage BridgeBatchDepositEvent) error {
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Where("batch_index = ?", distributeMessage.BatchIndex)
	updateFields := map[string]interface{}{
		"l2_token_address": distributeMessage.L2TokenAddress,
		"l2_block_number":  distributeMessage.L2BlockNumber,
		"l2_tx_hash":       distributeMessage.L2TxHash,
		"l2_log_index":     distributeMessage.L2LogIndex,
		"tx_status":        types.TxStatusBridgeBatchDistribute,
	}
	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("failed to UpdateBatchEventStatus, batchIndex: %d, error: %w", distributeMessage.BatchIndex, err)
	}
	return nil
}

// UpdateDistributeFailedStatus updates the tx_status of BridgeBatchDepositEvent given batch index and senders
func (c *BridgeBatchDepositEvent) UpdateDistributeFailedStatus(ctx context.Context, batchIndex uint64, senders []string) error {
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Where("batch_index = ?", batchIndex)
	db = db.Where("sender in (?)", senders)
	updateFields := map[string]interface{}{
		"tx_status": types.TxStatusBridgeBatchDistributeFailed,
	}
	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("failed to UpdateDistributeFailedStatus, batchIndex: %d, senders:%v, error: %w", batchIndex, senders, err)
	}
	return nil
}
