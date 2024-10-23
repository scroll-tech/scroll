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

	codec, err := encoding.CodecFromVersion(encoding.CodecV0)
	if err != nil {
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	daChunk, err := codec.NewDAChunk(chunk, totalL1MessagePoppedBefore)
	if err != nil {
		log.Error("failed to initialize new DA chunk", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	daChunkHash, err := daChunk.Hash()
	if err != nil {
		log.Error("failed to get DA chunk hash", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	totalL1CommitCalldataSize, err := codec.EstimateChunkL1CommitCalldataSize(chunk)
	if err != nil {
		log.Error("failed to estimate chunk L1 commit calldata size", "err", err)
		return nil, fmt.Errorf("Chunk.InsertChunk error: %w", err)
	}

	totalL1CommitGas, err := codec.EstimateChunkL1CommitGas(chunk)
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
		TotalL2TxGas:                 chunk.TotalGasUsed(),
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
