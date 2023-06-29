package orm

import (
	"context"
	"encoding/json"

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

// GetL2BlocksByChunkHash retrieves the L2 blocks associated with the specified chunk hash.
// The returned blocks are sorted in ascending order by their block number.
func (o *L2Block) GetL2BlocksByChunkHash(ctx context.Context, chunkHash string) ([]*types.WrappedBlock, error) {
	var l2Blocks []L2Block
	db := o.db.WithContext(ctx)
	db = db.Where("chunk_hash = ?", chunkHash)
	db = db.Order("number ASC")

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

// InsertL2Blocks inserts l2 blocks into the "l2_block" table.
func (o *L2Block) InsertL2Blocks(ctx context.Context, blocks []*types.WrappedBlock) error {
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

	if err := o.db.WithContext(ctx).Create(&l2Blocks).Error; err != nil {
		log.Error("failed to insert l2Blocks", "err", err)
		return err
	}
	return nil
}
