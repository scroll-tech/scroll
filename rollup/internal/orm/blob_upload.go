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

	// blob upload
	BatchIndex uint64 `json:"batch_index" gorm:"column:batch_index;primaryKey"`
	Platform   int16  `json:"platform" gorm:"column:platform;primaryKey"`
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
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "batch_index"}, {Name: "platform"}},
		DoUpdates: clause.AssignmentColumns([]string{"status"}),
	}).Create(blobUpload).Error; err != nil {
		return fmt.Errorf("BlobUpload.InsertOrUpdateBlobUpload error: %w, batch index: %v, platform: %v", err, batchIndex, platform)
	}
	return nil
}
