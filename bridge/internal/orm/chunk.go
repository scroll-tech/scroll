package orm

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"scroll-tech/common/types"

	bridgeTypes "scroll-tech/bridge/internal/types"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
)

// Chunk represents a chunk of blocks in the database.
type Chunk struct {
	db *gorm.DB `gorm:"-"`

	// block
	Index                     uint64 `json:"index" gorm:"column:index"`
	Hash                      string `json:"hash" gorm:"column:hash"`
	StartBlockNumber          uint64 `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash            string `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber            uint64 `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash              string `json:"end_block_hash" gorm:"column:end_block_hash"`
	TotalL2TxGas              uint64 `json:"total_l2_tx_gas" gorm:"column:total_l2_tx_gas"`
	TotalL2TxNum              uint64 `json:"total_l2_tx_num" gorm:"column:total_l2_tx_num"`
	TotalL1CommitCalldataSize uint64 `json:"total_l1_commit_calldata_size" gorm:"column:total_l1_commit_calldata_size"`
	TotalL1CommitGas          uint64 `json:"total_l1_commit_gas" gorm:"column:total_l1_commit_gas"`
	StartBlockTime            uint64 `json:"start_block_time" gorm:"column:start_block_time"`

	// chunk
	TotalL1MessagesPoppedBefore uint64 `json:"total_l1_messages_popped_before" gorm:"column:total_l1_messages_popped_before"`
	TotalL1MessagesPopped       uint64 `json:"total_l1_messages_popped" gorm:"column:total_l1_messages_popped"`

	// proof
	ProvingStatus    int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof            []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec     int16      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`

	// batch
	BatchHash string `json:"batch_hash" gorm:"column:batch_hash;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewChunk creates a new Chunk database instance.
func NewChunk(db *gorm.DB) *Chunk {
	return &Chunk{db: db}
}

// TableName returns the table name for the chunk model.
func (*Chunk) TableName() string {
	return "chunk"
}

// GetChunksInClosedRange retrieves chunks within the specified index range from the database.
func (o *Chunk) GetChunksInClosedRange(ctx context.Context, startIndex uint64, endIndex uint64) ([]*Chunk, error) {
	if startIndex > endIndex {
		return nil, errors.New("start index should be less than or equal to end index")
	}

	var chunks []*Chunk
	db := o.db.WithContext(ctx).Where("index >= ? AND index <= ?", startIndex, endIndex)
	db = db.Order("index ASC")

	if err := db.Find(&chunks).Error; err != nil {
		return nil, err
	}

	if startIndex+uint64(len(chunks)) != endIndex+1 {
		return nil, errors.New("number of chunks not expected in the specified range")
	}

	return chunks, nil
}

// GetChunkByStartBlockIndex retrieves a chunk from the database based on the start block number.
func (o *Chunk) GetChunkByStartBlockIndex(ctx context.Context, startBlockNumber uint64) (*Chunk, error) {
	var chunk Chunk
	if err := o.db.Where("start_block_number = ?", startBlockNumber).First(&chunk).Error; err != nil {
		return nil, err
	}
	return &chunk, nil
}

// GetUnbatchedChunks retrieves unbatched chunks from the database.
func (o *Chunk) GetUnbatchedChunks(ctx context.Context) ([]*Chunk, error) {
	var chunks []*Chunk
	err := o.db.WithContext(ctx).
		Where("batch_hash IS NULL").
		Order("start_block_number asc").
		Find(&chunks).Error
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetTotalL1MessagePoppedByEndBlockNumber retrieves the total number of L1 messages popped by the end block number.
func (o *Chunk) GetTotalL1MessagePoppedByEndBlockNumber(ctx context.Context, endBlockNumber uint64) (uint64, error) {
	var chunk Chunk
	if err := o.db.Where("endBlockNumber = ?", endBlockNumber).First(&chunk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return chunk.TotalL1MessagesPoppedBefore + chunk.TotalL1MessagesPopped, nil
}

// GetLatestChunk retrieves the latest chunk from the database.
func (o *Chunk) GetLatestChunk(ctx context.Context) (*Chunk, error) {
	var latestChunk Chunk
	err := o.db.WithContext(ctx).Order("index DESC").First(&latestChunk).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &latestChunk, nil
}

// InsertChunk inserts a new chunk into the database.
func (o *Chunk) InsertChunk(ctx context.Context, chunk *bridgeTypes.Chunk, dbTX ...*gorm.DB) (string, error) {
	if chunk == nil || len(chunk.Blocks) == 0 {
		return "", errors.New("invalid args")
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	var totalL1MessagePoppedBefore uint64
	parentChunk, err := o.GetLatestChunk(ctx)
	if err != nil {
		log.Error("failed to get latest chunk", "err", err)
		return "", err
	}
	if parentChunk != nil {
		totalL1MessagePoppedBefore = parentChunk.TotalL1MessagesPoppedBefore + parentChunk.TotalL1MessagesPopped
	}
	hash, err := chunk.Hash(totalL1MessagePoppedBefore)
	if err != nil {
		log.Error("failed to get chunk hash", "err", err)
		return "", err
	}

	var totalL2TxGas uint64
	var totalL2TxNum uint64
	var totalL1CommitCalldataSize uint64
	var totalL1CommitGas uint64
	for _, block := range chunk.Blocks {
		totalL2TxGas += block.Header.GasUsed
		totalL2TxNum += uint64(len(block.Transactions))
		totalL1CommitCalldataSize += block.ApproximateL1CommitCalldataSize()
		totalL1CommitGas += block.ApproximateL1CommitGas()
	}

	var chunkIndex uint64
	var lastChunk Chunk
	if err := db.Order("index desc").First(&lastChunk).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return "", err
		}
	} else {
		chunkIndex = lastChunk.Index + 1
	}

	numBlocks := len(chunk.Blocks)
	newChunk := Chunk{
		Index:                       chunkIndex,
		Hash:                        hex.EncodeToString(hash),
		StartBlockNumber:            chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:              chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:              chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:                chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
		TotalL2TxGas:                totalL2TxGas,
		TotalL2TxNum:                totalL2TxNum,
		TotalL1CommitCalldataSize:   totalL1CommitCalldataSize,
		TotalL1CommitGas:            totalL1CommitGas,
		StartBlockTime:              chunk.Blocks[0].Header.Time,
		TotalL1MessagesPoppedBefore: totalL1MessagePoppedBefore,
		TotalL1MessagesPopped:       chunk.NumL1Messages(totalL1MessagePoppedBefore),
	}

	if err := db.Create(&newChunk).Error; err != nil {
		log.Error("failed to insert chunk", "hash", hash, "err", err)
		return "", err
	}
	return newChunk.Hash, nil
}

// UpdateProvingStatus updates the proving status of a chunk.
func (o *Chunk) UpdateProvingStatus(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
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

	if err := db.Model(&Chunk{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateBatchHashInClosedRange updates the batch hash for chunks within the specified index range.
func (o *Chunk) UpdateBatchHashInClosedRange(ctx context.Context, startIndex uint64, endIndex uint64, batchHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.Model(&Chunk{}).Where("index >= ? AND index <= ?", startIndex, endIndex)

	if err := db.Update("batch_hash", batchHash).Error; err != nil {
		log.Error("failed to update batch_hash for chunks", "err", err)
		return err
	}
	return nil
}
