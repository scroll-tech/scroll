package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// BatchStatusType represents the type of batch status.
type BatchStatusType int

// Constants for BatchStatusType.
const (
	BatchStatusTypeUnknown BatchStatusType = iota
	BatchStatusTypeCommitted
	BatchStatusTypeReverted
	BatchStatusTypeFinalized
)

// BatchEvent represents a batch event.
type BatchEvent struct {
	db *gorm.DB `gorm:"column:-"`

	ID               uint64     `json:"id" gorm:"column:id;primary_key"`
	BatchStatus      int        `json:"batch_status" gorm:"column:batch_status"`
	BatchIndex       uint64     `json:"batch_index" gorm:"column:batch_index"`
	BatchHash        string     `json:"batch_hash" gorm:"column:batch_hash"`
	StartBlockNumber uint64     `json:"start_block_number" gorm:"column:start_block_number"`
	EndBlockNumber   uint64     `json:"end_block_number" gorm:"column:end_block_number"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

// TableName returns the table name for the BatchEvent model.
func (*BatchEvent) TableName() string {
	return "batch_event"
}

// NewBatchEvent returns a new instance of BatchEvent.
func NewBatchEvent(db *gorm.DB) *BatchEvent {
	return &BatchEvent{db: db}
}

// GetBatchesGEBlockHeight returns the batches with end block >= given block height in db.
func (c *BatchEvent) GetBatchesGEBlockHeight(ctx context.Context, blockHeight uint64) ([]*BatchEvent, error) {
	var batches []*BatchEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BatchEvent{})
	db = db.Where("end_block_number >= ?", blockHeight)
	db = db.Order("batch_index desc")
	if err := db.First(&batches).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get batches >= block height, error: %w", err)
	}
	return batches, nil
}

// GetBatchByIndex returns the batch by index.
func (c *BatchEvent) GetBatchByIndex(ctx context.Context, index uint64) (*BatchEvent, error) {
	var result BatchEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BatchEvent{})
	db = db.Where("batch_index = ?", index)
	if err := db.First(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to get batch by index, error: %w", err)
	}
	return &result, nil
}

// InsertOrUpdateBatchEvents inserts a new batch event or updates an existing one based on the BatchStatusType.
func (c *BatchEvent) InsertOrUpdateBatchEvents(ctx context.Context, l1BatchEvents []*BatchEvent, dbTX ...*gorm.DB) error {
	for _, l1BatchEvent := range l1BatchEvents {
		db := c.db
		if len(dbTX) > 0 && dbTX[0] != nil {
			db = dbTX[0]
		}
		db = db.WithContext(ctx)
		db = db.Model(&CrossMessage{})
		db = db.Model(BatchEvent{})
		updateFields := make(map[string]interface{})
		switch BatchStatusType(l1BatchEvent.BatchStatus) {
		case BatchStatusTypeCommitted:
			if err := db.Create(l1BatchEvent).Error; err != nil {
				return fmt.Errorf("failed to insert batch event, batch: %+v, error: %w", l1BatchEvent, err)
			}
		case BatchStatusTypeFinalized:
			db = db.Where("batch_index = ?", l1BatchEvent.BatchIndex)
			db = db.Where("batch_hash = ?", l1BatchEvent.BatchHash)
			updateFields["batch_status"] = BatchStatusTypeFinalized
			if err := db.Updates(updateFields).Error; err != nil {
				return fmt.Errorf("failed to update batch event, batch: %+v, error: %w", l1BatchEvent, err)
			}
		case BatchStatusTypeReverted:
			db = db.Where("batch_index = ?", l1BatchEvent.BatchIndex)
			db = db.Where("batch_hash = ?", l1BatchEvent.BatchHash)
			updateFields["batch_status"] = BatchStatusTypeReverted
			if err := db.Updates(updateFields).Error; err != nil {
				return fmt.Errorf("failed to update batch event, batch: %+v, error: %w", l1BatchEvent, err)
			}
			// Soft delete the batch event
			if err := db.Delete(l1BatchEvent).Error; err != nil {
				return fmt.Errorf("failed to soft delete batch event, batch: %+v, error: %w", l1BatchEvent, err)
			}
		}
	}
	return nil
}
