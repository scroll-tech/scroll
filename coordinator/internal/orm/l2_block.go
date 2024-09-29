package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
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

// GetL2BlockByNumber retrieves the L2 block by l2 block number
func (o *L2Block) GetL2BlockByNumber(ctx context.Context, blockNumber uint64) (*L2Block, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Where("number = ?", blockNumber)

	var l2Block L2Block
	if err := db.First(&l2Block).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetL2BlockByNumber error: %w, chunk block number: %v", err, blockNumber)
	}
	return &l2Block, nil
}

// GetL2BlocksInRange retrieves the L2 blocks within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end block numbers.
// The returned blocks are sorted in ascending order by their block number.
func (o *L2Block) GetL2BlocksInRange(ctx context.Context, startBlockNumber uint64, endBlockNumber uint64) ([]*encoding.Block, error) {
	if startBlockNumber > endBlockNumber {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange: start block number should be less than or equal to end block number, start block: %v, end block: %v", startBlockNumber, endBlockNumber)
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("header, transactions, withdraw_root, row_consumption")
	db = db.Where("number >= ? AND number <= ?", startBlockNumber, endBlockNumber)
	db = db.Order("number ASC")

	var l2Blocks []L2Block
	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
	}

	// sanity check
	if uint64(len(l2Blocks)) != endBlockNumber-startBlockNumber+1 {
		return nil, fmt.Errorf("L2Block.GetL2BlocksInRange: unexpected number of results, expected: %v, got: %v", endBlockNumber-startBlockNumber+1, len(l2Blocks))
	}

	var blocks []*encoding.Block
	for _, v := range l2Blocks {
		var block encoding.Block

		if err := json.Unmarshal([]byte(v.Transactions), &block.Transactions); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
		}

		block.Header = &gethTypes.Header{}
		if err := json.Unmarshal([]byte(v.Header), block.Header); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
		}

		block.WithdrawRoot = common.HexToHash(v.WithdrawRoot)

		if err := json.Unmarshal([]byte(v.RowConsumption), &block.RowConsumption); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
		}

		blocks = append(blocks, &block)
	}

	return blocks, nil
}

// InsertL2Blocks inserts l2 blocks into the "l2_block" table.
// for unit test
func (o *L2Block) InsertL2Blocks(ctx context.Context, blocks []*encoding.Block) error {
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

		rc, err := json.Marshal(block.RowConsumption)
		if err != nil {
			log.Error("failed to marshal RowConsumption", "hash", block.Header.Hash().String(), "err", err)
			return fmt.Errorf("L2Block.InsertL2Blocks error: %w, block hash: %v", err, block.Header.Hash().String())
		}

		l2Block := L2Block{
			Number:         block.Header.Number.Uint64(),
			Hash:           block.Header.Hash().String(),
			ParentHash:     block.Header.ParentHash.String(),
			Transactions:   string(txs),
			WithdrawRoot:   block.WithdrawRoot.Hex(),
			StateRoot:      block.Header.Root.Hex(),
			TxNum:          uint32(len(block.Transactions)),
			GasUsed:        block.Header.GasUsed,
			BlockTimestamp: block.Header.Time,
			Header:         string(header),
			RowConsumption: string(rc),
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
