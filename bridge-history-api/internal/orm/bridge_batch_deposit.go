package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
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
	L2TxHash       string     `json:"l2_tx_hash" gorm:"column:l2_tx_hash"`
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

// GetBatchEventSyncedHeightInDB returns the maximum l1_block_number from the batch_event_v2 table.
func (c *BridgeBatchDepositEvent) GetBatchEventSyncedHeightInDB(ctx context.Context) (uint64, error) {
	var batch BatchEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BatchEvent{})
	db = db.Order("l1_block_number desc")
	if err := db.First(&batch).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get batch synced height in db, error: %w", err)
	}
	return batch.L1BlockNumber, nil
}

// InsertBridgeBatchDepositEvent inserts a new BridgeBatchDepositEvent
func (c *BridgeBatchDepositEvent) InsertBridgeBatchDepositEvent(ctx context.Context, l1BatchDepositEvents []*BridgeBatchDepositEvent) error {
	for _, l1BatchEvent := range l1BatchDepositEvents {
		db := c.db
		db = db.WithContext(ctx)
		db = db.Model(&BridgeBatchDepositEvent{})
		if err := db.Create(l1BatchEvent).Error; err != nil {
			return fmt.Errorf("failed to InsertBridgeBatchDepositEvent, error: %w", err)
		}
	}
	return nil
}

// UpdateBatchEventStatus updates the tx_status of BridgeBatchDepositEvent given batch index
func (c *BridgeBatchDepositEvent) UpdateBatchEventStatus(ctx context.Context, batchIndex uint64) error {
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Where("batch_index = ?", batchIndex)
	updateFields := map[string]interface{}{
		"tx_status": types.TxStatusBridgeBatchDistribute,
	}
	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("failed to UpdateBatchEventStatus, batchIndex: %d, error: %w", batchIndex, err)
	}
	return nil
}

// UpdateDistributeFailedStatus updates the tx_status of BridgeBatchDepositEvent given batch index and senders
func (c *BridgeBatchDepositEvent) UpdateDistributeFailedStatus(ctx context.Context, batchIndex uint64, senders []string) error {
	db := c.db.WithContext(ctx)
	db = db.Model(&BridgeBatchDepositEvent{})
	db = db.Where("batch_index = ?", batchIndex)
	db = db.Where("senders in (?)", senders)
	updateFields := map[string]interface{}{
		"tx_status": types.TxStatusBridgeBatchDistributeFailed,
	}
	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("failed to UpdateDistributeFailedStatus, batchIndex: %d, senders:%v, error: %w", batchIndex, senders, err)
	}
	return nil
}
