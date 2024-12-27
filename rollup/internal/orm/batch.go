package orm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"

	rutils "scroll-tech/rollup/internal/utils"
)

// Batch represents a batch of chunks.
type Batch struct {
	db *gorm.DB `gorm:"column:-"`

	// batch
	Index           uint64 `json:"index" gorm:"column:index"`
	Hash            string `json:"hash" gorm:"column:hash"`
	DataHash        string `json:"data_hash" gorm:"column:data_hash"`
	StartChunkIndex uint64 `json:"start_chunk_index" gorm:"column:start_chunk_index"`
	StartChunkHash  string `json:"start_chunk_hash" gorm:"column:start_chunk_hash"`
	EndChunkIndex   uint64 `json:"end_chunk_index" gorm:"column:end_chunk_index"`
	EndChunkHash    string `json:"end_chunk_hash" gorm:"column:end_chunk_hash"`
	StateRoot       string `json:"state_root" gorm:"column:state_root"`
	WithdrawRoot    string `json:"withdraw_root" gorm:"column:withdraw_root"`
	ParentBatchHash string `json:"parent_batch_hash" gorm:"column:parent_batch_hash"`
	BatchHeader     []byte `json:"batch_header" gorm:"column:batch_header"`
	CodecVersion    int16  `json:"codec_version" gorm:"column:codec_version"`
	EnableCompress  bool   `json:"enable_compress" gorm:"column:enable_compress"` // use for debug
	BlobBytes       []byte `json:"blob_bytes" gorm:"column:blob_bytes"`

	// proof
	ChunkProofsStatus int16      `json:"chunk_proofs_status" gorm:"column:chunk_proofs_status;default:1"`
	ProvingStatus     int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof             []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt  *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt          *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec      int32      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`

	// rollup
	RollupStatus   int16      `json:"rollup_status" gorm:"column:rollup_status;default:1"`
	CommitTxHash   string     `json:"commit_tx_hash" gorm:"column:commit_tx_hash;default:NULL"`
	CommittedAt    *time.Time `json:"committed_at" gorm:"column:committed_at;default:NULL"`
	FinalizeTxHash string     `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash;default:NULL"`
	FinalizedAt    *time.Time `json:"finalized_at" gorm:"column:finalized_at;default:NULL"`

	// gas oracle
	OracleStatus int16  `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash string `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash;default:NULL"`

	// blob
	BlobDataProof []byte `json:"blob_data_proof" gorm:"column:blob_data_proof"`
	BlobSize      uint64 `json:"blob_size" gorm:"column:blob_size"`

	// bundle
	BundleHash string `json:"bundle_hash" gorm:"column:bundle_hash"`

	// metadata
	TotalL1CommitGas          uint64         `json:"total_l1_commit_gas" gorm:"column:total_l1_commit_gas;default:0"`
	TotalL1CommitCalldataSize uint64         `json:"total_l1_commit_calldata_size" gorm:"column:total_l1_commit_calldata_size;default:0"`
	CreatedAt                 time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt                 time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt                 gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
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
	db = db.Model(&Batch{})

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
		return nil, fmt.Errorf("Batch.GetBatches error: %w, fields: %v, orderByList: %v", err, fields, orderByList)
	}
	return batches, nil
}

// GetBatchCount retrieves the total number of batches in the database.
func (o *Batch) GetBatchCount(ctx context.Context) (uint64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("Batch.GetBatchCount error: %w", err)
	}
	return uint64(count), nil
}

// GetVerifiedProofByHash retrieves the verified aggregate proof for a batch with the given hash.
func (o *Batch) GetVerifiedProofByHash(ctx context.Context, hash string) (*message.BatchProof, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Select("proof")
	db = db.Where("hash = ? AND proving_status = ?", hash, types.ProvingTaskVerified)

	var batch Batch
	if err := db.Find(&batch).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetVerifiedProofByHash error: %w, batch hash: %v", err, hash)
	}

	var proof message.BatchProof
	if err := json.Unmarshal(batch.Proof, &proof); err != nil {
		return nil, fmt.Errorf("Batch.GetVerifiedProofByHash error: %w, batch hash: %v", err, hash)
	}
	return &proof, nil
}

// GetLatestBatch retrieves the latest batch from the database.
func (o *Batch) GetLatestBatch(ctx context.Context) (*Batch, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Order("index desc")

	var latestBatch Batch
	if err := db.First(&latestBatch).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetLatestBatch error: %w", err)
	}
	return &latestBatch, nil
}

// GetFirstUnbatchedChunkIndex retrieves the first unbatched chunk index.
func (o *Batch) GetFirstUnbatchedChunkIndex(ctx context.Context) (uint64, error) {
	// Get the latest batch
	latestBatch, err := o.GetLatestBatch(ctx)
	if err != nil {
		return 0, fmt.Errorf("Batch.GetFirstUnbatchedChunkIndex error: %w", err)
	}
	return latestBatch.EndChunkIndex + 1, nil
}

// GetBatchesGEIndexGECodecVersion retrieves batches that have a batch index greater than or equal to the given index and codec version.
// The returned batches are sorted in ascending order by their index.
func (o *Batch) GetBatchesGEIndexGECodecVersion(ctx context.Context, index uint64, codecv encoding.CodecVersion, limit int) ([]*Batch, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("index >= ?", index)
	db = db.Where("codec_version >= ?", codecv)
	db = db.Order("index ASC")

	if limit > 0 {
		db = db.Limit(limit)
	}

	var batches []*Batch
	if err := db.Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetBatchesGEIndexGECodecVersion error: %w", err)
	}
	return batches, nil
}

// GetRollupStatusByHashList retrieves the rollup statuses for a list of batch hashes.
func (o *Batch) GetRollupStatusByHashList(ctx context.Context, hashes []string) ([]types.RollupStatus, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Select("hash, rollup_status")
	db = db.Where("hash IN ?", hashes)

	var batches []Batch
	if err := db.Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetRollupStatusByHashList error: %w, hashes: %v", err, hashes)
	}

	hashToStatusMap := make(map[string]types.RollupStatus)
	for _, batch := range batches {
		hashToStatusMap[batch.Hash] = types.RollupStatus(batch.RollupStatus)
	}

	var statuses []types.RollupStatus
	for _, hash := range hashes {
		status, ok := hashToStatusMap[hash]
		if !ok {
			return nil, fmt.Errorf("Batch.GetRollupStatusByHashList: hash not found in database: %s", hash)
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetFailedAndPendingBatches retrieves batches with failed or pending status up to the specified limit.
// The returned batches are sorted in ascending order by their index.
func (o *Batch) GetFailedAndPendingBatches(ctx context.Context, limit int) ([]*Batch, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be greater than zero")
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("rollup_status = ? OR rollup_status = ?", types.RollupCommitFailed, types.RollupPending)
	db = db.Order("index ASC")
	db = db.Limit(limit)

	var batches []*Batch
	if err := db.Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetFailedAndPendingBatches error: %w", err)
	}
	return batches, nil
}

// GetBatchByIndex retrieves the batch by the given index.
func (o *Batch) GetBatchByIndex(ctx context.Context, index uint64) (*Batch, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("index = ?", index)

	var batch Batch
	if err := db.First(&batch).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetBatchByIndex error: %w, index: %v", err, index)
	}
	return &batch, nil
}

// InsertBatch inserts a new batch into the database.
func (o *Batch) InsertBatch(ctx context.Context, batch *encoding.Batch, codecVersion encoding.CodecVersion, metrics rutils.BatchMetrics, dbTX ...*gorm.DB) (*Batch, error) {
	if batch == nil {
		return nil, errors.New("invalid args: batch is nil")
	}

	numChunks := uint64(len(batch.Chunks))
	if numChunks == 0 {
		return nil, errors.New("invalid args: batch contains 0 chunk")
	}

	var startChunkIndex uint64
	if batch.Index > 0 {
		parentBatch, getErr := o.GetBatchByIndex(ctx, batch.Index-1)
		if getErr != nil {
			log.Error("failed to get batch by index", "index", batch.Index, "total l1 message popped before", batch.TotalL1MessagePoppedBefore,
				"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", getErr)
			return nil, fmt.Errorf("Batch.InsertBatch error: %w", getErr)
		}
		startChunkIndex = parentBatch.EndChunkIndex + 1
	}

	batchMeta, err := rutils.GetBatchMetadata(batch, codecVersion)
	if err != nil {
		log.Error("failed to get batch metadata", "index", batch.Index, "total l1 message popped before", batch.TotalL1MessagePoppedBefore,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	enableCompress, err := encoding.GetBatchEnableCompression(codecVersion, batch)
	if err != nil {
		log.Error("failed to get batch enable compress", "index", batch.Index, "total l1 message popped before", batch.TotalL1MessagePoppedBefore,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	newBatch := Batch{
		Index:                     batch.Index,
		Hash:                      batchMeta.BatchHash.Hex(),
		DataHash:                  batchMeta.BatchDataHash.Hex(),
		StartChunkHash:            batchMeta.StartChunkHash.Hex(),
		StartChunkIndex:           startChunkIndex,
		EndChunkHash:              batchMeta.EndChunkHash.Hex(),
		EndChunkIndex:             startChunkIndex + numChunks - 1,
		StateRoot:                 batch.StateRoot().Hex(),
		WithdrawRoot:              batch.WithdrawRoot().Hex(),
		ParentBatchHash:           batch.ParentBatchHash.Hex(),
		BatchHeader:               batchMeta.BatchBytes,
		CodecVersion:              int16(codecVersion),
		EnableCompress:            enableCompress,
		BlobBytes:                 batchMeta.BlobBytes,
		ChunkProofsStatus:         int16(types.ChunkProofsStatusPending),
		ProvingStatus:             int16(types.ProvingTaskUnassigned),
		RollupStatus:              int16(types.RollupPending),
		OracleStatus:              int16(types.GasOraclePending),
		TotalL1CommitGas:          metrics.L1CommitGas,
		TotalL1CommitCalldataSize: metrics.L1CommitCalldataSize,
		BlobDataProof:             batchMeta.BatchBlobDataProof,
		BlobSize:                  metrics.L1CommitBlobSize,
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db.WithContext(ctx)
	db = db.Model(&Batch{})

	if err := db.Create(&newBatch).Error; err != nil {
		log.Error("failed to insert batch", "batch", newBatch, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}
	return &newBatch, nil
}

// UpdateL2GasOracleStatusAndOracleTxHash updates the L2 gas oracle status and transaction hash for a batch.
func (o *Batch) UpdateL2GasOracleStatusAndOracleTxHash(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
	updateFields := make(map[string]interface{})
	updateFields["oracle_status"] = int(status)
	updateFields["oracle_tx_hash"] = txHash

	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateL2GasOracleStatusAndOracleTxHash error: %w, batch hash: %v, status: %v, txHash: %v", err, hash, status.String(), txHash)
	}
	return nil
}

// UpdateProvingStatus updates the proving status of a batch.
func (o *Batch) UpdateProvingStatus(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = int(status)

	switch status {
	case types.ProvingTaskAssigned:
		updateFields["prover_assigned_at"] = utils.NowUTC()
	case types.ProvingTaskUnassigned:
		updateFields["prover_assigned_at"] = nil
	case types.ProvingTaskVerified:
		updateFields["proved_at"] = utils.NowUTC()
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateProvingStatus error: %w, batch hash: %v, status: %v", err, hash, status.String())
	}
	return nil
}

// UpdateRollupStatus updates the rollup status of a batch.
func (o *Batch) UpdateRollupStatus(ctx context.Context, hash string, status types.RollupStatus, dbTX ...*gorm.DB) error {
	updateFields := make(map[string]interface{})
	updateFields["rollup_status"] = int(status)

	switch status {
	case types.RollupCommitted:
		updateFields["committed_at"] = utils.NowUTC()
	case types.RollupFinalized:
		updateFields["finalized_at"] = utils.NowUTC()
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateRollupStatus error: %w, batch hash: %v, status: %v", err, hash, status.String())
	}
	return nil
}

// UpdateCommitTxHashAndRollupStatus updates the commit transaction hash and rollup status for a batch.
func (o *Batch) UpdateCommitTxHashAndRollupStatus(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["commit_tx_hash"] = commitTxHash
	updateFields["rollup_status"] = int(status)
	if status == types.RollupCommitted {
		updateFields["committed_at"] = utils.NowUTC()
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateCommitTxHashAndRollupStatus error: %w, batch hash: %v, status: %v, commitTxHash: %v", err, hash, status.String(), commitTxHash)
	}
	return nil
}

// UpdateFinalizeTxHashAndRollupStatus updates the finalize transaction hash and rollup status for a batch.
func (o *Batch) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["finalize_tx_hash"] = finalizeTxHash
	updateFields["rollup_status"] = int(status)
	if status == types.RollupFinalized {
		updateFields["finalized_at"] = utils.NowUTC()
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateFinalizeTxHashAndRollupStatus error: %w, batch hash: %v, status: %v, finalizeTxHash: %v", err, hash, status.String(), finalizeTxHash)
	}
	return nil
}

// UpdateProofByHash updates the batch proof by hash.
// for unit test.
func (o *Batch) UpdateProofByHash(ctx context.Context, hash string, proof *message.BatchProof, proofTimeSec uint64) error {
	proofBytes, err := json.Marshal(proof)
	if err != nil {
		return fmt.Errorf("Batch.UpdateProofByHash error: %w, batch hash: %v", err, hash)
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proofBytes
	updateFields["proof_time_sec"] = proofTimeSec

	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)

	if err = db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateProofByHash error: %w, batch hash: %v", err, hash)
	}
	return nil
}

// UpdateBundleHashInRange updates the bundle_hash for bundles within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end indices.
func (o *Batch) UpdateBundleHashInRange(ctx context.Context, startIndex uint64, endIndex uint64, bundleHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("index >= ? AND index <= ?", startIndex, endIndex)

	if err := db.Update("bundle_hash", bundleHash).Error; err != nil {
		return fmt.Errorf("Batch.UpdateBundleHashInRange error: %w, start index: %v, end index: %v, batch hash: %v", err, startIndex, endIndex, bundleHash)
	}
	return nil
}

// UpdateProvingStatusByBundleHash updates the proving_status for batches within the specified bundle_hash
func (o *Batch) UpdateProvingStatusByBundleHash(ctx context.Context, bundleHash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = int(status)

	switch status {
	case types.ProvingTaskAssigned:
		updateFields["prover_assigned_at"] = utils.NowUTC()
	case types.ProvingTaskUnassigned:
		updateFields["prover_assigned_at"] = nil
	case types.ProvingTaskVerified:
		updateFields["proved_at"] = utils.NowUTC()
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("bundle_hash = ?", bundleHash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateProvingStatusByBundleHash error: %w, bundle hash: %v, status: %v", err, bundleHash, status.String())
	}
	return nil
}

// UpdateFinalizeTxHashAndRollupStatusByBundleHash updates the finalize transaction hash and rollup status for batches with the specified bundle_hash
func (o *Batch) UpdateFinalizeTxHashAndRollupStatusByBundleHash(ctx context.Context, bundleHash string, finalizeTxHash string, status types.RollupStatus, dbTX ...*gorm.DB) error {
	updateFields := make(map[string]interface{})
	updateFields["finalize_tx_hash"] = finalizeTxHash
	updateFields["rollup_status"] = int(status)

	switch status {
	case types.RollupFinalized:
		updateFields["finalized_at"] = utils.NowUTC()
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("bundle_hash = ?", bundleHash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateFinalizeTxHashAndRollupStatusByBundleHash error: %w, bundle hash: %v, status: %v", err, bundleHash, status.String())
	}
	return nil
}
