package orm

import (
	"context"
	"database/sql"

	"github.com/scroll-tech/go-ethereum/log"
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
	ImportTxHash    string `json:"import_tx_hash" gorm:"column:import_tx_hash"`
	GasOracleStatus int    `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash    string `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash"`
}

// NewL1Block create an l1Block instance
func NewL1Block(db *gorm.DB) *L1Block {
	return &L1Block{db: db}
}

// TableName define the L1Message table name
func (*L1Block) TableName() string {
	return "l1_block"
}

// GetLatestL1BlockHeight get the latest l1 block height
func (l *L1Block) GetLatestL1BlockHeight() (uint64, error) {
	var maxNumber sql.NullInt64
	result := l.db.Model(&L1Block{}).Select("COALESCE(MAX(number), 0)").Scan(&maxNumber)
	if result.Error != nil {
		return -1, result.Error
	}
	if maxNumber.Valid {
		return uint64(maxNumber.Int64), nil
	}
	return -1, nil
}

// GetL1BlockInfos get the l1 block infos
func (l *L1Block) GetL1BlockInfos(fields map[string]interface{}) ([]L1Block, error) {
	var l1Blocks []L1Block
	selectFileds := "number, hash, header_rlp, base_fee, block_status, oracle_status, import_tx_hash, oracle_tx_hash"
	db := l.db.Select(selectFileds)
	for key, value := range fields {
		db = db.Where(key, value)
	}
	db = db.Order("number ASC")
	if err := db.Find(&l1Blocks).Error; err != nil {
		return nil, err
	}
	return l1Blocks, nil
}

// InsertL1Blocks batch insert l1 blocks
func (l *L1Block) InsertL1Blocks(ctx context.Context, blocks []L1Block) error {
	if len(blocks) == 0 {
		return nil
	}

	err := l.db.WithContext(ctx).Create(&blocks).Error
	if err != nil {
		log.Error("failed to insert L1 Blocks", "err", err)
	}
	return err
}

// UpdateL1GasOracleStatusAndOracleTxHash update l1 gas oracle status and oracle tx hash
func (l *L1Block) UpdateL1GasOracleStatusAndOracleTxHash(ctx context.Context, blockHash string, status types.GasOracleStatus, txHash string) error {
	updateFields := map[string]interface{}{
		"oracle_status":  int(status),
		"oracle_tx_hash": txHash,
	}
	if err := l.db.WithContext(ctx).Model(&L1Block{}).Where("hash", blockHash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}
