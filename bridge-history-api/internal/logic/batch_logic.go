package logic

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"

	"bridge-history-api/orm"
)

// BatchLogic example service.
type BatchLogic struct {
	db *gorm.DB
}

// NewBatchLogic returns services backed with a "db"
func NewBatchLogic(db *gorm.DB) *BatchLogic {
	logic := &BatchLogic{db: db}
	return logic
}

// GetWithdrawRootByBatchIndex get withdraw root by batch index from db
func (b *BatchLogic) GetWithdrawRootByBatchIndex(ctx context.Context, batchIndex uint64) (string, error) {
	batchOrm := orm.NewRollupBatch(b.db)
	batch, err := batchOrm.GetRollupBatchByIndex(ctx, batchIndex)
	if err != nil {
		log.Debug("getWithdrawRootByBatchIndex failed", "error", err)
		return "", err
	}
	if batch == nil {
		log.Debug("getWithdrawRootByBatchIndex failed", "error", "batch not found")
		return "", nil
	}
	return batch.WithdrawRoot, nil
}
