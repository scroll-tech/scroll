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

// InsertBatch inserts a new batch into the database.
// for unit test
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
		RollupStatus:      int16(types.RollupPending),
		OracleStatus:      int16(types.GasOraclePending),
		BlobDataProof:     nil, // using mock value because this piece of codes is only used in unit tests
		BlobSize:          0,   // using mock value because this piece of codes is only used in unit tests
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Batch{})

	if err := db.Create(&newBatch).Error; err != nil {
		log.Error("failed to insert batch", "batch", newBatch, "err", err)
		return nil, fmt.Errorf("Batch.InsertBatch error: %w", err)
	}
	return &newBatch, nil
}
