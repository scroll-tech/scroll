package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// BlobUpload represents a blob upload record in the database.
type BlobUpload struct {
	db *gorm.DB `gorm:"-"`

	// blob upload
	BatchIndex uint64 `json:"batch_index" gorm:"column:batch_index"`
	BatchHash  string `json:"batch_hash" gorm:"column:batch_hash"`
	Platform   int16  `json:"platform" gorm:"column:platform"`
	Status     int16  `json:"status" gorm:"column:status"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewBlobUpload creates a new BlobUpload database instance.
func NewBlobUpload(db *gorm.DB) *BlobUpload {
	return &BlobUpload{db: db}
}

// TableName returns the table name for the BlobUpload model.
func (*BlobUpload) TableName() string {
	return "blob_upload"
}

// GetNextBatchIndexToUploadByPlatform retrieves the next batch index that hasn't been uploaded to corresponding blob storage service
func (o *BlobUpload) GetNextBatchIndexToUploadByPlatform(ctx context.Context, startBatch uint64, platform types.BlobStoragePlatform) (uint64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})
	db = db.Where("platform = ? AND status = ?", platform, types.BlobUploadStatusUploaded)
	db = db.Order("batch_index DESC")
	db = db.Limit(1)

	var blobUpload BlobUpload
	var batchIndex uint64
	if err := db.First(&blobUpload).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			batchIndex = startBatch
		} else {
			return 0, fmt.Errorf("BlobUpload.GetNextBatchIndexToUploadByPlatform error: %w", err)
		}
	} else {
		batchIndex = blobUpload.BatchIndex + 1
	}

	return batchIndex, nil
}

// GetBlobUpload retrieves the selected blob uploads from the database.
func (o *BlobUpload) GetBlobUploads(ctx context.Context, fields map[string]interface{}, orderByList []string, limit int) ([]*BlobUpload, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&BlobUpload{})

	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit > 0 {
		db = db.Limit(limit)
	}

	db = db.Order("batch_index ASC")

	var blobUploads []*BlobUpload
	if err := db.Find(&blobUploads).Error; err != nil {
		return nil, fmt.Errorf("BlobUpload.GetBlobUploads error: %w", err)
	}

	return blobUploads, nil
}

// InsertOrUpdateBlobUpload inserts a new blob upload record or updates the existing one.
func (o *BlobUpload) InsertOrUpdateBlobUpload(ctx context.Context, batchIndex uint64, batchHash string, platform types.BlobStoragePlatform, status types.BlobUploadStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)

	var existing BlobUpload
	err := db.Where("batch_index = ? AND batch_hash = ? AND platform = ? AND deleted_at IS NULL",
		batchIndex, batchHash, int16(platform),
	).First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		newRecord := BlobUpload{
			BatchIndex: batchIndex,
			BatchHash:  batchHash,
			Platform:   int16(platform),
			Status:     int16(status),
		}
		if err := db.Create(&newRecord).Error; err != nil {
			return fmt.Errorf("BlobUpload.InsertOrUpdateBlobUpload insert error: %w, batch index: %v, batch_hash: %v, platform: %v", err, batchIndex, batchHash, platform)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("BlobUpload.InsertOrUpdateBlobUpload query error: %w, batch index: %v, batch_hash: %v, platform: %v", err, batchIndex, batchHash, platform)
	}

	if err := db.Model(&existing).Update("status", int16(status)).Error; err != nil {
		return fmt.Errorf("BlobUpload.InsertOrUpdateBlobUpload update error: %w, batch index: %v, batch_hash: %v, platform: %v", err, batchIndex, batchHash, platform)
	}

	return nil
}
