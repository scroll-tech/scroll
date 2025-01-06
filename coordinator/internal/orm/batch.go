package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"
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
	EnableCompress  bool   `json:"enable_compress" gorm:"column:enable_compress"`
	BlobBytes       []byte `json:"blob_bytes" gorm:"column:blob_bytes"`

	// proof
	ChunkProofsStatus int16      `json:"chunk_proofs_status" gorm:"column:chunk_proofs_status;default:1"`
	ProvingStatus     int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof             []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt  *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt          *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec      int32      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`
	TotalAttempts     int16      `json:"total_attempts" gorm:"column:total_attempts;default:0"`
	ActiveAttempts    int16      `json:"active_attempts" gorm:"column:active_attempts;default:0"`

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

// GetUnassignedBatch retrieves unassigned batch based on the specified limit.
// The returned batches are sorted in ascending order by their index.
func (o *Batch) GetUnassignedBatch(ctx context.Context, maxActiveAttempts, maxTotalAttempts uint8) (*Batch, error) {
	var batch Batch
	db := o.db.WithContext(ctx)
	sql := fmt.Sprintf("SELECT * FROM batch WHERE proving_status = %d AND total_attempts < %d AND active_attempts < %d AND chunk_proofs_status = %d AND batch.deleted_at IS NULL ORDER BY batch.index LIMIT 1;",
		int(types.ProvingTaskUnassigned), maxTotalAttempts, maxActiveAttempts, int(types.ChunkProofsStatusReady))
	err := db.Raw(sql).Scan(&batch).Error
	if err != nil {
		return nil, fmt.Errorf("Batch.GetUnassignedBatch error: %w", err)
	}
	if batch.Hash == "" {
		return nil, nil
	}
	return &batch, nil
}

// GetAssignedBatch retrieves assigned batch based on the specified limit.
// The returned batches are sorted in ascending order by their index.
func (o *Batch) GetAssignedBatch(ctx context.Context, maxActiveAttempts, maxTotalAttempts uint8) (*Batch, error) {
	var batch Batch
	db := o.db.WithContext(ctx)
	sql := fmt.Sprintf("SELECT * FROM batch WHERE proving_status = %d AND total_attempts < %d AND active_attempts < %d AND chunk_proofs_status = %d AND batch.deleted_at IS NULL ORDER BY batch.index LIMIT 1;",
		int(types.ProvingTaskAssigned), maxTotalAttempts, maxActiveAttempts, int(types.ChunkProofsStatusReady))
	err := db.Raw(sql).Scan(&batch).Error
	if err != nil {
		return nil, fmt.Errorf("Batch.GetAssignedBatch error: %w", err)
	}
	if batch.Hash == "" {
		return nil, nil
	}
	return &batch, nil
}

// GetUnassignedAndChunksUnreadyBatches get the batches which is unassigned and chunks is not ready
func (o *Batch) GetUnassignedAndChunksUnreadyBatches(ctx context.Context, offset, limit int) ([]*Batch, error) {
	if offset < 0 || limit < 0 {
		return nil, errors.New("limit and offset must not be smaller than 0")
	}

	db := o.db.WithContext(ctx)
	db = db.Where("proving_status = ?", types.ProvingTaskUnassigned)
	db = db.Where("chunk_proofs_status = ?", types.ChunkProofsStatusPending)
	db = db.Order("index ASC")
	db = db.Offset(offset)
	db = db.Limit(limit)

	var batches []*Batch
	if err := db.Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetUnassignedAndChunksUnreadyBatches error: %w", err)
	}
	return batches, nil
}

// GetAssignedBatches retrieves all batches whose proving_status is either types.ProvingTaskAssigned.
func (o *Batch) GetAssignedBatches(ctx context.Context) ([]*Batch, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("proving_status = ?", int(types.ProvingTaskAssigned))

	var assignedBatches []*Batch
	if err := db.Find(&assignedBatches).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetAssignedBatches error: %w", err)
	}
	return assignedBatches, nil
}

// GetProvingStatusByHash retrieves the proving status of a batch given its hash.
func (o *Batch) GetProvingStatusByHash(ctx context.Context, hash string) (types.ProvingStatus, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Select("proving_status")
	db = db.Where("hash = ?", hash)

	var batch Batch
	if err := db.Find(&batch).Error; err != nil {
		return types.ProvingStatusUndefined, fmt.Errorf("Batch.GetProvingStatusByHash error: %w, batch hash: %v", err, hash)
	}
	return types.ProvingStatus(batch.ProvingStatus), nil
}

// GetLatestBatch retrieves the latest batch from the database.
func (o *Batch) GetLatestBatch(ctx context.Context) (*Batch, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Order("index desc")

	var latestBatch Batch
	if err := db.First(&latestBatch).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("Batch.GetLatestBatch error: %w", err)
	}
	return &latestBatch, nil
}

// GetAttemptsByHash get batch attempts by hash. Used by unit test
func (o *Batch) GetAttemptsByHash(ctx context.Context, hash string) (int16, int16, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash = ?", hash)
	var batch Batch
	if err := db.Find(&batch).Error; err != nil {
		return 0, 0, fmt.Errorf("Batch.GetAttemptsByHash error: %w, batch hash: %v", err, hash)
	}
	return batch.ActiveAttempts, batch.TotalAttempts, nil
}

// CheckIfBundleBatchProofsAreReady checks if all proofs for all batches of a given bundleHash are collected.
func (o *Batch) CheckIfBundleBatchProofsAreReady(ctx context.Context, bundleHash string) (bool, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("bundle_hash = ? AND proving_status != ?", bundleHash, types.ProvingTaskVerified)

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, fmt.Errorf("Chunk.CheckIfBundleBatchProofsAreReady error: %w, bundle hash: %v", err, bundleHash)
	}
	return count == 0, nil
}

// GetBatchByHash retrieves the given batch.
func (o *Batch) GetBatchByHash(ctx context.Context, hash string) (*Batch, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash = ?", hash)

	var batch Batch
	if err := db.First(&batch).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetBatchByHash error: %w, batch hash: %v", err, hash)
	}
	return &batch, nil
}

// GetBatchesByBundleHash retrieves the given batch.
func (o *Batch) GetBatchesByBundleHash(ctx context.Context, bundleHash string) ([]*Batch, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("bundle_hash = ?", bundleHash)
	db = db.Order("index ASC")

	var batches []*Batch
	if err := db.Find(&batches).Error; err != nil {
		return nil, fmt.Errorf("Batch.GetBatchesByBundleHash error: %w, bundle hash: %v", err, bundleHash)
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
func (o *Batch) InsertBatch(ctx context.Context, batch *encoding.Batch, dbTX ...*gorm.DB) (*Batch, error) {
	if batch == nil {
		return nil, errors.New("invalid args: batch is nil")
	}

	numChunks := uint64(len(batch.Chunks))
	if numChunks == 0 {
		return nil, errors.New("invalid args: batch contains 0 chunk")
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecV0)
	if err != nil {
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	daBatch, err := codec.NewDABatch(batch)
	if err != nil {
		log.Error("failed to create new DA batch",
			"index", batch.Index, "total l1 message popped before", batch.TotalL1MessagePoppedBefore,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, err
	}

	var startChunkIndex uint64
	parentBatch, err := o.GetLatestBatch(ctx)
	if err != nil {
		log.Error("failed to get latest batch", "index", batch.Index, "total l1 message popped before", batch.TotalL1MessagePoppedBefore,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	// if parentBatch==nil then err==gorm.ErrRecordNotFound, which means there's
	// no batch record in the db, we then use default empty values for the creating batch;
	// if parentBatch!=nil then err==nil, then we fill the parentBatch-related data into the creating batch
	if parentBatch != nil {
		startChunkIndex = parentBatch.EndChunkIndex + 1
	}

	startDAChunk, err := codec.NewDAChunk(batch.Chunks[0], batch.TotalL1MessagePoppedBefore)
	if err != nil {
		log.Error("failed to create start DA chunk", "index", batch.Index, "total l1 message popped before", batch.TotalL1MessagePoppedBefore,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	startDAChunkHash, err := startDAChunk.Hash()
	if err != nil {
		log.Error("failed to get start DA chunk hash", "index", batch.Index, "total l1 message popped before", batch.TotalL1MessagePoppedBefore,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	totalL1MessagePoppedBeforeEndDAChunk := batch.TotalL1MessagePoppedBefore
	for i := uint64(0); i < numChunks-1; i++ {
		totalL1MessagePoppedBeforeEndDAChunk += batch.Chunks[i].NumL1Messages(totalL1MessagePoppedBeforeEndDAChunk)
	}
	endDAChunk, err := codec.NewDAChunk(batch.Chunks[numChunks-1], totalL1MessagePoppedBeforeEndDAChunk)
	if err != nil {
		log.Error("failed to create end DA chunk", "index", batch.Index, "total l1 message popped before", totalL1MessagePoppedBeforeEndDAChunk,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	endDAChunkHash, err := endDAChunk.Hash()
	if err != nil {
		log.Error("failed to get end DA chunk hash", "index", batch.Index, "total l1 message popped before", totalL1MessagePoppedBeforeEndDAChunk,
			"parent hash", batch.ParentBatchHash.Hex(), "number of chunks", numChunks, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}

	newBatch := Batch{
		Index:             batch.Index,
		Hash:              daBatch.Hash().Hex(),
		DataHash:          daBatch.DataHash().Hex(),
		StartChunkHash:    startDAChunkHash.Hex(),
		StartChunkIndex:   startChunkIndex,
		EndChunkHash:      endDAChunkHash.Hex(),
		EndChunkIndex:     startChunkIndex + numChunks - 1,
		StateRoot:         batch.StateRoot().Hex(),
		WithdrawRoot:      batch.WithdrawRoot().Hex(),
		ParentBatchHash:   batch.ParentBatchHash.Hex(),
		BatchHeader:       daBatch.Encode(),
		ChunkProofsStatus: int16(types.ChunkProofsStatusPending),
		ProvingStatus:     int16(types.ProvingTaskUnassigned),
		TotalAttempts:     0,
		ActiveAttempts:    0,
		RollupStatus:      int16(types.RollupPending),
		OracleStatus:      int16(types.GasOraclePending),
		BlobDataProof:     nil, // using mock value because this piece of codes is only used in unit tests
		BlobSize:          0,   // using mock value because this piece of codes is only used in unit tests
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

// UpdateChunkProofsStatusByBatchHash updates the status of chunk_proofs_status field for a given batch hash.
// The function will set the chunk_proofs_status to the status provided.
func (o *Batch) UpdateChunkProofsStatusByBatchHash(ctx context.Context, batchHash string, status types.ChunkProofsStatus) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash = ?", batchHash)

	if err := db.Update("chunk_proofs_status", status).Error; err != nil {
		return fmt.Errorf("Batch.UpdateChunkProofsStatusByBatchHash error: %w, batch hash: %v, status: %v", err, batchHash, status.String())
	}
	return nil
}

// UpdateProvingStatusFailed updates the proving status failed of a batch.
func (o *Batch) UpdateProvingStatusFailed(ctx context.Context, hash string, maxAttempts uint8, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)
	db = db.Where("total_attempts >= ?", maxAttempts)
	db = db.Where("proving_status != ?", int(types.ProvingTaskVerified))
	if err := db.Update("proving_status", int(types.ProvingTaskFailed)).Error; err != nil {
		return fmt.Errorf("Batch.UpdateProvingStatus error: %w, batch hash: %v, status: %v", err, hash, types.ProvingTaskFailed.String())
	}
	return nil
}

// UpdateProofAndProvingStatusByHash updates the batch proof and proving status by hash.
func (o *Batch) UpdateProofAndProvingStatusByHash(ctx context.Context, hash string, proof []byte, provingStatus types.ProvingStatus, proofTimeSec uint64, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proof
	updateFields["proving_status"] = provingStatus
	updateFields["proof_time_sec"] = proofTimeSec
	updateFields["proved_at"] = utils.NowUTC()

	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateProofByHash error: %w, batch hash: %v", err, hash)
	}
	return nil
}

// UpdateBatchAttempts atomically increments the attempts count for the earliest available batch that meets the conditions.
func (o *Batch) UpdateBatchAttempts(ctx context.Context, index uint64, curActiveAttempts, curTotalAttempts int16) (int64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("index = ?", index)
	db = db.Where("active_attempts = ?", curActiveAttempts)
	db = db.Where("total_attempts = ?", curTotalAttempts)
	result := db.Updates(map[string]interface{}{
		"proving_status":  types.ProvingTaskAssigned,
		"total_attempts":  gorm.Expr("total_attempts + 1"),
		"active_attempts": gorm.Expr("active_attempts + 1"),
	})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to update batch, err:%w", result.Error)
	}
	return result.RowsAffected, nil
}

// DecreaseActiveAttemptsByHash decrements the active_attempts of a batch given its hash.
func (o *Batch) DecreaseActiveAttemptsByHash(ctx context.Context, batchHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})
	db = db.Where("hash = ?", batchHash)
	db = db.Where("proving_status != ?", int(types.ProvingTaskVerified))
	db = db.Where("active_attempts > ?", 0)
	result := db.UpdateColumn("active_attempts", gorm.Expr("active_attempts - 1"))
	if result.Error != nil {
		return fmt.Errorf("Chunk.DecreaseActiveAttemptsByHash error: %w, batch hash: %v", result.Error, batchHash)
	}
	if result.RowsAffected == 0 {
		log.Warn("No rows were affected in DecreaseActiveAttemptsByHash", "batch hash", batchHash)
	}
	return nil
}
