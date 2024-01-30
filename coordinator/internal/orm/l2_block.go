package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	rollupTypes "github.com/scroll-tech/go-ethereum/rollup/types"
	"gorm.io/gorm"
)

// L2Block represents a l2 block in the database.
type L2Block struct {
	db *gorm.DB `gorm:"column:-"`

	// block
	Number         uint64 `json:"number" gorm:"number"`
	Hash           string `json:"hash" gorm:"hash"`
	ParentHash     string `json:"parent_hash" gorm:"parent_hash"`
	Header         string `json:"header" gorm:"header"`
	Transactions   string `json:"transactions" gorm:"transactions"`
	WithdrawRoot   string `json:"withdraw_root" gorm:"withdraw_root"`
	StateRoot      string `json:"state_root" gorm:"state_root"`
	TxNum          uint32 `json:"tx_num" gorm:"tx_num"`
	GasUsed        uint64 `json:"gas_used" gorm:"gas_used"`
	BlockTimestamp uint64 `json:"block_timestamp" gorm:"block_timestamp"`
	RowConsumption string `json:"row_consumption" gorm:"row_consumption"`

	// chunk
	ChunkHash string `json:"chunk_hash" gorm:"chunk_hash;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewL2Block creates a new L2Block instance.
func NewL2Block(db *gorm.DB) *L2Block {
	return &L2Block{db: db}
}

// TableName returns the name of the "l2_block" table.
func (*L2Block) TableName() string {
	return "l2_block"
}

// GetL2BlockHashesByChunkHash retrieves the L2 block hashes associated with the specified chunk hash.
// The returned block hashes are sorted in ascending order by their block number.
func (o *L2Block) GetL2BlockHashesByChunkHash(ctx context.Context, chunkHash string) ([]common.Hash, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("header")
	db = db.Where("chunk_hash = ?", chunkHash)
	db = db.Order("number ASC")

	var l2Blocks []L2Block
	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetL2BlockHashesByChunkHash error: %w, chunk hash: %v", err, chunkHash)
	}

	var blockHashes []common.Hash
	for _, v := range l2Blocks {
		var header gethTypes.Header
		if err := json.Unmarshal([]byte(v.Header), &header); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlockHashesByChunkHash error: %w, chunk hash: %v", err, chunkHash)
		}
		blockHashes = append(blockHashes, header.Hash())
	}
	return blockHashes, nil
}

// InsertL2Blocks inserts l2 blocks into the "l2_block" table.
// for unit test
func (o *L2Block) InsertL2Blocks(ctx context.Context, blocks []*rollupTypes.WrappedBlock) error {
	var l2Blocks []L2Block
	for _, block := range blocks {
		header, err := json.Marshal(block.Header)
		if err != nil {
			log.Error("failed to marshal block header", "hash", block.Header.Hash().String(), "err", err)
			return fmt.Errorf("L2Block.InsertL2Blocks error: %w", err)
		}

		l2Block := L2Block{
			Number:         block.Header.Number.Uint64(),
			Hash:           block.Header.Hash().String(),
			ParentHash:     block.Header.ParentHash.String(),
			WithdrawRoot:   block.WithdrawRoot.Hex(),
			StateRoot:      block.Header.Root.Hex(),
			TxNum:          uint32(len(block.Transactions)),
			GasUsed:        block.Header.GasUsed,
			BlockTimestamp: block.Header.Time,
			Header:         string(header),
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

// UpdateChunkHashInRange updates the chunk hash for l2 blocks within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end indices.
// for unit test
func (o *L2Block) UpdateChunkHashInRange(ctx context.Context, startNumber uint64, endNumber uint64, chunkHash string) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Where("number >= ? AND number <= ?", startNumber, endNumber)

	if err := db.Update("chunk_hash", chunkHash).Error; err != nil {
		return fmt.Errorf("L2Block.UpdateChunkHashInRange error: %w, start number: %v, end number: %v, chunk hash: %v",
			err, startNumber, endNumber, chunkHash)
	}
	return nil
}
