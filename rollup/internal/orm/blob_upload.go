package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// BlobUpload represents a blob upload record in the database.
type BlobUpload struct {
	db *gorm.DB `gorm:"-"`

	BatchIndex uint64    `json:"batch_index" gorm:"column:batch_index;primaryKey"`
	Platform   string    `json:"platform" gorm:"column:platform;primaryKey"`
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
func (o *BlobUpload) InsertBlobUpload(ctx context.Context, batchIndex uint64, platform string, status types.BlobUploadStatus) error {
	blobUpload := &BlobUpload{
		BatchIndex: batchIndex,
		Platform:   platform,
		Status:     int16(status),
		UpdatedAt:  time.Now(),
	}

	db := o.db.WithContext(ctx)
	if err := db.Create(blobUpload).Error; err != nil {
		return fmt.Errorf("BlobUpload.InsertBlobUpload error: %w, batch index: %v, platform: %v", err, batchIndex, platform)
	}
	return nil
}

// UpdateBlobUploadStatus updates the status of a blob upload record.
func (o *BlobUpload) UpdateBlobUploadStatus(ctx context.Context, batchIndex uint64, platform string, status types.BlobUploadStatus) error {
	db := o.db.WithContext(ctx)
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

// GetBlobUploadByBatchIndexAndPlatform retrieves a blob upload record by batch index and platform.
func (o *BlobUpload) GetBlobUploadByBatchIndexAndPlatform(ctx context.Context, batchIndex uint64, platform string) (*BlobUpload, error) {
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

// GetPendingBlobUploads retrieves all pending blob upload records.
func (o *BlobUpload) GetPendingBlobUploads(ctx context.Context) ([]*BlobUpload, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("status = ?", types.BlobUploadStatusPending)
	db = db.Order("batch_index ASC")

	var blobUploads []*BlobUpload
	if err := db.Find(&blobUploads).Error; err != nil {
		return nil, fmt.Errorf("BlobUpload.GetPendingBlobUploads error: %w", err)
	}
	return blobUploads, nil
}

// GetFailedBlobUploads retrieves all failed blob upload records.
func (o *BlobUpload) GetFailedBlobUploads(ctx context.Context) ([]*BlobUpload, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("status = ?", types.BlobUploadStatusFailed)
	db = db.Order("batch_index ASC")

	var blobUploads []*BlobUpload
	if err := db.Find(&blobUploads).Error; err != nil {
		return nil, fmt.Errorf("BlobUpload.GetFailedBlobUploads error: %w", err)
	}
	return blobUploads, nil
}

// DeleteBlobUpload deletes a blob upload record.
func (o *BlobUpload) DeleteBlobUpload(ctx context.Context, batchIndex uint64, platform string) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("batch_index = ? AND platform = ?", batchIndex, platform)

	if err := db.Delete(&BlobUpload{}).Error; err != nil {
		return fmt.Errorf("BlobUpload.DeleteBlobUpload error: %w, batch index: %v, platform: %v", err, batchIndex, platform)
	}
	return nil
}
