package orm

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	coordinatorType "scroll-tech/coordinator/internal/types"
)

// BlockBatch is structure of stored block batch message
type BlockBatch struct {
	db *gorm.DB `gorm:"column:-"`

	Hash             string     `json:"hash" gorm:"column:hash"`
	Index            uint64     `json:"index" gorm:"column:index"`
	StartBlockNumber uint64     `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash   string     `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber   uint64     `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash     string     `json:"end_block_hash" gorm:"column:end_block_hash"`
	ParentHash       string     `json:"parent_hash" gorm:"column:parent_hash"`
	StateRoot        string     `json:"state_root" gorm:"column:state_root"`
	TotalTxNum       uint64     `json:"total_tx_num" gorm:"column:total_tx_num"`
	TotalL1TxNum     uint64     `json:"total_l1_tx_num" gorm:"column:total_l1_tx_num"`
	TotalL2Gas       uint64     `json:"total_l2_gas" gorm:"column:total_l2_gas"`
	ProvingStatus    int        `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof            []byte     `json:"proof" gorm:"column:proof"`
	ProofTimeSec     uint64     `json:"proof_time_sec" gorm:"column:proof_time_sec;default:0"`
	RollupStatus     int        `json:"rollup_status" gorm:"column:rollup_status;default:1"`
	CommitTxHash     string     `json:"commit_tx_hash" gorm:"column:commit_tx_hash;default:NULL"`
	OracleStatus     int        `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash     string     `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash;default:NULL"`
	FinalizeTxHash   string     `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash;default:NULL"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP()"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	CommittedAt      *time.Time `json:"committed_at" gorm:"column:committed_at;default:NULL"`
	FinalizedAt      *time.Time `json:"finalized_at" gorm:"column:finalized_at;default:NULL"`
}

// NewBlockBatch create an blockBatchOrm instance
func NewBlockBatch(db *gorm.DB) *BlockBatch {
	return &BlockBatch{db: db}
}

// TableName define the BlockBatch table name
func (*BlockBatch) TableName() string {
	return "block_batch"
}

// GetBlockBatches get the select block batches
func (o *BlockBatch) GetBlockBatches(fields map[string]interface{}, orderByList []string, limit int) ([]BlockBatch, error) {
	var blockBatches []BlockBatch
	db := o.db
	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit != 0 {
		db = db.Limit(limit)
	}

	if err := db.Find(&blockBatches).Error; err != nil {
		return nil, err
	}
	return blockBatches, nil
}

// GetAssignedBatchHashes get the hashes of block_batch where proving_status in (types.ProvingTaskAssigned, types.ProvingTaskProved)
func (o *BlockBatch) GetAssignedBatchHashes() ([]string, error) {
	var blockBatches []BlockBatch
	err := o.db.Select("hash").Where("proving_status IN (?)", []types.ProvingStatus{types.ProvingTaskAssigned, types.ProvingTaskProved}).Find(&blockBatches).Error
	if err != nil {
		return nil, err
	}
	var hashes []string
	for _, blockBatch := range blockBatches {
		hashes = append(hashes, blockBatch.Hash)
	}
	return hashes, nil
}

// InsertBlockBatchByBatchData insert a block batch data by the BatchData
func (o *BlockBatch) InsertBlockBatchByBatchData(tx *gorm.DB, batchData *coordinatorType.BatchData) (int64, error) {
	var db *gorm.DB
	if tx != nil {
		db = tx
	} else {
		db = o.db
	}

	numBlocks := len(batchData.Batch.Blocks)
	insertBlockBatch := BlockBatch{
		Hash:             batchData.Hash().Hex(),
		Index:            batchData.Batch.BatchIndex,
		StartBlockNumber: batchData.Batch.Blocks[0].BlockNumber,
		StartBlockHash:   batchData.Batch.Blocks[0].BlockHash.Hex(),
		EndBlockNumber:   batchData.Batch.Blocks[numBlocks-1].BlockNumber,
		EndBlockHash:     batchData.Batch.Blocks[numBlocks-1].BlockHash.Hex(),
		ParentHash:       batchData.Batch.ParentBatchHash.Hex(),
		StateRoot:        batchData.Batch.NewStateRoot.Hex(),
		TotalTxNum:       batchData.TotalTxNum,
		TotalL1TxNum:     batchData.TotalL1TxNum,
		TotalL2Gas:       batchData.TotalL2Gas,
		CreatedAt:        time.Now(),
	}
	result := db.Create(&insertBlockBatch)
	if result.Error != nil {
		log.Error("failed to insert block batch by batchData", "err", result.Error)
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// UpdateProvingStatus update the proving status
func (o *BlockBatch) UpdateProvingStatus(hash string, status types.ProvingStatus) error {
	updateFields := o.provingStatus(status)
	if err := o.db.Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

func (o *BlockBatch) provingStatus(status types.ProvingStatus) map[string]interface{} {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = int(status)

	switch status {
	case types.ProvingTaskAssigned:
		updateFields["prover_assigned_at"] = time.Now()
	case types.ProvingTaskUnassigned:
		updateFields["prover_assigned_at"] = nil
	case types.ProvingTaskProved, types.ProvingTaskVerified:
		updateFields["proved_at"] = time.Now()
	default:
	}
	return updateFields
}

// UpdateProofAndHashByHash update the block batch proof by hash
func (o *BlockBatch) UpdateProofAndHashByHash(ctx context.Context, hash string, proof []byte, proofTimeSec uint64, status types.ProvingStatus) error {
	updateFields := o.provingStatus(status)
	updateFields["proof"] = proof
	updateFields["proof_time_sec"] = proofTimeSec
	err := o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error
	if err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return err
}
