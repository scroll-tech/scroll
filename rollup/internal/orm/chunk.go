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

	"scroll-tech/rollup/internal/utils"
	rutils "scroll-tech/rollup/internal/utils"
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
	CodecVersion                 int16  `json:"codec_version" gorm:"column:codec_version"`
	EnableCompress               bool   `json:"enable_compress" gorm:"column:enable_compress"` // use for debug

	// proof
	ProvingStatus    int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof            []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec     int32      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`

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

// GetChunksInRange retrieves chunks within a given range (inclusive) from the database.
// The range is closed, i.e., it includes both start and end indices.
// The returned chunks are sorted in ascending order by their index.
func (o *Chunk) GetChunksInRange(ctx context.Context, startIndex uint64, endIndex uint64) ([]*Chunk, error) {
	if startIndex > endIndex {
		return nil, fmt.Errorf("Chunk.GetChunksInRange: start index should be less than or equal to end index, start index: %v, end index: %v", startIndex, endIndex)
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("index >= ? AND index <= ?", startIndex, endIndex)
	db = db.Order("index ASC")

	var chunks []*Chunk
	if err := db.Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("Chunk.GetChunksInRange error: %w, start index: %v, end index: %v", err, startIndex, endIndex)
	}

	// sanity check
	if uint64(len(chunks)) != endIndex-startIndex+1 {
		return nil, fmt.Errorf("Chunk.GetChunksInRange: incorrect number of chunks, expected: %v, got: %v, start index: %v, end index: %v", endIndex-startIndex+1, len(chunks), startIndex, endIndex)
	}

	return chunks, nil
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

// GetUnchunkedBlockHeight retrieves the first unchunked block number.
func (o *Chunk) GetUnchunkedBlockHeight(ctx context.Context) (uint64, error) {
	// Get the latest chunk
	latestChunk, err := o.getLatestChunk(ctx)
	if err != nil {
		return 0, fmt.Errorf("Chunk.GetUnchunkedBlockHeight error: %w", err)
	}
	if latestChunk == nil {
		// if there is no chunk, return block number 1,
		// because no need to chunk genesis block number
		return 1, nil
	}
	return latestChunk.EndBlockNumber + 1, nil
}

// GetChunksGEIndex retrieves chunks that have a chunk index greater than the or equal to the given index.
// The returned chunks are sorted in ascending order by their index.
func (o *Chunk) GetChunksGEIndex(ctx context.Context, index uint64, limit int) ([]*Chunk, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("index >= ?", index)
	db = db.Order("index ASC")

	if limit > 0 {
		db = db.Limit(limit)
	}

	var chunks []*Chunk
	if err := db.Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("Chunk.GetChunksGEIndex error: %w", err)
	}
	return chunks, nil
}

// GetChunkByIndex retrieves a chunk that has the exact chunk index as given.
func (o *Chunk) GetChunkByIndex(ctx context.Context, index uint64) (*Chunk, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("index = ?", index)

	var chunk Chunk
	if err := db.First(&chunk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("Chunk.GetChunkByIndex error: %w", err)
	}
	return &chunk, nil
}

// GetChunksByBatchHash retrieves chunks by batch hash
// for test
func (o *Chunk) GetChunksByBatchHash(ctx context.Context, batchHash string) ([]*Chunk, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("batch_hash = ?", batchHash)

	var chunks []*Chunk
	if err := db.Find(&chunks).Error; err != nil {
		return nil, fmt.Errorf("Chunk.GetChunksByBatchHash error: %w", err)
	}
	return chunks, nil
}

// InsertChunk inserts a new chunk into the database.
func (o *Chunk) InsertChunk(ctx context.Context, chunk *encoding.Chunk, codecConfig rutils.CodecConfig, metrics utils.ChunkMetrics, dbTX ...*gorm.DB) (*Chunk, error) {
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

	chunkHash, err := utils.GetChunkHash(chunk, totalL1MessagePoppedBefore, codecConfig.Version)
	if err != nil {
		log.Error("failed to get chunk hash", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	numBlocks := len(chunk.Blocks)
	newChunk := Chunk{
		Index:                        chunkIndex,
		Hash:                         chunkHash.Hex(),
		StartBlockNumber:             chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:               chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:               chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:                 chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
		TotalL2TxGas:                 chunk.L2GasUsed(),
		TotalL2TxNum:                 chunk.NumL2Transactions(),
		TotalL1CommitCalldataSize:    metrics.L1CommitCalldataSize,
		TotalL1CommitGas:             metrics.L1CommitGas,
		StartBlockTime:               chunk.Blocks[0].Header.Time,
		TotalL1MessagesPoppedBefore:  totalL1MessagePoppedBefore,
		TotalL1MessagesPoppedInChunk: chunk.NumL1Messages(totalL1MessagePoppedBefore),
		ParentChunkHash:              parentChunkHash,
		StateRoot:                    chunk.Blocks[numBlocks-1].Header.Root.Hex(),
		ParentChunkStateRoot:         parentChunkStateRoot,
		WithdrawRoot:                 chunk.Blocks[numBlocks-1].WithdrawRoot.Hex(),
		CodecVersion:                 int16(codecConfig.Version),
		EnableCompress:               codecConfig.EnableCompress,
		ProvingStatus:                int16(types.ProvingTaskUnassigned),
		CrcMax:                       metrics.CrcMax,
		BlobSize:                     metrics.L1CommitBlobSize,
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

// UpdateProvingStatus updates the proving status of a chunk.
func (o *Chunk) UpdateProvingStatus(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = int(status)

	switch status {
	case types.ProvingTaskAssigned:
		updateFields["prover_assigned_at"] = time.Now()
	case types.ProvingTaskUnassigned:
		updateFields["prover_assigned_at"] = nil
	case types.ProvingTaskVerified:
		updateFields["proved_at"] = time.Now()
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Chunk.UpdateProvingStatus error: %w, chunk hash: %v, status: %v", err, hash, status.String())
	}
	return nil
}

// UpdateProvingStatusByBatchHash updates the proving_status for chunks within the specified batch_hash
func (o *Chunk) UpdateProvingStatusByBatchHash(ctx context.Context, batchHash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = int(status)

	switch status {
	case types.ProvingTaskAssigned:
		updateFields["prover_assigned_at"] = time.Now()
	case types.ProvingTaskUnassigned:
		updateFields["prover_assigned_at"] = nil
	case types.ProvingTaskVerified:
		updateFields["proved_at"] = time.Now()
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("batch_hash = ?", batchHash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Chunk.UpdateProvingStatusByBatchHash error: %w, batch hash: %v, status: %v", err, batchHash, status.String())
	}
	return nil
}

// UpdateBatchHashInRange updates the batch_hash for chunks within the specified range (inclusive).
// The range is closed, i.e., it includes both start and end indices.
func (o *Chunk) UpdateBatchHashInRange(ctx context.Context, startIndex uint64, endIndex uint64, batchHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Chunk{})
	db = db.Where("index >= ? AND index <= ?", startIndex, endIndex)

	if err := db.Update("batch_hash", batchHash).Error; err != nil {
		return fmt.Errorf("Chunk.UpdateBatchHashInRange error: %w, start index: %v, end index: %v, batch hash: %v", err, startIndex, endIndex, batchHash)
	}
	return nil
}
