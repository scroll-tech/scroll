package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	gethTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
)

// L2Block represents a l2 block in the database.
type L2Block struct {
	db *gorm.DB `gorm:"column:-"`

	// block
	Number             uint64 `json:"number" gorm:"number"`
	Hash               string `json:"hash" gorm:"hash"`
	ParentHash         string `json:"parent_hash" gorm:"parent_hash"`
	Header             string `json:"header" gorm:"header"`
	Transactions       string `json:"transactions" gorm:"transactions"`
	WithdrawRoot       string `json:"withdraw_root" gorm:"withdraw_root"`
	StateRoot          string `json:"state_root" gorm:"state_root"`
	TxNum              uint32 `json:"tx_num" gorm:"tx_num"`
	GasUsed            uint64 `json:"gas_used" gorm:"gas_used"`
	BlockTimestamp     uint64 `json:"block_timestamp" gorm:"block_timestamp"`
	RowConsumption     string `json:"row_consumption" gorm:"row_consumption"`
	LastAppliedL1Block uint64 `json:"last_applied_l1_block" gorm:"last_applied_l1_block"`

	// chunk
	ChunkHash string `json:"chunk_hash" gorm:"chunk_hash;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
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
func (o *L2Block) GetL2BlocksLatestHeight(ctx context.Context) (uint64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("COALESCE(MAX(number), 0)")

	var maxNumber uint64
	if err := db.Row().Scan(&maxNumber); err != nil {
		return 0, fmt.Errorf("L2Block.GetL2BlocksLatestHeight error: %w", err)
	}
	return maxNumber, nil
}

func (o *L2Block) GetL2BlocksLastAppliedL1BlockNumber(ctx context.Context) (uint64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("COALESCE(MAX(last_applied_l1_block), 0)")

	var lastAppliedL1Number uint64
	if err := db.Row().Scan(&lastAppliedL1Number); err != nil {
		return 0, fmt.Errorf("L2Block.GetL2BlocksLastAppliedL1BlockNumber error: %w", err)
	}
	return lastAppliedL1Number, nil
}

// GetL2WrappedBlocksGEHeight retrieves L2 blocks that have a block number greater than or equal to the given height.
// The blocks are converted into WrappedBlock format for output.
// The returned blocks are sorted in ascending order by their block number.
func (o *L2Block) GetL2WrappedBlocksGEHeight(ctx context.Context, height uint64, limit int) ([]*types.WrappedBlock, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&L2Block{})
	db = db.Select("header, transactions, withdraw_root, row_consumption, last_applied_l1_block")
	db = db.Where("number >= ?", height)
	db = db.Order("number ASC")

	if limit > 0 {
		db = db.Limit(limit)
	}

	var l2Blocks []L2Block
	if err := db.Find(&l2Blocks).Error; err != nil {
		return nil, fmt.Errorf("L2Block.GetL2WrappedBlocksGEHeight error: %w", err)
	}

	var wrappedBlocks []*types.WrappedBlock
	for _, v := range l2Blocks {
		var wrappedBlock types.WrappedBlock

		if err := json.Unmarshal([]byte(v.Transactions), &wrappedBlock.Transactions); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2WrappedBlocksGEHeight error: %w", err)
		}

		wrappedBlock.Header = &gethTypes.Header{}
		if err := json.Unmarshal([]byte(v.Header), wrappedBlock.Header); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2WrappedBlocksGEHeight error: %w", err)
		}

		wrappedBlock.WithdrawRoot = common.HexToHash(v.WithdrawRoot)

		if err := json.Unmarshal([]byte(v.RowConsumption), &wrappedBlock.RowConsumption); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2WrappedBlocksGEHeight error: %w", err)
		}

		wrappedBlock.LastAppliedL1Block = v.LastAppliedL1Block

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
	db = db.Select("header, transactions, withdraw_root, row_consumption, last_applied_l1_block")
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

		wrappedBlock.WithdrawRoot = common.HexToHash(v.WithdrawRoot)

		if err := json.Unmarshal([]byte(v.RowConsumption), &wrappedBlock.RowConsumption); err != nil {
			return nil, fmt.Errorf("L2Block.GetL2BlocksInRange error: %w, start block: %v, end block: %v", err, startBlockNumber, endBlockNumber)
		}

		wrappedBlock.LastAppliedL1Block = v.LastAppliedL1Block

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

		rc, err := json.Marshal(block.RowConsumption)
		if err != nil {
			log.Error("failed to marshal RowConsumption", "hash", block.Header.Hash().String(), "err", err)
			return fmt.Errorf("L2Block.InsertL2Blocks error: %w", err)
		}

		l2Block := L2Block{
			Number:             block.Header.Number.Uint64(),
			Hash:               block.Header.Hash().String(),
			ParentHash:         block.Header.ParentHash.String(),
			Transactions:       string(txs),
			WithdrawRoot:       block.WithdrawRoot.Hex(),
			StateRoot:          block.Header.Root.Hex(),
			TxNum:              uint32(len(block.Transactions)),
			GasUsed:            block.Header.GasUsed,
			BlockTimestamp:     block.Header.Time,
			RowConsumption:     string(rc),
			Header:             string(header),
			LastAppliedL1Block: block.LastAppliedL1Block,
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
		return fmt.Errorf("L2Block.UpdateChunkHashInRange: incorrect number of rows affected, expected: %v, got: %v", endIndex-startIndex+1, tx.RowsAffected)
	}

	return nil
}
