package orm

import (
	"context"
	"time"

	"gorm.io/gorm"

	"scroll-tech/common/types"
)

type BlockBatch struct {
	db *gorm.DB `gorm:"-"`

	Hash                string    `json:"hash" gorm:"hash"`
	Index               uint64    `json:"index" gorm:"index"`
	StartBlockNumber    uint64    `json:"start_block_number" gorm:"start_block_number"`
	StartBlockHash      string    `json:"start_block_hash" gorm:"start_block_hash"`
	EndBlockNumber      uint64    `json:"end_block_number" gorm:"end_block_number"`
	EndBlockHash        string    `json:"end_block_hash" gorm:"end_block_hash"`
	ParentHash          string    `json:"parent_hash" gorm:"parent_hash"`
	StateRoot           string    `json:"state_root" gorm:"state_root"`
	TotalTxNum          uint64    `json:"total_tx_num" gorm:"total_tx_num"`
	TotalL1TxNum        uint64    `json:"total_l1_tx_num" gorm:"total_l1_tx_num"`
	TotalL2Gas          uint64    `json:"total_l2_gas" gorm:"total_l2_gas"`
	ProvingStatus       int       `json:"proving_status" gorm:"proving_status"`
	Proof               string    `json:"proof" gorm:"proof"`
	InstanceCommitments string    `json:"instance_commitments" gorm:"instance_commitments"`
	ProofTimeSec        uint64    `json:"proof_time_sec" gorm:"proof_time_sec"`
	RollupStatus        int       `json:"rollup_status" gorm:"rollup_status"`
	CommitTxHash        string    `json:"commit_tx_hash" gorm:"commit_tx_hash"`
	OracleStatus        int       `json:"oracle_status" gorm:"oracle_status"`
	OracleTxHash        string    `json:"oracle_tx_hash" gorm:"oracle_tx_hash"`
	FinalizeTxHash      string    `json:"finalize_tx_hash" gorm:"finalize_tx_hash"`
	CreatedAt           time.Time `json:"created_at" gorm:"created_at"`
	ProverAssignedAt    time.Time `json:"prover_assigned_at" gorm:"prover_assigned_at"`
	ProvedAt            time.Time `json:"proved_at" gorm:"proved_at"`
	CommittedAt         time.Time `json:"committed_at" gorm:"committed_at"`
	FinalizedAt         time.Time `json:"finalized_at" gorm:"finalized_at"`
}

// NewBlockBatch create an blockBatchOrm instance
func NewBlockBatch(db *gorm.DB) *BlockBatch {
	return &BlockBatch{db: db}
}

// TableName define the L1Message table name
func (*BlockBatch) TableName() string {
	return "block_batch"
}

// GetBatchCount get the batch count
func (o *BlockBatch) GetBatchCount() (int64, error) {
	var count int64
	if err := o.db.Model(&BlockBatch{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetBlockBatches get the select block batches
func (o *BlockBatch) GetBlockBatches(fields map[string]interface{}) ([]BlockBatch, error) {
	var blockBatches []BlockBatch
	db := o.db
	for key, value := range fields {
		db.Where(key, value)
	}
	if err := db.Find(&blockBatches).Error; err != nil {
		return nil, err
	}
	return blockBatches, nil
}

// GetVerifiedProofAndInstanceCommitmentsByHash get verified proof and instance comments by hash
func (o *BlockBatch) GetVerifiedProofAndInstanceCommitmentsByHash(hash string) ([]byte, []byte, error) {
	var blockBatch BlockBatch
	err := o.db.Select("proof, instance_commitments").Where("hash", hash).Where("proving_status").Find(&blockBatch).Error
	if err != nil {
		return nil, nil, err
	}
	return []byte(blockBatch.Proof), []byte(blockBatch.InstanceCommitments), nil
}

// GetPendingBatches get the pending batches
func (o *BlockBatch) GetPendingBatches(limit int) ([]string, error) {
	var blockBatches []BlockBatch
	err := o.db.Select("hash").Where("rollup_status", types.RollupPending).Order("index ASC").Limit(limit).Error
	if err != nil {
		return nil, err
	}

	var hashes []string
	for _, v := range blockBatches {
		hashes = append(hashes, v.Hash)
	}
	return hashes, nil
}

// GetLatestBatch get the latest batch
// Need to optimize the query.
func (o *BlockBatch) GetLatestBatch() (*BlockBatch, error) {
	var blockBatch BlockBatch
	subQuery := o.db.Table("block_batch").Select("max(index)")
	err := o.db.Where("index", subQuery).Find(&blockBatch).Error
	if err != nil {
		return nil, err
	}
	return &blockBatch, nil
}

func (o *BlockBatch) GetLatestBatchByRollupStatus(rollupStatuses []types.RollupStatus) (*BlockBatch, error) {
	var blockBatch BlockBatch
	subQuery := o.db.Table("block_batch").Select("max(index)").Where("rollup_status IN (?)", rollupStatuses)
	err := o.db.Where("index", subQuery).Find(&blockBatch).Error
	if err != nil {
		return nil, err
	}
	return &blockBatch, nil
}

//// GetLatestFinalizedBatch get the latest finalized batch
//// Need to optimize the query.
//func (o *BlockBatch) GetLatestFinalizedBatch() (*BlockBatch, error) {
//	var blockBatch BlockBatch
//	subQuery := o.db.Table("block_batch").Select("max(index)").Where("rollup_status", types.RollupFinalized)
//	err := o.db.Where("index", subQuery).Find(&blockBatch).Error
//	if err != nil {
//		return nil, err
//	}
//	return &blockBatch, nil
//}
//
//// GetLatestFinalizingOrFinalizedBatch get the latest finalizing or finalized batch
//func (o *BlockBatch) GetLatestFinalizingOrFinalizedBatch() (*BlockBatch, error) {
//	var blockBatch BlockBatch
//	subQuery := o.db.Table("block_batch").Select("max(index)").Where("rollup_status IN (?)", []interface{}{types.RollupFinalizing, types.RollupFinalized})
//	err := o.db.Where("index", subQuery).Find(&blockBatch).Error
//	if err != nil {
//		return nil, err
//	}
//	return &blockBatch, nil
//}

// GetCommittedBatches get the committed block batches
func (o *BlockBatch) GetCommittedBatches(limit int) ([]string, error) {
	var blockBatches []BlockBatch
	err := o.db.Select("hash").Where("rollup_status", types.RollupCommitted).Order("index ASC").Limit(limit).Error
	if err != nil {
		return nil, err
	}

	var hashes []string
	for _, v := range blockBatches {
		hashes = append(hashes, v.Hash)
	}
	return hashes, nil
}

// GetRollupStatusByHashList get rollup status by hash list
func (o *BlockBatch) GetRollupStatusByHashList(hashes []string) ([]types.RollupStatus, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	var blockBatches []BlockBatch
	err := o.db.Select("hash, rollup_status").Where("hash IN (?)", hashes).Find(&blockBatches).Error
	if err != nil {
		return nil, err
	}

	var statuses []types.RollupStatus
	for _, v := range blockBatches {
		statuses = append(statuses, types.RollupStatus(v.RollupStatus))
	}
	return statuses, nil
}

// UpdateProvingStatus update the proving status
func (o *BlockBatch) UpdateProvingStatus(hash string, status types.ProvingStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = status

	switch status {
	case types.ProvingTaskAssigned:
		updateFields["prover_assigned_at"] = time.Now()
	case types.ProvingTaskUnassigned:
	case types.ProvingTaskProved, types.ProvingTaskVerified:
		updateFields["proved_at"] = time.Now()
	default:
	}

	if err := o.db.Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateRollupStatus update the rollup status
func (o *BlockBatch) UpdateRollupStatus(ctx context.Context, hash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["rollup_status"] = status

	switch status {
	case types.RollupCommitted:
		updateFields["committed_at"] = time.Now()
	case types.RollupFinalized:
		updateFields["finalized_at"] = time.Now()
	}
	if err := o.db.Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateSkippedBatches update the skipped batches
func (o *BlockBatch) UpdateSkippedBatches() (int64, error) {
	provingStatusList := []interface{}{
		types.ProvingTaskSkipped,
		types.ProvingTaskFailed,
	}
	result := o.db.Model(&BlockBatch{}).Where("rollup_status", types.RollupCommitted).
		Where("proving_status IN (?)", provingStatusList).Update("rollup_status", types.RollupFinalizationSkipped)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// UpdateCommitTxHashAndRollupStatus update the commit tx hash and rollup status
func (o *BlockBatch) UpdateCommitTxHashAndRollupStatus(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["commit_tx_hash"] = commitTxHash
	updateFields["rollup_status"] = status
	if status == types.RollupCommitted {
		updateFields["committed_at"] = time.Now()
	}
	if err := o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateFinalizeTxHashAndRollupStatus update the finalize tx hash and rollup status
func (o *BlockBatch) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["finalize_tx_hash"] = finalizeTxHash
	updateFields["rollup_status"] = status
	if status == types.RollupFinalized {
		updateFields["finalized_at"] = time.Now()
	}
	if err := o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateL2GasOracleStatusAndOracleTxHash update the l2 gas oracle status and oracle tx hash
func (o *BlockBatch) UpdateL2GasOracleStatusAndOracleTxHash(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
	updateFields := make(map[string]interface{})
	updateFields["oracle_status"] = status
	updateFields["oracle_tx_hash"] = txHash
	if err := o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}
