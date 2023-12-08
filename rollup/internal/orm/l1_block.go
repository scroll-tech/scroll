package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// L1Block is structure of stored l1 block message
type L1Block struct {
	db *gorm.DB `gorm:"column:-"`

	// block
	Number  uint64 `json:"number" gorm:"column:number"`
	Hash    string `json:"hash" gorm:"column:hash"`
	BaseFee uint64 `json:"base_fee" gorm:"column:base_fee"`

	// oracle
	GasOracleStatus int16  `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash    string `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewL1Block create a l1Block instance
func NewL1Block(db *gorm.DB) *L1Block {
	return &L1Block{db: db}
}

// TableName define the L1Block table name
func (*L1Block) TableName() string {
	return "l1_block"
}

// GetLatestL1BlockHeight get the latest l1 block height
func (o *L1Block) GetLatestL1BlockHeight(ctx context.Context) (uint64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L1Block{})
	db = db.Select("COALESCE(MAX(number), 0)")

	var maxNumber uint64
	if err := db.Row().Scan(&maxNumber); err != nil {
		return 0, fmt.Errorf("L1Block.GetLatestL1BlockHeight error: %w", err)
	}
	return maxNumber, nil
}

// GetL1Blocks get the l1 blocks
func (o *L1Block) GetL1Blocks(ctx context.Context, fields map[string]interface{}) ([]L1Block, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L1Block{})

	for key, value := range fields {
		db = db.Where(key, value)
	}

	db = db.Order("number ASC")

	var l1Blocks []L1Block
	if err := db.Find(&l1Blocks).Error; err != nil {
		return nil, fmt.Errorf("L1Block.GetL1Blocks error: %w, fields: %v", err, fields)
	}
	return l1Blocks, nil
}

// InsertL1Blocks batch inserts l1 blocks.
// If there's a block number conflict (e.g., due to reorg), soft deletes the existing block and inserts the new one.
func (o *L1Block) InsertL1Blocks(ctx context.Context, blocks []L1Block) error {
	if len(blocks) == 0 {
		return nil
	}

	return o.db.Transaction(func(tx *gorm.DB) error {
		minBlockNumber := blocks[0].Number
		for _, block := range blocks[1:] {
			if block.Number < minBlockNumber {
				minBlockNumber = block.Number
			}
		}

		db := tx.WithContext(ctx)
		db = db.Model(&L1Block{})
		db = db.Where("number >= ?", minBlockNumber)
		result := db.Delete(&L1Block{})

		if result.Error != nil {
			return fmt.Errorf("L1Block.InsertL1Blocks error: soft deleting blocks failed, block numbers starting from: %v, error: %w", minBlockNumber, result.Error)
		}

		// If the number of deleted blocks exceeds the limit (input length + 64), treat it as an anomaly.
		// Because reorg with >= 64 blocks is very unlikely to happen.
		if result.RowsAffected >= int64(len(blocks)+64) {
			return fmt.Errorf("L1Block.InsertL1Blocks error: too many blocks were deleted, count: %d", result.RowsAffected)
		}

		if err := db.Create(&blocks).Error; err != nil {
			return fmt.Errorf("L1Block.InsertL1Blocks error: %w", err)
		}
		return nil
	})
}

// UpdateL1GasOracleStatusAndOracleTxHash update l1 gas oracle status and oracle tx hash
func (o *L1Block) UpdateL1GasOracleStatusAndOracleTxHash(ctx context.Context, blockHash string, status types.GasOracleStatus, txHash string) error {
	updateFields := map[string]interface{}{
		"oracle_status":  int(status),
		"oracle_tx_hash": txHash,
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&L1Block{})
	db = db.Where("hash", blockHash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("L1Block.UpdateL1GasOracleStatusAndOracleTxHash error: %w, block hash: %v, status: %v, tx hash: %v", err, blockHash, status.String(), txHash)
	}
	return nil
}
