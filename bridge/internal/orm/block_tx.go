package orm

import (
	"encoding/json"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge/internal/types"
)

// BlockTx is structure of stored block tx message
type BlockTx struct {
	db *gorm.DB `gorm:"column:-"`

	Number         uint64 `json:"number" gorm:"number"`
	Hash           string `json:"hash" gorm:"hash"`
	ParentHash     string `json:"parent_hash" gorm:"parent_hash"`
	Tx             string `json:"tx" gorm:"column:tx"`
	BatchHash      string `json:"batch_hash" gorm:"batch_hash;default:NULL"`
	TxNum          uint64 `json:"tx_num" gorm:"tx_num"`
	GasUsed        uint64 `json:"gas_used" gorm:"gas_used"`
	BlockTimestamp uint64 `json:"block_timestamp" gorm:"block_timestamp"`
}

// NewBlockTx create an blockTxOrm instance
func NewBlockTx(db *gorm.DB) *BlockTx {
	return &BlockTx{db: db}
}

// TableName define the BlockTx table name
func (*BlockTx) TableName() string {
	return "block_tx"
}

// GetL2BlocksLatestHeight get the l2 blocks latest height
func (o *BlockTx) GetL2BlocksLatestHeight() (int64, error) {
	result := o.db.Model(&BlockTx{}).Select("COALESCE(MAX(number), -1)").Row()
	if result.Err() != nil {
		return -1, result.Err()
	}
	var maxNumber int64
	if err := result.Scan(&maxNumber); err != nil {
		return -1, err
	}
	return maxNumber, nil
}

// GetL2WrappedBlocks get the l2 wrapped blocks
func (o *BlockTx) GetL2WrappedBlocks(fields map[string]interface{}) ([]*types.WrappedBlock, error) {
	var blockTxs []BlockTx
	db := o.db.Select("tx")
	for key, value := range fields {
		db = db.Where(key, value)
	}
	if err := db.Find(&blockTxs).Error; err != nil {
		return nil, err
	}

	var wrappedBlocks []*types.WrappedBlock
	for _, v := range blockTxs {
		var wrappedBlock types.WrappedBlock
		if err := json.Unmarshal([]byte(v.Tx), &wrappedBlock); err != nil {
			break
		}
		wrappedBlocks = append(wrappedBlocks, &wrappedBlock)
	}
	return wrappedBlocks, nil
}

// GetL2BlockTxs get l2 block txs
func (o *BlockTx) GetL2BlockTxs(fields map[string]interface{}, orderByList []string, limit int) ([]BlockTx, error) {
	var blockTxs []BlockTx
	db := o.db.Select("number, hash, parent_hash, batch_hash, tx_num, gas_used, block_timestamp")
	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit != 0 {
		db = db.Limit(limit)
	}

	if err := db.Find(&blockTxs).Error; err != nil {
		return nil, err
	}
	return blockTxs, nil
}

// GetUnbatchedL2Blocks get unbatched l2 blocks
func (o *BlockTx) GetUnbatchedL2Blocks(fields map[string]interface{}, orderByList []string, limit int) ([]BlockTx, error) {
	var unbatchedBlockTxs []BlockTx
	db := o.db.Select("number, hash, parent_hash, batch_hash, tx_num, gas_used, block_timestamp").Where("batch_hash is NULL")
	for key, value := range fields {
		db = db.Where(key, value)
	}
	if err := db.Find(&unbatchedBlockTxs).Error; err != nil {
		return nil, err
	}
	return unbatchedBlockTxs, nil
}

// InsertWrappedBlocks insert block to block tx
func (o *BlockTx) InsertWrappedBlocks(blocks []*types.WrappedBlock) error {
	var blockTxs []BlockTx
	for _, block := range blocks {
		number := block.Header.Number.Uint64()
		hash := block.Header.Hash().String()
		txNum := len(block.Transactions)
		mtime := block.Header.Time
		gasCost := block.Header.GasUsed

		data, err := json.Marshal(block)
		if err != nil {
			log.Error("failed to marshal block", "hash", hash, "err", err)
			return err
		}

		tmpBlockTx := BlockTx{
			Number:         number,
			Hash:           hash,
			ParentHash:     block.Header.ParentHash.String(),
			Tx:             string(data),
			TxNum:          uint64(txNum),
			GasUsed:        gasCost,
			BlockTimestamp: mtime,
		}
		blockTxs = append(blockTxs, tmpBlockTx)
	}

	if err := o.db.Create(&blockTxs).Error; err != nil {
		log.Error("failed to insert blockTxs", "err", err)
		return err
	}
	return nil
}

// UpdateBatchHashForL2Blocks update the batch_hash of block tx
func (o *BlockTx) UpdateBatchHashForL2Blocks(blockNumbers []uint64, batchHash string, tx ...*gorm.DB) error {
	db := o.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}

	err := db.Model(&BlockTx{}).Where("number IN (?)", blockNumbers).Update("batch_hash", batchHash).Error
	return err
}
