package orm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// L2Block represents a l2 block in the database.
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
	ChunkHash        string `json:"chunk_hash" gorm:"chunk_hash;default:NULL"`
}

// NewL2Block creates a new L2Block instance
func NewL2Block(db *gorm.DB) *L2Block {
	return &L2Block{db: db}
}

// TableName returns the name of the "l2_block" table.
func (*L2Block) TableName() string {
	return "l2_block"
}

// GetL2BlocksLatestHeight retrieves the height of the latest L2 block.
// If the l2_block table is empty, it returns 0 to represent the genesis block height.
// In case of an error, it returns -1 along with the error.
func (o *L2Block) GetL2BlocksLatestHeight(ctx context.Context) (int64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("COALESCE(MAX(number), 0)")

	var maxNumber int64
	if err := db.Row().Scan(&maxNumber); err != nil {
		return -1, fmt.Errorf("L2Block.GetL2BlocksLatestHeight error: %w", err)
	}
	return maxNumber, nil
}

// GetUnchunkedBlocks get the l2 blocks that have not been put into a chunk.
// The returned blocks are sorted in ascending order by their block number.
func (o *L2Block) GetUnchunkedBlocks(ctx context.Context) ([]*types.WrappedBlock, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("header, transactions, withdraw_trie_root")
	db = db.Where("chunk_hash IS NULL")
	db = db.Order("number ASC")

	var l2Blocks []L2Block
	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetUnchunkedBlocks error: %w", err)
	}

	var wrappedBlocks []*types.WrappedBlock
	for _, v := range l2Blocks {
		var wrappedBlock types.WrappedBlock

		if err := json.Unmarshal([]byte(v.Transactions), &wrappedBlock.Transactions); err != nil {
			return nil, fmt.Errorf("L2Block.GetUnchunkedBlocks error: %w", err)
		}

		wrappedBlock.Header = &gethTypes.Header{}
		if err := json.Unmarshal([]byte(v.Header), wrappedBlock.Header); err != nil {
			return nil, fmt.Errorf("L2Block.GetUnchunkedBlocks error: %w", err)
		}

		wrappedBlock.WithdrawTrieRoot = common.HexToHash(v.WithdrawTrieRoot)
		wrappedBlocks = append(wrappedBlocks, &wrappedBlock)
	}

	return wrappedBlocks, nil
}

// GetL2Blocks retrieves selected L2Blocks from the database.
// The returned L2Blocks are sorted in ascending order by their block number.
func (o *L2Block) GetL2Blocks(ctx context.Context, fields map[string]interface{}, orderByList []string, limit int) ([]*L2Block, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})

	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit > 0 {
		db = db.Limit(limit)
	}

	db = db.Order("number ASC")

	var l2Blocks []*L2Block
	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetL2Blocks error: %w, fields: %v, orderByList: %v", err, fields, orderByList)
	}
	return l2Blocks, nil
}

// GetL2BlocksInRange retrieves the L2 blocks within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end block numbers.
// The returned blocks are sorted in ascending order by their block number.
func (o *L2Block) GetL2BlocksInRange(ctx context.Context, startBlockNumber uint64, endBlockNumber uint64) ([]*types.WrappedBlock, error) {
	if startBlockNumber > endBlockNumber {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange: start block number should be less than or equal to end block number, start block: %v, end block: %v", startBlockNumber, endBlockNumber)
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("header, transactions, withdraw_trie_root")
	db = db.Where("number >= ? AND number <= ?", startBlockNumber, endBlockNumber)
	db = db.Order("number ASC")

	var l2Blocks []L2Block
	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
	}

	// sanity check
	if uint64(len(l2Blocks)) != endBlockNumber-startBlockNumber+1 {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange: number of blocks not expected in the specified range, expected: %v, got: %v", endBlockNumber-startBlockNumber+1, len(l2Blocks))
	}

	var wrappedBlocks []*types.WrappedBlock
	for _, v := range l2Blocks {
		var wrappedBlock types.WrappedBlock

		if err := json.Unmarshal([]byte(v.Transactions), &wrappedBlock.Transactions); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
		}

		wrappedBlock.Header = &gethTypes.Header{}
		if err := json.Unmarshal([]byte(v.Header), wrappedBlock.Header); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
		}

		wrappedBlock.WithdrawTrieRoot = common.HexToHash(v.WithdrawTrieRoot)
		wrappedBlocks = append(wrappedBlocks, &wrappedBlock)
	}

	return wrappedBlocks, nil
}

// InsertL2Blocks inserts l2 blocks into the "l2_block" table.
func (o *L2Block) InsertL2Blocks(ctx context.Context, blocks []*types.WrappedBlock) error {
	var l2Blocks []L2Block
	for _, block := range blocks {
		header, err := json.Marshal(block.Header)
		if err != nil {
			log.Error("failed to marshal block header", "hash", block.Header.Hash().String(), "err", err)
			return fmt.Errorf("L2Block.InsertL2Blocks error: %w", err)
		}

		txs, err := json.Marshal(block.Transactions)
		if err != nil {
			log.Error("failed to marshal transactions", "hash", block.Header.Hash().String(), "err", err)
			return fmt.Errorf("L2Block.InsertL2Blocks error: %w", err)
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

	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})

	if err := db.Create(&l2Blocks).Error; err != nil {
		return fmt.Errorf("L2Block.InsertL2Blocks error: %w", err)
	}
	return nil
}

// UpdateChunkHashInRange updates the chunk_hash of block tx within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end indices.
// This function ensures the number of rows updated must equal to (endIndex - startIndex + 1).
// If the rows affected do not match this expectation, an error is returned.
func (o *L2Block) UpdateChunkHashInRange(ctx context.Context, startIndex uint64, endIndex uint64, chunkHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Where("number >= ? AND number <= ?", startIndex, endIndex)

	tx := db.Update("chunk_hash", chunkHash)
	if tx.Error != nil {
		return fmt.Errorf("L2Block.UpdateChunkHashInRange error: %w, start index: %v, end index: %v, chunk hash: %v", tx.Error, startIndex, endIndex, chunkHash)
	}

	// sanity check
	if uint64(tx.RowsAffected) != endIndex-startIndex+1 {
		return fmt.Errorf("L2Block.UpdateChunkHashInRange: incorrect number of affected rows after update, expected: %v, got: %v", endIndex-startIndex+1, tx.RowsAffected)
	}

	return nil
}
