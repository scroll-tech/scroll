package orm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
)

// Chunk represents a chunk of blocks in the database.
type Chunk struct {
	db *gorm.DB `gorm:"-"`

	// chunk
	Index                        uint64 `json:"index" gorm:"column:index"`
	Hash                         string `json:"hash" gorm:"column:hash"`
	StartBlockNumber             uint64 `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash               string `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber               uint64 `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash                 string `json:"end_block_hash" gorm:"column:end_block_hash"`
	StartBlockTime               uint64 `json:"start_block_time" gorm:"column:start_block_time"`
	TotalL1MessagesPoppedBefore  uint64 `json:"total_l1_messages_popped_before" gorm:"column:total_l1_messages_popped_before"`
	TotalL1MessagesPoppedInChunk uint64 `json:"total_l1_messages_popped_in_chunk" gorm:"column:total_l1_messages_popped_in_chunk"`
	ParentChunkHash              string `json:"parent_chunk_hash" gorm:"column:parent_chunk_hash"`
	StateRoot                    string `json:"state_root" gorm:"column:state_root"`
	ParentChunkStateRoot         string `json:"parent_chunk_state_root" gorm:"column:parent_chunk_state_root"`
	WithdrawRoot                 string `json:"withdraw_root" gorm:"column:withdraw_root"`

	// proof
	ProvingStatus    int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof            []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec     int32      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`
	TotalAttempts    int16      `json:"total_attempts" gorm:"column:total_attempts;default:0"`
	ActiveAttempts   int16      `json:"active_attempts" gorm:"column:active_attempts;default:0"`

	// batch
	BatchHash string `json:"batch_hash" gorm:"column:batch_hash;default:NULL"`

	// blob
	CrcMax   uint64 `json:"crc_max" gorm:"column:crc_max"`
	BlobSize uint64 `json:"blob_size" gorm:"column:blob_size"`

	// metadata
	TotalL2TxGas              uint64         `json:"total_l2_tx_gas" gorm:"column:total_l2_tx_gas"`
	TotalL2TxNum              uint64         `json:"total_l2_tx_num" gorm:"column:total_l2_tx_num"`
	TotalL1CommitCalldataSize uint64         `json:"total_l1_commit_calldata_size" gorm:"column:total_l1_commit_calldata_size"`
	TotalL1CommitGas          uint64         `json:"total_l1_commit_gas" gorm:"column:total_l1_commit_gas"`
	CreatedAt                 time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt                 time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt                 gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewChunk creates a new Chunk database instance.
func NewChunk(db *gorm.DB) *Chunk {
	return &Chunk{db: db}
}

// TableName returns the table name for the chunk model.
func (*Chunk) TableName() string {
	return "chunk"
}

// GetUnassignedChunk retrieves unassigned chunk based on the specified limit.
// The returned chunks are sorted in ascending order by their index.
func (o *Chunk) GetUnassignedChunk(ctx context.Context, maxActiveAttempts, maxTotalAttempts uint8, height uint64) (*Chunk, error) {
	var chunk Chunk
	db := o.db.WithContext(ctx)
	sql := fmt.Sprintf("SELECT * FROM chunk WHERE proving_status = %d AND total_attempts < %d AND active_attempts < %d AND end_block_number <= %d AND chunk.deleted_at IS NULL ORDER BY chunk.index LIMIT 1;",
		int(types.ProvingTaskUnassigned), maxTotalAttempts, maxActiveAttempts, height)
	err := db.Raw(sql).Scan(&chunk).Error
	if err != nil {
		return nil, fmt.Errorf("Chunk.GetUnassignedChunk error: %w", err)
	}
	if chunk.Hash == "" {
		return nil, nil
	}
	return &chunk, nil
}

// GetAssignedChunk retrieves assigned chunk based on the specified limit.
// The returned chunks are sorted in ascending order by their index.
func (o *Chunk) GetAssignedChunk(ctx context.Context, maxActiveAttempts, maxTotalAttempts uint8, height uint64) (*Chunk, error) {
	var chunk Chunk
	db := o.db.WithContext(ctx)
	sql := fmt.Sprintf("SELECT * FROM chunk WHERE proving_status = %d AND total_attempts < %d AND active_attempts < %d AND end_block_number <= %d AND chunk.deleted_at IS NULL ORDER BY chunk.index LIMIT 1;",
		int(types.ProvingTaskAssigned), maxTotalAttempts, maxActiveAttempts, height)
	err := db.Raw(sql).Scan(&chunk).Error
	if err != nil {
		return nil, fmt.Errorf("Chunk.GetAssignedChunk error: %w", err)
	}
	if chunk.Hash == "" {
		return nil, nil
	}
	return &chunk, nil
}

// GetChunksByBatchHash retrieves the chunks associated with a specific batch hash.
// The returned chunks are sorted in ascending order by their associated chunk index.
func (o *Chunk) GetChunksByBatchHash(ctx context.Context, batchHash string) ([]*Chunk, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("batch_hash", batchHash)
	db = db.Order("index ASC")

	var chunks []*Chunk
	if err := db.Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("Chunk.GetChunksByBatchHash error: %w, batch hash: %v", err, batchHash)
	}
	return chunks, nil
}

// GetProofsByBatchHash retrieves the proofs associated with a specific batch hash.
// It returns a slice of decoded proofs (message.ChunkProof) obtained from the database.
// The returned proofs are sorted in ascending order by their associated chunk index.
func (o *Chunk) GetProofsByBatchHash(ctx context.Context, batchHash string) ([]*message.ChunkProof, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("batch_hash", batchHash)
	db = db.Order("index ASC")

	var chunks []*Chunk
	if err := db.Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("Chunk.GetProofsByBatchHash error: %w, batch hash: %v", err, batchHash)
	}

	var proofs []*message.ChunkProof
	for _, chunk := range chunks {
		var proof message.ChunkProof
		if err := json.Unmarshal(chunk.Proof, &proof); err != nil {
			return nil, fmt.Errorf("Chunk.GetProofsByBatchHash unmarshal proof error: %w, batch hash: %v, chunk hash: %v", err, batchHash, chunk.Hash)
		}
		proofs = append(proofs, &proof)
	}

	return proofs, nil
}

// getLatestChunk retrieves the latest chunk from the database.
func (o *Chunk) getLatestChunk(ctx context.Context) (*Chunk, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Order("index desc")

	var latestChunk Chunk
	if err := db.First(&latestChunk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("Chunk.getLatestChunk error: %w", err)
	}
	return &latestChunk, nil
}

// GetProvingStatusByHash retrieves the proving status of a chunk given its hash.
func (o *Chunk) GetProvingStatusByHash(ctx context.Context, hash string) (types.ProvingStatus, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Select("proving_status")
	db = db.Where("hash = ?", hash)

	var chunk Chunk
	if err := db.Find(&chunk).Error; err != nil {
		return types.ProvingStatusUndefined, fmt.Errorf("Chunk.GetProvingStatusByHash error: %w, chunk hash: %v", err, hash)
	}
	return types.ProvingStatus(chunk.ProvingStatus), nil
}

// CheckIfBatchChunkProofsAreReady checks if all proofs for all chunks of a given batchHash are collected.
func (o *Chunk) CheckIfBatchChunkProofsAreReady(ctx context.Context, batchHash string) (bool, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("batch_hash = ? AND proving_status != ?", batchHash, types.ProvingTaskVerified)

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, fmt.Errorf("Chunk.CheckIfBatchChunkProofsAreReady error: %w, batch hash: %v", err, batchHash)
	}
	return count == 0, nil
}

// GetChunkByHash retrieves the given chunk.
func (o *Chunk) GetChunkByHash(ctx context.Context, chunkHash string) (*Chunk, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("hash = ?", chunkHash)

	var chunk Chunk
	if err := db.First(&chunk).Error; err != nil {
		return nil, fmt.Errorf("Chunk.GetChunkByHash error: %w, chunk hash: %v", err, chunkHash)
	}
	return &chunk, nil
}

// GetChunkByStartBlockNumber retrieves the given chunk.
func (o *Chunk) GetChunkByStartBlockNumber(ctx context.Context, startBlockNumber uint64) (*Chunk, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("start_block_number = ?", startBlockNumber)

	var chunk Chunk
	if err := db.First(&chunk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("Chunk.GetChunkByStartBlockNumber error: %w, chunk start block number: %v", err, startBlockNumber)
	}
	return &chunk, nil
}

// GetAttemptsByHash get chunk attempts by hash. Used by unit test
func (o *Chunk) GetAttemptsByHash(ctx context.Context, hash string) (int16, int16, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("hash = ?", hash)
	var chunk Chunk
	if err := db.Find(&chunk).Error; err != nil {
		return 0, 0, fmt.Errorf("Batch.GetAttemptsByHash error: %w, batch hash: %v", err, hash)
	}
	return chunk.ActiveAttempts, chunk.TotalAttempts, nil
}

// InsertChunk inserts a new chunk into the database.
// for unit test
func (o *Chunk) InsertChunk(ctx context.Context, chunk *encoding.Chunk, dbTX ...*gorm.DB) (*Chunk, error) {
	if chunk == nil || len(chunk.Blocks) == 0 {
		return nil, errors.New("invalid args")
	}

	var chunkIndex uint64
	var totalL1MessagePoppedBefore uint64
	var parentChunkHash string
	var parentChunkStateRoot string
	parentChunk, err := o.getLatestChunk(ctx)
	if err != nil {
		log.Error("failed to get latest chunk", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	// if parentChunk==nil then err==gorm.ErrRecordNotFound, which means there's
	// no chunk record in the db, we then use default empty values for the creating chunk;
	// if parentChunk!=nil then err==nil, then we fill the parentChunk-related data into the creating chunk
	if parentChunk != nil {
		chunkIndex = parentChunk.Index + 1
		totalL1MessagePoppedBefore = parentChunk.TotalL1MessagesPoppedBefore + parentChunk.TotalL1MessagesPoppedInChunk
		parentChunkHash = parentChunk.Hash
		parentChunkStateRoot = parentChunk.StateRoot
	}

	daChunk, err := codecv0.NewDAChunk(chunk, totalL1MessagePoppedBefore)
	if err != nil {
		log.Error("failed to initialize new DA chunk", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	daChunkHash, err := daChunk.Hash()
	if err != nil {
		log.Error("failed to get DA chunk hash", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	totalL1CommitCalldataSize, err := codecv0.EstimateChunkL1CommitCalldataSize(chunk)
	if err != nil {
		log.Error("failed to estimate chunk L1 commit calldata size", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	totalL1CommitGas, err := codecv0.EstimateChunkL1CommitGas(chunk)
	if err != nil {
		log.Error("failed to estimate chunk L1 commit gas", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	numBlocks := len(chunk.Blocks)
	newChunk := Chunk{
		Index:                        chunkIndex,
		Hash:                         daChunkHash.Hex(),
		StartBlockNumber:             chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:               chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:               chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:                 chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
		TotalL2TxGas:                 chunk.L2GasUsed(),
		TotalL2TxNum:                 chunk.NumL2Transactions(),
		TotalL1CommitCalldataSize:    totalL1CommitCalldataSize,
		TotalL1CommitGas:             totalL1CommitGas,
		StartBlockTime:               chunk.Blocks[0].Header.Time,
		TotalL1MessagesPoppedBefore:  totalL1MessagePoppedBefore,
		TotalL1MessagesPoppedInChunk: chunk.NumL1Messages(totalL1MessagePoppedBefore),
		ParentChunkHash:              parentChunkHash,
		StateRoot:                    chunk.Blocks[numBlocks-1].Header.Root.Hex(),
		ParentChunkStateRoot:         parentChunkStateRoot,
		WithdrawRoot:                 chunk.Blocks[numBlocks-1].WithdrawRoot.Hex(),
		ProvingStatus:                int16(types.ProvingTaskUnassigned),
		TotalAttempts:                0,
		ActiveAttempts:               0,
		CrcMax:                       0, // using mock value because this piece of codes is only used in unit tests
		BlobSize:                     0, // using mock value because this piece of codes is only used in unit tests
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Chunk{})

	if err := db.Create(&newChunk).Error; err != nil {
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w, chunk hash: %v", err, newChunk.Hash)
	}

	return &newChunk, nil
}

// UpdateProvingStatusFailed updates the proving status failed of a batch.
func (o *Chunk) UpdateProvingStatusFailed(ctx context.Context, hash string, maxAttempts uint8, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("hash", hash)
	db = db.Where("total_attempts >= ?", maxAttempts)
	db = db.Where("proving_status != ?", int(types.ProvingTaskVerified))
	if err := db.Update("proving_status", int(types.ProvingTaskFailed)).Error; err != nil {
		return fmt.Errorf("Batch.UpdateProvingStatus error: %w, batch hash: %v, status: %v", err, hash, types.ProvingTaskFailed.String())
	}
	return nil
}

// UpdateProofAndProvingStatusByHash updates the chunk proof and proving_status by hash.
func (o *Chunk) UpdateProofAndProvingStatusByHash(ctx context.Context, hash string, proof []byte, status types.ProvingStatus, proofTimeSec uint64, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proof
	updateFields["proving_status"] = int(status)
	updateFields["proof_time_sec"] = proofTimeSec
	updateFields["proved_at"] = utils.NowUTC()

	db = db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Chunk.UpdateProofByHash error: %w, chunk hash: %v", err, hash)
	}
	return nil
}

// UpdateBatchHashInRange updates the batch_hash for chunks within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end indices.
// for unit test
func (o *Chunk) UpdateBatchHashInRange(ctx context.Context, startIndex uint64, endIndex uint64, batchHash string) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("index >= ? AND index <= ?", startIndex, endIndex)

	if err := db.Update("batch_hash", batchHash).Error; err != nil {
		return fmt.Errorf("Chunk.UpdateBatchHashInRange error: %w, start index: %v, end index: %v, batch hash: %v", err, startIndex, endIndex, batchHash)
	}
	return nil
}

// UpdateChunkAttempts atomically increments the attempts count for the earliest available chunk that meets the conditions.
func (o *Chunk) UpdateChunkAttempts(ctx context.Context, index uint64, curActiveAttempts, curTotalAttempts int16) (int64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("index = ?", index)
	db = db.Where("active_attempts = ?", curActiveAttempts)
	db = db.Where("total_attempts = ?", curTotalAttempts)
	result := db.Updates(map[string]interface{}{
		"proving_status":  types.ProvingTaskAssigned,
		"total_attempts":  gorm.Expr("total_attempts + 1"),
		"active_attempts": gorm.Expr("active_attempts + 1"),
	})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to update chunk, err:%w", result.Error)
	}
	return result.RowsAffected, nil
}

// DecreaseActiveAttemptsByHash decrements the active_attempts of a chunk given its hash.
func (o *Chunk) DecreaseActiveAttemptsByHash(ctx context.Context, chunkHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("hash = ?", chunkHash)
	db = db.Where("proving_status != ?", int(types.ProvingTaskVerified))
	db = db.Where("active_attempts > ?", 0)
	result := db.UpdateColumn("active_attempts", gorm.Expr("active_attempts - 1"))
	if result.Error != nil {
		return fmt.Errorf("Chunk.DecreaseActiveAttemptsByHash error: %w, chunk hash: %v", result.Error, chunkHash)
	}
	if result.RowsAffected == 0 {
		log.Warn("No rows were affected in DecreaseActiveAttemptsByHash", "chunk hash", chunkHash)
	}
	return nil
}
