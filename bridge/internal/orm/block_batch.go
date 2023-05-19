package orm

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"

	bridgeTypes "scroll-tech/bridge/internal/types"
)

type BlockBatch struct {
	db *gorm.DB `gorm:"column:-"`

	Hash                string    `json:"hash" gorm:"column:hash"`
	Index               uint64    `json:"index" gorm:"column:index"`
	StartBlockNumber    uint64    `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash      string    `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber      uint64    `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash        string    `json:"end_block_hash" gorm:"column:end_block_hash"`
	ParentHash          string    `json:"parent_hash" gorm:"column:parent_hash"`
	StateRoot           string    `json:"state_root" gorm:"column:state_root"`
	TotalTxNum          uint64    `json:"total_tx_num" gorm:"column:total_tx_num"`
	TotalL1TxNum        uint64    `json:"total_l1_tx_num" gorm:"column:total_l1_tx_num"`
	TotalL2Gas          uint64    `json:"total_l2_gas" gorm:"column:total_l2_gas"`
	ProvingStatus       int       `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof               string    `json:"proof" gorm:"column:proof"`
	InstanceCommitments string    `json:"instance_commitments" gorm:"column:instance_commitments"`
	ProofTimeSec        uint64    `json:"proof_time_sec" gorm:"column:proof_time_sec;default:0"`
	RollupStatus        int       `json:"rollup_status" gorm:"column:rollup_status;default:1"`
	CommitTxHash        string    `json:"commit_tx_hash" gorm:"column:commit_tx_hash"`
	OracleStatus        int       `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash        string    `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash"`
	FinalizeTxHash      string    `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash"`
	CreatedAt           time.Time `json:"created_at" gorm:"column:created_at"`
	ProverAssignedAt    time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at"`
	ProvedAt            time.Time `json:"proved_at" gorm:"column:proved_at"`
	CommittedAt         time.Time `json:"committed_at" gorm:"column:committed_at"`
	FinalizedAt         time.Time `json:"finalized_at" gorm:"column:finalized_at"`
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
func (o *BlockBatch) GetBlockBatches(fields map[string]interface{}, orderByList []string, limit int) ([]BlockBatch, error) {
	var blockBatches []BlockBatch
	db := o.db
	for key, value := range fields {
		db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db.Order(orderBy)
	}

	if limit != 0 {
		db.Limit(limit)
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
	var tmpRollupStatus []int
	for _, v := range rollupStatuses {
		tmpRollupStatus = append(tmpRollupStatus, int(v))
	}
	subQuery := o.db.Table("block_batch").Select("max(index)").Where("rollup_status IN (?)", tmpRollupStatus)
	err := o.db.Where("index = (?)", subQuery).Find(&blockBatch).Error
	if err != nil {
		return nil, err
	}
	return &blockBatch, nil
}

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

// InsertBlockBatchByBatchData insert a block batch data by the BatchData
func (o *BlockBatch) InsertBlockBatchByBatchData(tx *gorm.DB, batchData *bridgeTypes.BatchData) (int64, error) {
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
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = int(status)

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
	updateFields["rollup_status"] = int(status)

	switch status {
	case types.RollupCommitted:
		updateFields["committed_at"] = time.Now()
	case types.RollupFinalized:
		updateFields["finalized_at"] = time.Now()
	}
	if err := o.db.Model(&BlockBatch{}).WithContext(ctx).Where("hash", hash).Updates(updateFields).Error; err != nil {
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
	updateFields["rollup_status"] = int(status)
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
	updateFields["rollup_status"] = int(status)
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
	updateFields["oracle_status"] = int(status)
	updateFields["oracle_tx_hash"] = txHash
	if err := o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateProofByHash update the block batch proof by hash
// for unit test
func (o *BlockBatch) UpdateProofByHash(ctx context.Context, hash string, proof, instanceCommitments []byte, proofTimeSec uint64) error {
	updateFields := make(map[string]interface{})
	updateFields["proof"] = proof
	updateFields["instance_commitments"] = instanceCommitments
	updateFields["proof_time_sec"] = proofTimeSec
	err := o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error
	if err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return err
}
