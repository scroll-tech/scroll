package orm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
)

const defaultBatchHeaderVersion = 0

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

// GetBatches retrieves selected batches from the database.
// The returned batches are sorted in ascending order by their index.
func (o *Batch) GetBatches(ctx context.Context, fields map[string]interface{}, orderByList []string, limit int) ([]*Batch, error) {
	db := o.db.WithContext(ctx)

	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit > 0 {
		db = db.Limit(limit)
	}

	db = db.Order("index ASC")

	var batches []*Batch
	if err := db.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// GetBatchCount retrieves the total number of batches in the database.
func (o *Batch) GetBatchCount(ctx context.Context) (uint64, error) {
	var count int64
	err := o.db.WithContext(ctx).Model(&Batch{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return uint64(count), nil
}

// GetVerifiedProofByHash retrieves the verified aggregate proof for a batch with the given hash.
func (o *Batch) GetVerifiedProofByHash(ctx context.Context, hash string) (*message.AggProof, error) {
	var batch Batch
	err := o.db.WithContext(ctx).Where("hash = ? AND proving_status = ?", hash, types.ProvingTaskVerified).First(&batch).Error
	if err != nil {
		return nil, err
	}

	var proof message.AggProof
	err = json.Unmarshal(batch.Proof, &proof)
	if err != nil {
		return nil, err
	}

	return &proof, nil
}

// GetLatestBatch retrieves the latest batch from the database.
func (o *Batch) GetLatestBatch(ctx context.Context) (*Batch, error) {
	var latestBatch Batch
	err := o.db.WithContext(ctx).Order("index desc").First(&latestBatch).Error
	if err != nil {
		return nil, err
	}
	return &latestBatch, nil
}

// GetRollupStatusByHashList retrieves the rollup statuses for a list of batch hashes.
func (o *Batch) GetRollupStatusByHashList(ctx context.Context, hashes []string) ([]types.RollupStatus, error) {
	if len(hashes) == 0 {
		return []types.RollupStatus{}, nil
	}

	var batches []Batch
	err := o.db.WithContext(ctx).Where("hash IN ?", hashes).Find(&batches).Error
	if err != nil {
		return nil, err
	}

	hashToStatusMap := make(map[string]types.RollupStatus)
	for _, batch := range batches {
		hashToStatusMap[batch.Hash] = types.RollupStatus(batch.RollupStatus)
	}

	var statuses []types.RollupStatus
	for _, hash := range hashes {
		status, ok := hashToStatusMap[hash]
		if !ok {
			return nil, fmt.Errorf("hash not found in database: %s", hash)
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetPendingBatches retrieves pending batches up to the specified limit.
// The returned batches are sorted in ascending order by their index.
func (o *Batch) GetPendingBatches(ctx context.Context, limit int) ([]*Batch, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be greater than zero")
	}

	var batches []*Batch
	db := o.db.WithContext(ctx)

	db = db.Where("rollup_status = ?", types.RollupPending).Order("index ASC").Limit(limit)

	if err := db.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

// GetBatchByIndex retrieves the batch by the given index.
func (o *Batch) GetBatchByIndex(ctx context.Context, index uint64) (*Batch, error) {
	var batch Batch
	err := o.db.WithContext(ctx).Where("index = ?", index).First(&batch).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// InsertBatch inserts a new batch into the database.
func (o *Batch) InsertBatch(ctx context.Context, startChunkIndex, endChunkIndex uint64, startChunkHash, endChunkHash string, chunks []*types.Chunk, dbTX ...*gorm.DB) (*Batch, error) {
	if len(chunks) == 0 {
		return nil, errors.New("invalid args")
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	parentBatch, err := o.GetLatestBatch(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("failed to get the latest batch", "err", err)
		return nil, err
	}

	var batchIndex uint64
	var parentBatchHash common.Hash
	var totalL1MessagePoppedBefore uint64
	var version uint8 = defaultBatchHeaderVersion

	// if parentBatch==nil then err==gorm.ErrRecordNotFound, which means there's
	// not batch record in the db, we then use default empty values for the creating batch;
	// if parentBatch!=nil then err=nil, then we fill the parentBatch-related data into the creating batch
	if parentBatch != nil {
		batchIndex = parentBatch.Index + 1
		parentBatchHash = common.HexToHash(parentBatch.Hash)

		var parentBatchHeader *types.BatchHeader
		parentBatchHeader, err = types.DecodeBatchHeader(parentBatch.BatchHeader)
		if err != nil {
			log.Error("failed to decode parent batch header", "index", parentBatch.Index, "hash", parentBatch.Hash, "err", err)
			return nil, err
		}

		totalL1MessagePoppedBefore = parentBatchHeader.TotalL1MessagePopped()
		version = parentBatchHeader.Version()
	}

	batchHeader, err := types.NewBatchHeader(version, batchIndex, totalL1MessagePoppedBefore, parentBatchHash, chunks)
	if err != nil {
		log.Error("failed to create batch header",
			"index", batchIndex, "total l1 message popped before", totalL1MessagePoppedBefore,
			"parent hash", parentBatchHash, "number of chunks", len(chunks), "err", err)
		return nil, err
	}

	numChunks := len(chunks)
	lastChunkBlockNum := len(chunks[numChunks-1].Blocks)

	newBatch := Batch{
		Index:           batchIndex,
		Hash:            batchHeader.Hash().Hex(),
		StartChunkHash:  startChunkHash,
		StartChunkIndex: startChunkIndex,
		EndChunkHash:    endChunkHash,
		EndChunkIndex:   endChunkIndex,
		StateRoot:       chunks[numChunks-1].Blocks[lastChunkBlockNum-1].Header.Root.Hex(),
		WithdrawRoot:    chunks[numChunks-1].Blocks[lastChunkBlockNum-1].WithdrawTrieRoot.Hex(),
		BatchHeader:     batchHeader.Encode(),
		ProvingStatus:   int16(types.ProvingTaskUnassigned),
		RollupStatus:    int16(types.RollupPending),
	}

	if err := db.WithContext(ctx).Create(&newBatch).Error; err != nil {
		log.Error("failed to insert batch", "batch", newBatch, "err", err)
		return nil, err
	}

	return &newBatch, nil
}

// UpdateSkippedBatches updates the skipped batches in the database.
func (o *Batch) UpdateSkippedBatches(ctx context.Context) (uint64, error) {
	provingStatusList := []interface{}{
		int(types.ProvingTaskSkipped),
		int(types.ProvingTaskFailed),
	}
	result := o.db.Model(&Batch{}).Where("rollup_status", int(types.RollupCommitted)).
		Where("proving_status IN (?)", provingStatusList).Update("rollup_status", int(types.RollupFinalizationSkipped))
	if result.Error != nil {
		return 0, result.Error
	}
	return uint64(result.RowsAffected), nil
}

// UpdateL2GasOracleStatusAndOracleTxHash updates the L2 gas oracle status and transaction hash for a batch.
func (o *Batch) UpdateL2GasOracleStatusAndOracleTxHash(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
	updateFields := make(map[string]interface{})
	updateFields["oracle_status"] = int(status)
	updateFields["oracle_tx_hash"] = txHash
	if err := o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
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

	if err := db.Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateRollupStatus updates the rollup status of a batch.
func (o *Batch) UpdateRollupStatus(ctx context.Context, hash string, status types.RollupStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	updateFields := make(map[string]interface{})
	updateFields["rollup_status"] = int(status)

	switch status {
	case types.RollupCommitted:
		updateFields["committed_at"] = time.Now()
	case types.RollupFinalized:
		updateFields["finalized_at"] = time.Now()
	}
	if err := db.Model(&Batch{}).WithContext(ctx).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateCommitTxHashAndRollupStatus updates the commit transaction hash and rollup status for a batch.
func (o *Batch) UpdateCommitTxHashAndRollupStatus(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["commit_tx_hash"] = commitTxHash
	updateFields["rollup_status"] = int(status)
	if status == types.RollupCommitted {
		updateFields["committed_at"] = time.Now()
	}
	if err := o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateFinalizeTxHashAndRollupStatus updates the finalize transaction hash and rollup status for a batch.
func (o *Batch) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["finalize_tx_hash"] = finalizeTxHash
	updateFields["rollup_status"] = int(status)
	if status == types.RollupFinalized {
		updateFields["finalized_at"] = time.Now()
	}
	if err := o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateProofByHash updates the block batch proof by hash.
// for unit test.
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
