package orm

import (
	"encoding/json"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge/internal/types"
)

// L2Block is structure of stored l2 block message
// L2Block represents a row in the "l2_block" table.
type L2Block struct {
	db *gorm.DB `gorm:"column:-"`

	Number           uint64 `json:"number" gorm:"number"`
	Hash             string `json:"hash" gorm:"hash"`
	ParentHash       string `json:"parent_hash" gorm:"parent_hash"`
	Header           string `json:"header" gorm:"header"`
	Transactions     string `json:"transactions" gorm:"transactions"`
	WithdrawTrieRoot string `json:"withdraw_trie_root" gorm:"withdraw_trie_root"`
	TxNum            uint64 `json:"tx_num" gorm:"tx_num"`
	GasUsed          uint64 `json:"gas_used" gorm:"gas_used"`
	BlockTimestamp   uint64 `json:"block_timestamp" gorm:"block_timestamp"`
	ChunkHash        string `json:"chunk_hash" gorm:"chunk_hash"`
}

// NewL2Block creates a new L2Block instance
func NewL2Block(db *gorm.DB) *L2Block {
	return &L2Block{db: db}
}

// TableName returns the name of the "l2_block" table.
func (*L2Block) TableName() string {
	return "l2_block"
}

// GetL2BlocksLatestHeight get the l2 blocks latest height
func (o *L2Block) GetL2BlocksLatestHeight() (int64, error) {
	result := o.db.Model(&L2Block{}).Select("COALESCE(MAX(number), -1)").Row()
	if result.Err() != nil {
		return -1, result.Err()
	}
	var maxNumber int64
	if err := result.Scan(&maxNumber); err != nil {
		return -1, err
	}
	return maxNumber, nil
}

// GetUnchunkedBlocks get the l2 blocks that have not been put into a chunk
func (o *L2Block) GetUnchunkedBlocks() ([]*types.WrappedBlock, error) {
	var l2Blocks []L2Block
	db := o.db.Select("header, transactions, withdraw_trie_root")
	db = db.Where("chunk_hash IS NULL")

	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, err
	}

	var wrappedBlocks []*types.WrappedBlock
	for _, v := range l2Blocks {
		var wrappedBlock types.WrappedBlock

		if err := json.Unmarshal([]byte(v.Transactions), &wrappedBlock.Transactions); err != nil {
			return nil, err
		}

		wrappedBlock.Header = &gethTypes.Header{}
		if err := json.Unmarshal([]byte(v.Header), wrappedBlock.Header); err != nil {
			return nil, err
		}

		wrappedBlock.WithdrawTrieRoot = common.HexToHash(v.WithdrawTrieRoot)
		wrappedBlocks = append(wrappedBlocks, &wrappedBlock)
	}

	return wrappedBlocks, nil
}

// GetL2WrappedBlocks get the l2 wrapped blocks
func (o *L2Block) GetL2WrappedBlocks(fields map[string]interface{}, orderByList []string, limit int) ([]*types.WrappedBlock, error) {
	var l2Blocks []L2Block
	db := o.db.Select("header, transactions, withdraw_trie_root")

	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit != 0 {
		db = db.Limit(limit)
	}

	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, err
	}

	var wrappedBlocks []*types.WrappedBlock
	for _, v := range l2Blocks {
		var wrappedBlock types.WrappedBlock

		if err := json.Unmarshal([]byte(v.Transactions), &wrappedBlock.Transactions); err != nil {
			return nil, err
		}

		wrappedBlock.Header = &gethTypes.Header{}
		if err := json.Unmarshal([]byte(v.Header), wrappedBlock.Header); err != nil {
			return nil, err
		}

		wrappedBlock.WithdrawTrieRoot = common.HexToHash(v.WithdrawTrieRoot)
		wrappedBlocks = append(wrappedBlocks, &wrappedBlock)
	}

	return wrappedBlocks, nil
}

// GetL2Blocks get l2 blocks
func (o *L2Block) GetL2Blocks(fields map[string]interface{}, orderByList []string, limit int) ([]L2Block, error) {
	var l2Blocks []L2Block
	db := o.db.Select("number, hash, parent_hash, chunk_hash, tx_num, gas_used, block_timestamp")
	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit != 0 {
		db = db.Limit(limit)
	}

	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, err
	}
	return l2Blocks, nil
}

// InsertL2Blocks inserts l2 blocks into the "l2_block" table.
func (o *L2Block) InsertL2Blocks(blocks []*types.WrappedBlock) error {
	var l2Blocks []L2Block
	for _, block := range blocks {
		header, err := json.Marshal(block.Header)
		if err != nil {
			log.Error("failed to marshal block header", "hash", block.Header.Hash().String(), "err", err)
			return err
		}

		txs, err := json.Marshal(block.Transactions)
		if err != nil {
			log.Error("failed to marshal transactions", "hash", block.Header.Hash().String(), "err", err)
			return err
		}

		l2Block := L2Block{
			Number:           block.Header.Number.Uint64(),
			Hash:             block.Header.Hash().String(),
			ParentHash:       block.Header.ParentHash.String(),
			Transactions:     string(txs),
			WithdrawTrieRoot: block.WithdrawTrieRoot.Hex(),
			TxNum:            uint64(len(block.Transactions)),
			GasUsed:          block.Header.GasUsed,
			BlockTimestamp:   block.Header.Time,
			Header:           string(header),
		}
		l2Blocks = append(l2Blocks, l2Block)
	}

	if err := o.db.Create(&l2Blocks).Error; err != nil {
		log.Error("failed to insert l2Blocks", "err", err)
		return err
	}
	return nil
}

// UpdateBatchHashForL2Blocks update the batch_hash of block tx
func (o *L2Block) UpdateBatchHashForL2Blocks(blockNumbers []uint64, batchHash string, tx ...*gorm.DB) error {
	db := o.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}

	err := db.Model(&L2Block{}).Where("number IN (?)", blockNumbers).Update("batch_hash", batchHash).Error
	return err
}
