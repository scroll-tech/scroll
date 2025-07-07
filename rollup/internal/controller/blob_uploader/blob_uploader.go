package blob_uploader

import (
	"context"
	"errors"
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

	blobUploader.metrics = initBlobUploaderMetrics(reg)

	return blobUploader, nil
}

func (b *BlobUploader) UploadBlobToS3() {
	// skip upload if s3 uploader is not configured
	if b.s3Uploader == nil {
		return
	}

	// get un-uploaded batches from database in ascending order by their index.
	dbBatch, err := b.GetFirstUnuploadedBatchByPlatform(b.ctx, b.cfg.StartBatch, types.BlobStoragePlatformS3)
	if err != nil {
		log.Error("Failed to fetch unuploaded batch", "err", err)
		return
	}

	// nothing to do if we don't have any pending batches
	if dbBatch == nil {
		log.Debug("no pending batches to upload")
		return
	}

	// construct blob
	codecVersion := encoding.CodecVersion(dbBatch.CodecVersion)
	blob, err := b.constructBlobCodec(dbBatch)
	if err != nil {
		log.Error("failed to construct constructBlobCodec payload ", "codecVersion", codecVersion, "batch index", dbBatch.Index, "err", err)
		b.metrics.rollupBlobUploaderUploadToS3FailedTotal.Inc()
		if updateErr := b.blobUploadOrm.InsertOrUpdateBlobUpload(b.ctx, dbBatch.Index, dbBatch.Hash, types.BlobStoragePlatformS3, types.BlobUploadStatusFailed); updateErr != nil {
			log.Error("failed to update blob upload status to failed", "batch index", dbBatch.Index, "err", updateErr)
		}
		return
	}

	// calculate versioned blob hash
	versionedBlobHash, err := utils.CalculateVersionedBlobHash(*blob)
	if err != nil {
		log.Error("failed to calculate versioned blob hash", "batch index", dbBatch.Index, "err", err)
		b.metrics.rollupBlobUploaderUploadToS3FailedTotal.Inc()
		// update status to failed
		if updateErr := b.blobUploadOrm.InsertOrUpdateBlobUpload(b.ctx, dbBatch.Index, dbBatch.Hash, types.BlobStoragePlatformS3, types.BlobUploadStatusFailed); updateErr != nil {
			log.Error("failed to update blob upload status to failed", "batch index", dbBatch.Index, "err", updateErr)
		}
		return
	}

	// upload blob data to s3 bucket
	key := common.BytesToHash(versionedBlobHash[:]).Hex()
	err = b.s3Uploader.UploadData(b.ctx, blob[:], key)
	if err != nil {
		log.Error("failed to upload blob data to AWS S3", "batch index", dbBatch.Index, "versioned blob hash", key, "err", err)
		b.metrics.rollupBlobUploaderUploadToS3FailedTotal.Inc()
		// update status to failed
		if updateErr := b.blobUploadOrm.InsertOrUpdateBlobUpload(b.ctx, dbBatch.Index, dbBatch.Hash, types.BlobStoragePlatformS3, types.BlobUploadStatusFailed); updateErr != nil {
			log.Error("failed to update blob upload status to failed", "batch index", dbBatch.Index, "err", updateErr)
		}
		return
	}

	// update status to uploaded
	if err = b.blobUploadOrm.InsertOrUpdateBlobUpload(b.ctx, dbBatch.Index, dbBatch.Hash, types.BlobStoragePlatformS3, types.BlobUploadStatusUploaded); err != nil {
		log.Error("failed to update blob upload status to uploaded", "batch index", dbBatch.Index, "err", err)
		b.metrics.rollupBlobUploaderUploadToS3FailedTotal.Inc()
		return
	}

	b.metrics.rollupBlobUploaderUploadToS3SuccessTotal.Inc()
	log.Info("Successfully uploaded blob to S3", "batch index", dbBatch.Index, "versioned blob hash", key)
}

func (b *BlobUploader) constructBlobCodec(dbBatch *orm.Batch) (*kzg4844.Blob, error) {
	var dbChunks []*orm.Chunk

	dbChunks, err := b.chunkOrm.GetChunksInRange(b.ctx, dbBatch.StartChunkIndex, dbBatch.EndChunkIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks in range: %v", err)
	}

	// check codec version
	for _, dbChunk := range dbChunks {
		if dbBatch.CodecVersion != dbChunk.CodecVersion {
			return nil, fmt.Errorf("batch codec version is different from chunk codec version, batch index: %d, chunk index: %d, batch codec version: %d, chunk codec version: %d", dbBatch.Index, dbChunk.Index, dbBatch.CodecVersion, dbChunk.CodecVersion)
		}
	}

	chunks := make([]*encoding.Chunk, len(dbChunks))
	var allBlocks []*encoding.Block // collect blocks for CodecV7
	for i, c := range dbChunks {
		blocks, getErr := b.l2BlockOrm.GetL2BlocksInRange(b.ctx, c.StartBlockNumber, c.EndBlockNumber)
		if getErr != nil {
			return nil, fmt.Errorf("failed to get blocks in range for batch %d: %w", dbBatch.Index, getErr)
		}
		chunks[i] = &encoding.Chunk{Blocks: blocks}
		allBlocks = append(allBlocks, blocks...)
	}

	var encodingBatch *encoding.Batch
	codecVersion := encoding.CodecVersion(dbBatch.CodecVersion)
	switch codecVersion {
	case encoding.CodecV0:
		return nil, fmt.Errorf("codec version 0 doesn't support blob, batch index: %d", dbBatch.Index)
	case encoding.CodecV1, encoding.CodecV2, encoding.CodecV3, encoding.CodecV4, encoding.CodecV5, encoding.CodecV6:
		encodingBatch = &encoding.Batch{
			Index:                      dbBatch.Index,
			TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
			ParentBatchHash:            common.HexToHash(dbBatch.ParentBatchHash),
			Chunks:                     chunks,
		}

	case encoding.CodecV7:
		encodingBatch = &encoding.Batch{
			Index:                  dbBatch.Index,
			ParentBatchHash:        common.HexToHash(dbBatch.ParentBatchHash),
			Chunks:                 chunks,
			PrevL1MessageQueueHash: common.HexToHash(dbBatch.PrevL1MessageQueueHash),
			PostL1MessageQueueHash: common.HexToHash(dbBatch.PostL1MessageQueueHash),
			Blocks:                 allBlocks,
		}
	default:
		return nil, fmt.Errorf("unsupported codec version, batch index: %d, batch codec version: %d", dbBatch.Index, codecVersion)
	}

	codec, err := encoding.CodecFromVersion(codecVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get codec from version %d, err: %w", dbBatch.CodecVersion, err)
	}

	daBatch, err := codec.NewDABatch(encodingBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to create DA batch: %w", err)
	}

	if daBatch.Blob() == nil {
		return nil, fmt.Errorf("codec version doesn't support blob, batch index: %d, batch codec version: %d", dbBatch.Index, dbBatch.CodecVersion)
	}

	return daBatch.Blob(), nil
}

// GetFirstUnuploadedBatchByPlatform retrieves the first batch that either hasn't been uploaded to corresponding blob storage service
// The batch must have a commit_tx_hash (committed).
func (b *BlobUploader) GetFirstUnuploadedBatchByPlatform(ctx context.Context, startBatch uint64, platform types.BlobStoragePlatform) (*orm.Batch, error) {
	batchIndex, err := b.blobUploadOrm.GetNextBatchIndexToUploadByPlatform(ctx, startBatch, platform)
	if err != nil {
		return nil, err
	}

	var batch *orm.Batch
	for {
		var err error
		batch, err = b.batchOrm.GetBatchByIndex(ctx, batchIndex)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Debug("got batch not proposed for blob uploading", "batch_index", batchIndex, "platform", platform.String())
				return nil, nil
			}
			return nil, err
		}

		// to check if the parent batch uploaded
		// if no, there is a batch revert happened, we need to fallback to upload previous batch
		// skip the check if the parent batch is genesis batch
		if batchIndex <= 1 || batchIndex == startBatch {
			break
		}
		fields := map[string]interface{}{
			"batch_index = ?": batchIndex - 1,
			"batch_hash = ?":  batch.ParentBatchHash,
			"platform = ?":    platform,
			"status = ?":      types.BlobUploadStatusUploaded,
		}
		blobUpload, err := b.blobUploadOrm.GetBlobUploads(ctx, fields, nil, 1)
		if err != nil {
			return nil, err
		}

		if len(blobUpload) == 0 {
			batchIndex--
			continue
		}

		break
	}

	if len(batch.CommitTxHash) == 0 {
		log.Debug("got batch not committed for blob uploading", "batch_index", batchIndex, "platform", platform.String())
		return nil, nil
	}

	return batch, nil
}
