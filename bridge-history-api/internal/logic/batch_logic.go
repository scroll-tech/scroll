package logic

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"

	"bridge-history-api/orm"
)

// BatchLogic example service.
type BatchLogic struct {
	rollupOrm *orm.RollupBatch
}

// NewBatchLogic returns services backed with a "db"
func NewBatchLogic(db *gorm.DB) *BatchLogic {
	logic := &BatchLogic{rollupOrm: orm.NewRollupBatch(db)}
	return logic
}

// GetWithdrawRootByBatchIndex get withdraw root by batch index from db
func (b *BatchLogic) GetWithdrawRootByBatchIndex(ctx context.Context, batchIndex uint64) (string, error) {
	batch, err := b.rollupOrm.GetRollupBatchByIndex(ctx, batchIndex)
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
