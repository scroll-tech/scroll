package orm

import (
	"time"

	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// RollupBatch is the struct for rollup_batch table
type RollupBatch struct {
	db *gorm.DB `gorm:"column:-"`

	ID               uint64     `json:"id" gorm:"column:id"`
	BatchIndex       uint64     `json:"batch_index" gorm:"column:batch_index"`
	BatchHash        string     `json:"batch_hash" gorm:"column:batch_hash"`
	CommitHeight     uint64     `json:"commit_height" gorm:"column:commit_height"`
	StartBlockNumber uint64     `json:"start_block_number" gorm:"column:start_block_number"`
	EndBlockNumber   uint64     `json:"end_block_number" gorm:"column:end_block_number"`
	CreatedAt        *time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        *time.Time `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewRollupBatch create an RollupBatch instance
func NewRollupBatch(db *gorm.DB) *RollupBatch {
	return &RollupBatch{db: db}
}

// BatchInsertRollupBatchDBTx batch insert rollup batch into db and return the transaction
func (r *RollupBatch) BatchInsertRollupBatchDBTx(dbTx *gorm.DB, batches []*RollupBatch) (*gorm.DB, error) {
	if len(batches) == 0 {
		return dbTx, nil
	}
	err := dbTx.Table("rollup_batch").Create(&batches).Error

	if err != nil {
		batchIndexes := make([]uint64, 0, len(batches))
		heights := make([]uint64, 0, len(batches))
		for _, batch := range batches {
			batchIndexes = append(batchIndexes, batch.BatchIndex)
			heights = append(heights, batch.CommitHeight)
		}
		log.Error("failed to insert rollup batch", "batchIndexes", batchIndexes, "heights", heights, "err", err)
	}
	return dbTx, err
}

// GetLatestRollupBatch return the latest rollup batch in db
func (r *RollupBatch) GetLatestRollupBatch() (*RollupBatch, error) {
	result := &RollupBatch{}
	err := r.db.Table("rollup_batch").Where("batch_hash is not NULL").Order("batch_index desc").First(result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

// GetRollupBatchByIndex return the rollup batch by index
func (r *RollupBatch) GetRollupBatchByIndex(index uint64) (*RollupBatch, error) {
	result := &RollupBatch{}
	err := r.db.Table("rollup_batch").Where("batch_index = ?", index).First(result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}
