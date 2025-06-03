package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scroll-tech/common/types"
)

// BlobUpload represents a blob upload record in the database.
type BlobUpload struct {
	db *gorm.DB `gorm:"-"`

	BatchIndex uint64    `json:"batch_index" gorm:"column:batch_index;primaryKey"`
	Platform   int16     `json:"platform" gorm:"column:platform;primaryKey"`
	Status     int16     `json:"status" gorm:"column:status"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// NewBlobUpload creates a new BlobUpload database instance.
func NewBlobUpload(db *gorm.DB) *BlobUpload {
	return &BlobUpload{db: db}
}

// TableName returns the table name for the BlobUpload model.
func (*BlobUpload) TableName() string {
	return "blob_upload"
}

// InsertBlobUpload inserts a new blob upload record into the database.
func (o *BlobUpload) InsertBlobUpload(ctx context.Context, batchIndex uint64, platform types.BlobStoragePlatform, status types.BlobUploadStatus, dbTX ...*gorm.DB) error {
	blobUpload := &BlobUpload{
		BatchIndex: batchIndex,
		Platform:   int16(platform),
		Status:     int16(status),
		UpdatedAt:  time.Now(),
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	if err := db.Create(blobUpload).Error; err != nil {
		return fmt.Errorf("BlobUpload.InsertBlobUpload error: %w, batch index: %v, platform: %v", err, batchIndex, platform)
	}
	return nil
}

// UpdateBlobUploadStatus updates the status of a blob upload record.
func (o *BlobUpload) UpdateBlobUploadStatus(ctx context.Context, batchIndex uint64, platform types.BlobStoragePlatform, status types.BlobUploadStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("batch_index = ? AND platform = ?", batchIndex, platform)

	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if err := db.Updates(updates).Error; err != nil {
		return fmt.Errorf("BlobUpload.UpdateBlobUploadStatus error: %w, batch index: %v, platform: %v", err, batchIndex, platform)
	}
	return nil
}

// InsertOrUpdateBlobUpload inserts a new blob upload record or updates the existing one.
func (o *BlobUpload) InsertOrUpdateBlobUpload(ctx context.Context, batchIndex uint64, platform types.BlobStoragePlatform, status types.BlobUploadStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	blobUpload := &BlobUpload{
		BatchIndex: batchIndex,
		Platform:   int16(platform),
		Status:     int16(status),
		UpdatedAt:  time.Now(),
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "batch_index"}, {Name: "platform"}},
		DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
	}).Create(blobUpload).Error; err != nil {
		return fmt.Errorf("BlobUpload.InsertOrUpdateBlobUpload error: %w, batch index: %v, platform: %v", err, batchIndex, platform)
	}
	return nil
}

// GetBlobUploadByBatchIndexAndPlatform retrieves a blob upload record by batch index and platform.
func (o *BlobUpload) GetBlobUploadByBatchIndexAndPlatform(ctx context.Context, batchIndex uint64, platform types.BlobStoragePlatform) (*BlobUpload, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("batch_index = ? AND platform = ?", batchIndex, platform)

	var blobUpload BlobUpload
	if err := db.First(&blobUpload).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("BlobUpload.GetBlobUploadByBatchIndexAndPlatform error: %w, batch index: %v, platform: %v", err, batchIndex, platform)
	}
	return &blobUpload, nil
}

// GetPendingBlobUploadsByPlatform retrieves all pending blob upload records by platform.
func (o *BlobUpload) GetPendingBlobUploadsByPlatform(ctx context.Context, platform types.BlobStoragePlatform) ([]*BlobUpload, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("status = ? AND platform = ?", types.BlobUploadStatusPending, platform)
	db = db.Order("batch_index ASC")

	var blobUploads []*BlobUpload
	if err := db.Find(&blobUploads).Error; err != nil {
		return nil, fmt.Errorf("BlobUpload.GetPendingBlobUploadsByPlatform error: %w", err)
	}
	return blobUploads, nil
}

// GetFailedBlobUploadsByPlatform retrieves all failed blob upload records by platform.
func (o *BlobUpload) GetFailedBlobUploadsByPlatform(ctx context.Context, platform types.BlobStoragePlatform) ([]*BlobUpload, error) {

	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("status = ? AND platform = ?", types.BlobUploadStatusFailed, platform)
	db = db.Order("batch_index ASC")

	var blobUploads []*BlobUpload
	if err := db.Find(&blobUploads).Error; err != nil {
		return nil, fmt.Errorf("BlobUpload.GetFailedBlobUploadsByPlatform error: %w", err)
	}
	return blobUploads, nil
}
