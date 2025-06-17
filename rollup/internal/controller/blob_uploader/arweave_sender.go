package blob_uploader

import (
	"context"
	"fmt"
	"scroll-tech/common/types"
	"scroll-tech/common/utils"
	"scroll-tech/rollup/internal/config"
	"scroll-tech/rollup/internal/orm"
	"time"

	"github.com/permadao/goar"
	"github.com/permadao/goar/schema"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
)

// Arweave Uploader is responsible for uploading data to AWS S3.
type ArweaveUploader struct {
	ctx           context.Context
	wallet        *goar.Wallet
	client        *goar.Client
	txTag         string
	confirmations int
	blobUploadOrm *orm.BlobUpload
}

func NewArweaveUploader(ctx context.Context, cfg *config.ArweaveConfig, db *gorm.DB) (*ArweaveUploader, error) {
	wallet, err := goar.NewWallet([]byte(cfg.PrivateKey), cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to new arweave uploader: %w", err)
	}

	client := goar.NewClient(cfg.Endpoint)

	arweaveUploader := ArweaveUploader{
		ctx:           ctx,
		wallet:        wallet,
		client:        client,
		txTag:         cfg.TxTag,
		confirmations: int(cfg.Confirmations),
		blobUploadOrm: orm.NewBlobUpload(db),
	}

	go arweaveUploader.loop(ctx)

	return &arweaveUploader, nil
}

// UploadData uploads data to arweave
func (u *ArweaveUploader) UploadData(ctx context.Context, data []byte, objectKey string) (*schema.Transaction, error) {
	tx, err := u.wallet.SendData(
		data,
		[]schema.Tag{
			schema.Tag{
				Name:  u.txTag,
				Value: objectKey,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

// UploadDataSpeedUp uploads data to arweave with higher gas price
func (u *ArweaveUploader) UploadDataSpeedUp(ctx context.Context, data []byte, objectKey string, speedFactor int64) (*schema.Transaction, error) {
	tx, err := u.wallet.SendDataSpeedUp(
		data,
		[]schema.Tag{
			schema.Tag{
				Name:  u.txTag,
				Value: objectKey,
			},
		},
		speedFactor,
	)
	if err != nil {
		return nil, err
	}

	return &tx, nil
}

// checkPendingBlobUploads checks the confirmation status of pending blob uploads transactions against the latest confirmed block number.
// If a transaction hasn't been confirmed after a certain number of blocks, it will be resubmitted with an increased gas price.
func (u *ArweaveUploader) checkPendingBlobUploads(ctx context.Context) {
	pendingBlobUploads, err := u.blobUploadOrm.GetPendingBlobUploads(ctx, 100)
	if err != nil {
		log.Error("failed to load pending blob uploads", "err", err)
		return
	}

	for _, blobUpload := range pendingBlobUploads {
		status, err := u.client.GetTransactionStatus(blobUpload.TxHash)
		if err != nil {
			if err == schema.ErrPendingTx {
				if time.Since(blobUpload.UpdatedAt) > 10*time.Minute {
					// transaction pending too long, we need to bump the gas price
					(blobUpload.BatchHash, blobUpload.BatchIndex)

					
				}
				u.client.GetTransactionByID(blobUpload.TxHash)
				log.Debug("got pending arweave transaction, waitting for confirmation")
			}
			if err == schema.ErrNotFound || err == schema.ErrInvalidId {

			}
			log.Error("failed to get arweave transaction status", "err", err)
			return
		}

		if status.NumberOfConfirmations >= int(u.confirmations) {
			if err := u.blobUploadOrm.UpdateUploadStatus(u.ctx, blobUpload.TxHash, types.BlobUploadStatusUploaded); err != nil {
				log.Warn("UpdateUploadStatus failed", "transaction hash", blobUpload.TxHash, "upload status", types.BlobUploadStatusUploaded, "err", err)
				return
			}

		}

	}
}

// Loop is the main event loop
func (u *ArweaveUploader) loop(ctx context.Context) {
	checkTick := time.NewTicker(time.Duration(10) * time.Second)
	defer checkTick.Stop()

	for {
		select {
		case <-checkTick.C:
			u.checkPendingBlobUploads(ctx)
		case <-ctx.Done():
			return
		}
	}
}
