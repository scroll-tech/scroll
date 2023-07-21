package orm

import (
	"context"
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
	result := &RollupBatch{}
	err := r.db.WithContext(ctx).Unscoped().Select("commit_height").Order("id desc").Limit(1).Find(&result).Error
	return result.CommitHeight, err
}

// GetLatestRollupBatch return the latest rollup batch in db
func (r *RollupBatch) GetLatestRollupBatch(ctx context.Context) (*RollupBatch, error) {
	result := &RollupBatch{}
	err := r.db.WithContext(ctx).Model(&RollupBatch{}).Where("batch_hash is not NULL").Order("batch_index desc").First(result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

// GetRollupBatchByIndex return the rollup batch by index
func (r *RollupBatch) GetRollupBatchByIndex(ctx context.Context, index uint64) (*RollupBatch, error) {
	result := &RollupBatch{}
	err := r.db.WithContext(ctx).Model(&RollupBatch{}).Where("batch_index = ?", index).First(result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

// BatchInsertRollupBatch batch insert rollup batch into db and return the transaction
func (r *RollupBatch) BatchInsertRollupBatch(ctx context.Context, batches []*RollupBatch, dbTx ...*gorm.DB) error {
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
		log.Error("failed to insert rollup batch", "batchIndexes", batchIndexes, "heights", heights, "err", err)
	}
	return err
}
