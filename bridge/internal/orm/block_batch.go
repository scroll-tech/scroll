package orm

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	bridgeTypes "scroll-tech/bridge/internal/types"
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

// GetBlockBatchesHashByRollupStatus get the block batches by rollup status
func (o *BlockBatch) GetBlockBatchesHashByRollupStatus(status types.RollupStatus, limit int) ([]string, error) {
	var blockBatches []BlockBatch
	err := o.db.Select("hash").Where("rollup_status", int(status)).Order("index ASC").Limit(limit).Find(&blockBatches).Error
	if err != nil {
		return nil, err
	}

	var hashes []string
	for _, v := range blockBatches {
		hashes = append(hashes, v.Hash)
	}
	return hashes, nil
}

// GetVerifiedProofByHash get verified proof and instance comments by hash
func (o *BlockBatch) GetVerifiedProofByHash(hash string) (*message.AggProof, error) {
	result := o.db.Model(&BlockBatch{}).Select("proof").Where("hash", hash).Where("proving_status", int(types.ProvingTaskVerified)).Row()
	if result.Err() != nil {
		return nil, result.Err()
	}

	var proofBytes []byte
	if err := result.Scan(&proofBytes); err != nil {
		return nil, err
	}

	var proof message.AggProof
	if err := json.Unmarshal(proofBytes, &proof); err != nil {
		return nil, err
	}

	return &proof, nil
}

// GetLatestBatch get the latest batch
// because we will `initializeGenesis()` when we start the `L2Watcher`, so a batch must exist.
func (o *BlockBatch) GetLatestBatch() (*BlockBatch, error) {
	var blockBatch BlockBatch
	err := o.db.Order("index DESC").Limit(1).First(&blockBatch).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &blockBatch, nil
}

// GetLatestBatchByRollupStatus get the latest block batch by rollup status
func (o *BlockBatch) GetLatestBatchByRollupStatus(rollupStatuses []types.RollupStatus) (*BlockBatch, error) {
	var tmpRollupStatus []int
	for _, v := range rollupStatuses {
		tmpRollupStatus = append(tmpRollupStatus, int(v))
	}
	var blockBatch BlockBatch
	err := o.db.Where("rollup_status IN (?)", tmpRollupStatus).Order("index DESC").Limit(1).First(&blockBatch).Error
	if err != nil {
		return nil, err
	}
	return &blockBatch, nil
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

	var (
		statuses   []types.RollupStatus
		_statusMap = make(map[string]types.RollupStatus, len(hashes))
	)
	for _, _batch := range blockBatches {
		_statusMap[_batch.Hash] = types.RollupStatus(_batch.RollupStatus)
	}
	for _, _hash := range hashes {
		statuses = append(statuses, _statusMap[_hash])
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
		updateFields["prover_assigned_at"] = nil
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
		int(types.ProvingTaskSkipped),
		int(types.ProvingTaskFailed),
	}
	result := o.db.Model(&BlockBatch{}).Where("rollup_status", int(types.RollupCommitted)).
		Where("proving_status IN (?)", provingStatusList).Update("rollup_status", int(types.RollupFinalizationSkipped))
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
func (o *BlockBatch) UpdateProofByHash(ctx context.Context, hash string, proof *message.AggProof, proofTimeSec uint64) error {
	proofBytes, err := json.Marshal(proof)
	if err != nil {
		return err
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proofBytes
	updateFields["proof_time_sec"] = proofTimeSec
	err = o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error
	if err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return err
}
