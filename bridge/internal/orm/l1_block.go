package orm

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// L1Block is structure of stored l1 block message
type L1Block struct {
	db *gorm.DB `gorm:"column:-"`

	Number          uint64 `json:"number" gorm:"column:number"`
	Hash            string `json:"hash" gorm:"column:hash"`
	HeaderRLP       string `json:"header_rlp" gorm:"column:header_rlp"`
	BaseFee         uint64 `json:"base_fee" gorm:"column:base_fee"`
	BlockStatus     int    `json:"block_status" gorm:"column:block_status;default:1"`
	ImportTxHash    string `json:"import_tx_hash" gorm:"column:import_tx_hash;default:NULL"`
	GasOracleStatus int    `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash    string `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash;default:NULL"`
}

// NewL1Block create an l1Block instance
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

	result := db.Row()
	if result.Err() != nil {
		return 0, fmt.Errorf("L1Block.GetLatestL1BlockHeight error: %w", result.Err())
	}

	var maxNumber uint64
	if err := result.Scan(&maxNumber); err != nil {
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

// InsertL1Blocks batch insert l1 blocks
func (o *L1Block) InsertL1Blocks(ctx context.Context, blocks []L1Block) error {
	if len(blocks) == 0 {
		return nil
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&L1Block{})

	if err := db.Create(&blocks).Error; err != nil {
		return fmt.Errorf("L1Block.InsertL1Blocks error: %w", err)
	}
	return nil
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
