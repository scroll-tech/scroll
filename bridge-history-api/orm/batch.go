package orm

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// RollupBatch is the struct for rollup_batch table
type RollupBatch struct {
	db *gorm.DB `gorm:"column:-"`

	ID               uint64         `json:"id" gorm:"column:id"`
	BatchIndex       uint64         `json:"batch_index" gorm:"column:batch_index"`
	BatchHash        string         `json:"batch_hash" gorm:"column:batch_hash"`
	CommitHeight     uint64         `json:"commit_height" gorm:"column:commit_height"`
	StartBlockNumber uint64         `json:"start_block_number" gorm:"column:start_block_number"`
	EndBlockNumber   uint64         `json:"end_block_number" gorm:"column:end_block_number"`
	WithdrawRoot     string         `json:"withdraw_root" gorm:"column:withdraw_root;default:NULL"`
	CreatedAt        *time.Time     `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        *time.Time     `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewRollupBatch create an RollupBatch instance
func NewRollupBatch(db *gorm.DB) *RollupBatch {
	return &RollupBatch{db: db}
}

// TableName returns the table name for the Batch model.
func (*RollupBatch) TableName() string {
	return "rollup_batch"
}

// GetLatestRollupBatchProcessedHeight return latest processed height from rollup_batch table
func (r *RollupBatch) GetLatestRollupBatchProcessedHeight(ctx context.Context) (uint64, error) {
	var result RollupBatch
	err := r.db.WithContext(ctx).Unscoped().Select("commit_height").Order("id desc").First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("RollupBatch.GetLatestRollupBatchProcessedHeight error: %w", err)
	}
	return result.CommitHeight, nil
}

// GetLatestRollupBatch return the latest rollup batch in db
func (r *RollupBatch) GetLatestRollupBatch(ctx context.Context) (*RollupBatch, error) {
	var result RollupBatch
	err := r.db.WithContext(ctx).Model(&RollupBatch{}).Where("batch_hash is not NULL").Order("batch_index desc").First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("RollupBatch.GetLatestRollupBatch error: %w", err)
	}
	return &result, nil
}

// GetRollupBatchByIndex return the rollup batch by index
func (r *RollupBatch) GetRollupBatchByIndex(ctx context.Context, index uint64) (*RollupBatch, error) {
	var result RollupBatch
	err := r.db.WithContext(ctx).Model(&RollupBatch{}).Where("batch_index = ?", index).First(&result).Error
	if err != nil {
		return nil, fmt.Errorf("RollupBatch.GetRollupBatchByIndex error: %w", err)
	}
	return &result, nil
}

// GetRollupBatchesByIndexes return the rollup batches by indexes
func (r *RollupBatch) GetRollupBatchesByIndexes(ctx context.Context, indexes []uint64) ([]*RollupBatch, error) {
	var results []*RollupBatch
	err := r.db.WithContext(ctx).Model(&RollupBatch{}).Where("batch_index IN ?", indexes).Find(&results).Error
	if err != nil {
		return nil, fmt.Errorf("RollupBatch.GetRollupBatchesByIndexes error: %w", err)
	}
	return results, nil
}

// InsertRollupBatch batch insert rollup batch into db and return the transaction
func (r *RollupBatch) InsertRollupBatch(ctx context.Context, batches []*RollupBatch, dbTx ...*gorm.DB) error {
	if len(batches) == 0 {
		return nil
	}
	db := r.db
	if len(dbTx) > 0 && dbTx[0] != nil {
		db = dbTx[0]
	}
	err := db.WithContext(ctx).Model(&RollupBatch{}).Create(&batches).Error
	if err != nil {
		batchIndexes := make([]uint64, 0, len(batches))
		heights := make([]uint64, 0, len(batches))
		for _, batch := range batches {
			batchIndexes = append(batchIndexes, batch.BatchIndex)
			heights = append(heights, batch.CommitHeight)
		}
		log.Error("failed to insert rollup batch", "batchIndexes", batchIndexes, "heights", heights)
		return fmt.Errorf("RollupBatch.InsertRollupBatch error: %w", err)
	}
	return nil
}

// UpdateRollupBatchWithdrawRoot updates the withdraw_root column in rollup_batch table
func (r *RollupBatch) UpdateRollupBatchWithdrawRoot(ctx context.Context, batchIndex uint64, withdrawRoot string) error {
	err := r.db.WithContext(ctx).Model(&RollupBatch{}).Where("batch_index = ?", batchIndex).Update("withdraw_root", withdrawRoot).Error
	if err != nil {
		return fmt.Errorf("RollupBatch.UpdateRuollupBatch error: %w", err)
	}
	return nil
}
