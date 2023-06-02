package orm

import "gorm.io/gorm"

// BlockTrace is structure of stored block trace message
type BlockTrace struct {
	db *gorm.DB `gorm:"column:-"`

	Number         uint64 `json:"number" gorm:"number"`
	Hash           string `json:"hash" gorm:"hash"`
	ParentHash     string `json:"parent_hash" gorm:"parent_hash"`
	Trace          string `json:"trace" gorm:"column:trace"`
	BatchHash      string `json:"batch_hash" gorm:"batch_hash;default:NULL"`
	TxNum          uint64 `json:"tx_num" gorm:"tx_num"`
	GasUsed        uint64 `json:"gas_used" gorm:"gas_used"`
	BlockTimestamp uint64 `json:"block_timestamp" gorm:"block_timestamp"`
}

// NewBlockTrace create an blockTraceOrm instance
func NewBlockTrace(db *gorm.DB) *BlockTrace {
	return &BlockTrace{db: db}
}

// TableName define the BlockTrace table name
func (*BlockTrace) TableName() string {
	return "block_trace"
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
