package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// UpdateStatusType represents the whether batch info is updated in message table.
type UpdateStatusType int

// Constants for UpdateStatusType.
const (
	UpdateStatusTypeUnupdated UpdateStatusType = iota
	UpdateStatusTypeUpdated
)

// BatchEvent represents a batch event.
type BatchEvent struct {
	db *gorm.DB `gorm:"column:-"`

	ID               uint64     `json:"id" gorm:"column:id;primary_key"`
	L1BlockNumber    uint64     `json:"l1_block_number" gorm:"column:l1_block_number"`
	BatchStatus      int        `json:"batch_status" gorm:"column:batch_status"`
	BatchIndex       uint64     `json:"batch_index" gorm:"column:batch_index"`
	BatchHash        string     `json:"batch_hash" gorm:"column:batch_hash"`
	StartBlockNumber uint64     `json:"start_block_number" gorm:"column:start_block_number"`
	EndBlockNumber   uint64     `json:"end_block_number" gorm:"column:end_block_number"`
	UpdateStatus     int        `json:"update_status" gorm:"column:update_status"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

// TableName returns the table name for the BatchEvent model.
func (*BatchEvent) TableName() string {
	return "batch_event_v2"
}

// NewBatchEvent returns a new instance of BatchEvent.
func NewBatchEvent(db *gorm.DB) *BatchEvent {
	return &BatchEvent{db: db}
}

// GetBatchEventSyncedHeightInDB returns the maximum l1_block_number from the batch_event_v2 table.
func (c *BatchEvent) GetBatchEventSyncedHeightInDB(ctx context.Context) (uint64, error) {
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

// GetFinalizedBatchesLEBlockHeight returns the finalized batches with end block <= given block height in db.
func (c *BatchEvent) GetFinalizedBatchesLEBlockHeight(ctx context.Context, blockHeight uint64) ([]*BatchEvent, error) {
	var batches []*BatchEvent
	db := c.db.WithContext(ctx)
	db = db.Model(&BatchEvent{})
	db = db.Where("end_block_number <= ?", blockHeight)
	db = db.Where("batch_status = ?", BatchStatusTypeFinalized)
	db = db.Where("update_status = ?", UpdateStatusTypeUnupdated)
	db = db.Order("batch_index asc")
	if err := db.Find(&batches).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get batches >= block height, error: %w", err)
	}
	return batches, nil
}

// InsertOrUpdateBatchEvents inserts a new batch event or updates an existing one based on the BatchStatusType.
func (c *BatchEvent) InsertOrUpdateBatchEvents(ctx context.Context, l1BatchEvents []*BatchEvent) error {
	for _, l1BatchEvent := range l1BatchEvents {
		db := c.db
		db = db.WithContext(ctx)
		db = db.Model(&BatchEvent{})
		updateFields := make(map[string]interface{})
		switch BatchStatusType(l1BatchEvent.BatchStatus) {
		case BatchStatusTypeCommitted:
			// Use the clause to either insert or ignore on conflict
			onConflict := clause.OnConflict{
				Columns:   []clause.Column{{Name: "batch_hash"}},
				DoNothing: true,
			}
			db = db.Clauses(onConflict)
			if err := db.Create(l1BatchEvent).Error; err != nil {
				return fmt.Errorf("failed to insert or ignore batch event, error: %w", err)
			}
		case BatchStatusTypeFinalized:
			db = db.Where("batch_index = ?", l1BatchEvent.BatchIndex)
			db = db.Where("batch_hash = ?", l1BatchEvent.BatchHash)
			updateFields["batch_status"] = BatchStatusTypeFinalized
			if err := db.Updates(updateFields).Error; err != nil {
				return fmt.Errorf("failed to update batch event, error: %w", err)
			}
		case BatchStatusTypeReverted:
			db = db.Where("batch_index = ?", l1BatchEvent.BatchIndex)
			db = db.Where("batch_hash = ?", l1BatchEvent.BatchHash)
			updateFields["batch_status"] = BatchStatusTypeReverted
			if err := db.Updates(updateFields).Error; err != nil {
				return fmt.Errorf("failed to update batch event, error: %w", err)
			}
			// Soft delete the batch event.
			if err := db.Delete(l1BatchEvent).Error; err != nil {
				return fmt.Errorf("failed to soft delete batch event, error: %w", err)
			}
		}
	}
	return nil
}

// UpdateBatchEventStatus updates the UpdateStatusType of a BatchEvent given its batch index.
func (c *BatchEvent) UpdateBatchEventStatus(ctx context.Context, batchIndex uint64) error {
	db := c.db.WithContext(ctx)
	db = db.Model(&BatchEvent{})
	db = db.Where("batch_index = ?", batchIndex)
	updateFields := map[string]interface{}{
		"update_status": UpdateStatusTypeUpdated,
	}
	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("failed to update batch event status, batchIndex: %d, error: %w", batchIndex, err)
	}
	return nil
}
