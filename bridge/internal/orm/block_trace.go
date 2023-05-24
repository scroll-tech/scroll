package orm

import (
	"encoding/json"
	"errors"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/bridge/internal/types"
)

type BlockTrace struct {
	db *gorm.DB `gorm:"column:-"`

	Number         uint64 `json:"number" db:"number"`
	Hash           string `json:"hash" db:"hash"`
	ParentHash     string `json:"parent_hash" db:"parent_hash"`
	Trace          string `json:"trace" gorm:"column:trace"`
	BatchHash      string `json:"batch_hash" db:"batch_hash"`
	TxNum          uint64 `json:"tx_num" db:"tx_num"`
	GasUsed        uint64 `json:"gas_used" db:"gas_used"`
	BlockTimestamp uint64 `json:"block_timestamp" db:"block_timestamp"`
}

// NewBlockTrace create an blockTraceOrm instance
func NewBlockTrace(db *gorm.DB) *BlockTrace {
	return &BlockTrace{db: db}
}

// TableName define the L1Message table name
func (*BlockTrace) TableName() string {
	return "block_trace"
}

// GetL2BlocksLatestHeight get the l2 blocks latest height
func (o *BlockTrace) GetL2BlocksLatestHeight() (uint64, error) {
	var blockTrace BlockTrace
	err := o.db.Select("COALESCE(MAX(number), -1)").First(&blockTrace).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	return blockTrace.Number, nil
}

// GetL2WrappedBlocks get the l2 wrapped blocks
func (o *BlockTrace) GetL2WrappedBlocks(fields map[string]interface{}) ([]*types.WrappedBlock, error) {
	var blockTraces []BlockTrace
	db := o.db.Select("trace")
	for key, value := range fields {
		db = db.Where(key, value)
	}
	if err := db.Find(&blockTraces).Error; err != nil {
		return nil, err
	}

	var wrappedBlocks []*types.WrappedBlock
	for _, v := range blockTraces {
		var wrappedBlock types.WrappedBlock
		if err := json.Unmarshal([]byte(v.Trace), &wrappedBlock); err != nil {
			break
		}
		wrappedBlocks = append(wrappedBlocks, &wrappedBlock)
	}
	return wrappedBlocks, nil
}

// GetL2BlockInfos get l2 block infos
func (o *BlockTrace) GetL2BlockInfos(fields map[string]interface{}, orderByList []string, limit int) ([]BlockTrace, error) {
	var blockTraces []BlockTrace
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

	if err := db.Find(&blockTraces).Error; err != nil {
		return nil, err
	}
	return blockTraces, nil
}

// GetUnbatchedL2Blocks get unbatched l2 blocks
func (o *BlockTrace) GetUnbatchedL2Blocks(fields map[string]interface{}, orderByList []string, limit int) ([]BlockTrace, error) {
	var unbatchedBlockTraces []BlockTrace
	db := o.db.Select("number, hash, parent_hash, batch_hash, tx_num, gas_used, block_timestamp").Where("batch_hash is NULL")
	for key, value := range fields {
		db = db.Where(key, value)
	}
	if err := db.Find(&unbatchedBlockTraces).Error; err != nil {
		return nil, err
	}
	return unbatchedBlockTraces, nil
}

// InsertWrappedBlocks insert block to block trace
func (o *BlockTrace) InsertWrappedBlocks(blocks []*types.WrappedBlock) error {
	var blockTraces []BlockTrace
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

		tmpBlockTrace := BlockTrace{
			Number:         number,
			Hash:           hash,
			ParentHash:     block.Header.ParentHash.String(),
			Trace:          string(data),
			TxNum:          uint64(txNum),
			GasUsed:        gasCost,
			BlockTimestamp: mtime,
		}
		blockTraces = append(blockTraces, tmpBlockTrace)
	}

	if err := o.db.Create(&blockTraces).Error; err != nil {
		log.Error("failed to insert blockTraces", "err", err)
		return err
	}
	return nil
}

// UpdateBatchHashForL2Blocks update the batch_hash of block trace
func (o *BlockTrace) UpdateBatchHashForL2Blocks(tx *gorm.DB, numbers []uint64, batchHash string) error {
	var db *gorm.DB
	if tx != nil {
		db = tx
	} else {
		db = o.db
	}

	err := db.Model(&BlockTrace{}).Where("number IN (?)", numbers).Update("batch_hash", batchHash).Error
	if err != nil {
		return err
	}
	return nil
}
