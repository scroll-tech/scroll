package orm

import (
	"context"
	"encoding/json"
	"time"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"gorm.io/gorm"
)

// Batch represents a batch of chunks.
type Batch struct {
	db *gorm.DB `gorm:"column:-"`

	// batch
	Index           uint64 `json:"index" gorm:"column:index"`
	Hash            string `json:"hash" gorm:"column:hash"`
	StartChunkIndex uint64 `json:"start_chunk_index" gorm:"column:start_chunk_index"`
	StartChunkHash  string `json:"start_chunk_hash" gorm:"column:start_chunk_hash"`
	EndChunkIndex   uint64 `json:"end_chunk_index" gorm:"column:end_chunk_index"`
	EndChunkHash    string `json:"end_chunk_hash" gorm:"column:end_chunk_hash"`
	StateRoot       string `json:"state_root" gorm:"column:state_root"`
	WithdrawRoot    string `json:"withdraw_root" gorm:"column:withdraw_root"`
	BatchHeader     []byte `json:"batch_header" gorm:"column:batch_header"`

	// proof
	ProvingStatus    int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof            []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec     int        `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`

	// rollup
	RollupStatus   int16      `json:"rollup_status" gorm:"column:rollup_status;default:1"`
	CommitTxHash   string     `json:"commit_tx_hash" gorm:"column:commit_tx_hash;default:NULL"`
	CommittedAt    *time.Time `json:"committed_at" gorm:"column:committed_at;default:NULL"`
	FinalizeTxHash string     `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash;default:NULL"`
	FinalizedAt    *time.Time `json:"finalized_at" gorm:"column:finalized_at;default:NULL"`

	// gas oracle
	OracleStatus int16  `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash string `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewBatch creates a new Batch database instance.
func NewBatch(db *gorm.DB) *Batch {
	return &Batch{db: db}
}

// TableName returns the table name for the Batch model.
func (*Batch) TableName() string {
	return "batch"
}

// GetPendingBatches retrieves all or limited number of pending batches based on the specified limit.
// The returned batches are sorted in ascending order by their index.
func (o *Batch) GetPendingBatches(ctx context.Context) ([]*Batch, error) {
	var batches []*Batch
	db := o.db.WithContext(ctx)
	db = db.Where("rollup_status = ?", types.RollupPending)
	db = db.Order("index ASC")

	if err := db.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// GetAssignedBatches retrieves all batches whose proving_status is either types.ProvingTaskAssigned or types.ProvingTaskProved.
func (o *Batch) GetAssignedBatches(ctx context.Context) ([]*Batch, error) {
	var assignedBatches []*Batch
	err := o.db.WithContext(ctx).
		Where("proving_status IN (?)", []int{int(types.ProvingTaskAssigned), int(types.ProvingTaskProved)}).
		Find(&assignedBatches).Error
	if err != nil {
		return nil, err
	}
	return assignedBatches, nil
}

// UpdateProvingStatus updates the proving status of a batch.
func (o *Batch) UpdateProvingStatus(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

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

	if err := db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateProofByHash updates the batch proof by hash.
func (o *Batch) UpdateProofByHash(ctx context.Context, hash string, proof *message.AggProof, proofTimeSec uint64) error {
	proofBytes, err := json.Marshal(proof)
	if err != nil {
		return err
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proofBytes
	updateFields["proof_time_sec"] = proofTimeSec
	err = o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error
	return err
}
