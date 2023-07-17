package orm

import (
	"github.com/ethereum/go-ethereum/log"
	"gorm.io/gorm"
)

// RollupBatch is the struct for rollup_batch table
type RollupBatch struct {
	db *gorm.DB `gorm:"column:-"`

	ID               uint64 `json:"id" gorm:"column:id"`
	BatchIndex       uint64 `json:"batch_index" gorm:"column:batch_index"`
	BatchHash        string `json:"batch_hash" gorm:"column:batch_hash"`
	CommitHeight     uint64 `json:"commit_height" gorm:"column:commit_height"`
	StartBlockNumber uint64 `json:"start_block_number" gorm:"column:start_block_number"`
	EndBlockNumber   uint64 `json:"end_block_number" gorm:"column:end_block_number"`
}

// NewRollupBatch create an RollupBatch instance
func NewRollupBatch(db *gorm.DB) *RollupBatch {
	return &RollupBatch{db: db}
}

func (r *RollupBatch) BatchInsertRollupBatchDBTx(dbTx *gorm.DB, batches []*RollupBatch) error {
	if len(batches) == 0 {
		return nil
	}
	err := dbTx.Create(&batches).Error

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

func (r *RollupBatch) GetLatestRollupBatch() (*RollupBatch, error) {
	result := &RollupBatch{}
	err := r.db.Model(result).Select("id, batch_index, commit_height, batch_hash, start_block_number, end_block_number").Order("batch_index desc").Limit(1).Find(result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (r *RollupBatch) GetRollupBatchByIndex(index uint64) (*RollupBatch, error) {
	result := &RollupBatch{}
	err := r.db.Select("id, batch_index, commit_height, batch_hash, start_block_number, end_block_number").Where("batch_index = ?", index).Find(result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}
