package orm

import "gorm.io/gorm"

type BlockTrace struct {
	db *gorm.DB `gorm:"-"`

	Number         uint64 `json:"number" db:"number"`
	Hash           string `json:"hash" db:"hash"`
	ParentHash     string `json:"parent_hash" db:"parent_hash"`
	Trace          string `json:"trace" gorm:"trace"`
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

// GetL2BlockInfos get l2 block infos
func (o *BlockTrace) GetL2BlockInfos(fields map[string]interface{}, orderByList []string, limit int) ([]BlockTrace, error) {
	var blockTraces []BlockTrace
	db := o.db.Select("number, hash, parent_hash, batch_hash, tx_num, gas_used, block_timestamp")
	for key, value := range fields {
		db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db.Order(orderBy)
	}

	if limit != 0 {
		db.Limit(limit)
	}

	if err := db.Find(&blockTraces).Error; err != nil {
		return nil, err
	}
	return blockTraces, nil
}
