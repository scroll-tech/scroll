package blob_uploader

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

// BlobUploader is responsible for uploading blobs to blob storage services.
type BlobUploader struct {
	ctx context.Context

	cfg *config.BlobUploaderConfig

	s3Uploader *S3Uploader
	batchOrm   *orm.Batch

	metrics *blobUploaderMetrics
}

// NewBlobUploader will return a new instance of BlobUploader
func NewBlobUploader(ctx context.Context, db *gorm.DB, cfg *config.BlobUploaderConfig, reg prometheus.Registerer) (*BlobUploader, error) {
	var s3Uploader *S3Uploader
	var err error
	if cfg.AWSS3Config != nil {
		s3Uploader, err = NewS3Uploader(cfg.AWSS3Config)
		if err != nil {
			return nil, fmt.Errorf("new blob uploader failed, err: %w", err)
		}
	}

	blobUploader := &BlobUploader{
		ctx:        ctx,
		cfg:        cfg,
		s3Uploader: s3Uploader,
		batchOrm:   orm.NewBatch(db),
	}

	blobUploader.metrics = initblobUploaderMetrics(reg)

	return blobUploader, nil
}

func (b *BlobUploader) UploadBlobToS3() {
	// get un-uploaded batches from database in ascending order by their index.
	dbBatches, err := b.batchOrm.GetFirstUnuploadedAndFailedBatch(b.ctx, types.BlobStoragePlatformS3)
	if err != nil {
		log.Error("Failed to fetch unuploaded batch", "err", err)
		return
	}

	// nothing to do if we don't have any pending batches
	if dbBatches == nil {
		return
	}

	// upload data to s3 bucket
	b.s3Uploader.UploadData(b.ctx, )
}

