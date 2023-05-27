package orm

import (
	"context"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
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
	case types.ProvingTaskProved, types.ProvingTaskVerified:
		updateFields["proved_at"] = time.Now()
	default:
	}
	return updateFields
}

// UpdateProofAndHashByHash update the block batch proof by hash
func (o *BlockBatch) UpdateProofAndHashByHash(ctx context.Context, hash string, proof, instanceCommitments []byte, proofTimeSec uint64, status types.ProvingStatus) error {
	updateFields := o.provingStatus(status)
	updateFields["proof"] = proof
	updateFields["instance_commitments"] = instanceCommitments
	updateFields["proof_time_sec"] = proofTimeSec
	err := o.db.WithContext(ctx).Model(&BlockBatch{}).Where("hash", hash).Updates(updateFields).Error
	if err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return err
}
