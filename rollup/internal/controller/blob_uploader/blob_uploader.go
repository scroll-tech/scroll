package blob_uploader

import (
	"context"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"

	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
)

// BlobUploader is responsible for uploading blobs to blob storage services.
type BlobUploader struct {
	ctx context.Context

	cfg *config.BlobUploaderConfig

	s3Uploader *S3Uploader

	blobUploadOrm *orm.BlobUpload
	batchOrm      *orm.Batch
	chunkOrm      *orm.Chunk
	l2BlockOrm    *orm.L2Block

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
		ctx:           ctx,
		cfg:           cfg,
		s3Uploader:    s3Uploader,
		batchOrm:      orm.NewBatch(db),
		chunkOrm:      orm.NewChunk(db),
		l2BlockOrm:    orm.NewL2Block(db),
		blobUploadOrm: orm.NewBlobUpload(db),
	}

	blobUploader.metrics = initblobUploaderMetrics(reg)

	return blobUploader, nil
}

func (b *BlobUploader) UploadBlobToS3() {
	log.Info("Try to UploadBlobToS3")
	// get un-uploaded batches from database in ascending order by their index.
	dbBatch, err := b.batchOrm.GetFirstUnuploadedAndFailedBatch(b.ctx, b.cfg.StartBatch, types.BlobStoragePlatformS3)
	if err != nil {
		log.Error("Failed to fetch unuploaded batch", "err", err)
		return
	}

	// nothing to do if we don't have any pending batches
	if dbBatch == nil {
		return
	}

	// construct blob
	var blob *kzg4844.Blob
	codecVersion := encoding.CodecVersion(dbBatch.CodecVersion)
	switch codecVersion {
	case encoding.CodecV7:
		blob, err = b.constructBlobCodecV7(dbBatch)
		if err != nil {
			log.Error("failed to construct constructBlobCodecV7 payload for V7", "codecVersion", codecVersion, "batch index", dbBatch.Index, "err", err)
			return
		}
	default:
		log.Error("unsupported codec version in UploadBlobToS3", "codecVersion", codecVersion, "batch index", dbBatch.Index)
		return
	}

	// calculate versioned blob hash
	versionedBlobHash, err := utils.CalculateVersionedBlobHash(*blob)
	if err != nil {
		log.Error("failed to calculate versioned blob hash", "batch index", dbBatch.Index, "err", err)
		return
	}

	// upload blob data to s3 bucket
	key := common.Bytes2Hex(versionedBlobHash[:])
	err = b.s3Uploader.UploadData(b.ctx, blob[:], key)
	if err != nil {
		log.Error("failed to upload blob data to AWS S3", "batch index", dbBatch.Index, "versioned blob hash", key, "err", err)
		// Update status to failed
		if err = b.blobUploadOrm.InsertOrUpdateBlobUpload(b.ctx, dbBatch.Index, types.BlobStoragePlatformS3, types.BlobUploadStatusFailed); err != nil {
			log.Error("failed to update blob upload status to failed", "batch index", dbBatch.Index, "err", err)
		}
		return
	}

	// Update status to uploaded
	if err = b.blobUploadOrm.InsertOrUpdateBlobUpload(b.ctx, dbBatch.Index, types.BlobStoragePlatformS3, types.BlobUploadStatusUploaded); err != nil {
		log.Error("failed to update blob upload status to uploaded", "batch index", dbBatch.Index, "err", err)
		return
	}

	b.metrics.rollupBlobUploaderUploadToS3Total.Inc()
	log.Info("Successfully uploaded blob to S3", "batch index", dbBatch.Index, "versioned blob hash", key)
}

func (b *BlobUploader) constructBlobCodecV7(dbBatch *orm.Batch) (*kzg4844.Blob, error) {
	var dbChunks []*orm.Chunk

	// Verify batches compatibility
	dbChunks, err := b.chunkOrm.GetChunksInRange(b.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks in range: %v", err)
	}

	// check codec version
	var batchBlocks []*encoding.Block
	for _, dbChunk := range dbChunks {
		if dbBatch.CodecVersion != dbChunk.CodecVersion {
			return nil, fmt.Errorf("batch codec version is different from chunk codec version, batch index: %d, chunk index: %d, batch codec version: %d, chunk codec version: %d", dbBatch.Index, dbChunk.Index, dbBatch.CodecVersion, dbChunk.CodecVersion)
		}

		blocks, err := b.l2BlockOrm.GetL2BlocksInRange(b.ctx, dbChunk.StartBlockNumber, dbChunk.EndBlockNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get blocks in range for batch %d: %w", dbBatch.Index, err)
		}

		batchBlocks = append(batchBlocks, blocks...)
	}

	encodingBatch := &encoding.Batch{
		Index:                  dbBatch.Index,
		ParentBatchHash:        common.HexToHash(dbBatch.ParentBatchHash),
		PrevL1MessageQueueHash: common.HexToHash(dbBatch.PrevL1MessageQueueHash),
		PostL1MessageQueueHash: common.HexToHash(dbBatch.PostL1MessageQueueHash),
		Blocks:                 batchBlocks,
	}

	version := encoding.CodecVersion(dbBatch.CodecVersion)
	codec, err := encoding.CodecFromVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version %d, err: %w", dbBatch.CodecVersion, err)
	}

	daBatch, err := codec.NewDABatch(encodingBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to create DA batch: %w", err)
	}

	return daBatch.Blob(), nil

}
