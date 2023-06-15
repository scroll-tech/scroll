package orm

import (
	"context"
	"encoding/hex"
	"errors"
	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/common/types"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
)

type Chunk struct {
	db *gorm.DB `gorm:"-"`

	// block
	Index            uint64 `json:"index" gorm:"column:index"`
	Hash             string `json:"hash" gorm:"column:hash"`
	StartBlockNumber uint64 `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash   string `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber   uint64 `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash     string `json:"end_block_hash" gorm:"column:end_block_hash"`
	TotalGasUsed     uint64 `json:"total_gas_used" gorm:"column:total_gas_used"`
	TotalTxNum       uint64 `json:"total_tx_num" gorm:"column:total_tx_num"`
	TotalPayloadSize uint64 `json:"total_payload_size" gorm:"column:total_payload_size"`

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

func NewChunk(db *gorm.DB) *Chunk {
	return &Chunk{db: db}
}

func (*Chunk) TableName() string {
	return "chunk"
}

func (o *Chunk) RangeGetChunks(ctx context.Context, startIndex uint64, endIndex uint64) ([]*Chunk, error) {
	if startIndex > endIndex {
		return nil, errors.New("start index should be less than or equal to end index")
	}

	var chunks []*Chunk
	db := o.db.WithContext(ctx)
	db = db.Where("index >= ? AND index <= ?", startIndex, endIndex)
	db = db.Order("index ASC")

	if err := db.Find(&chunks).Error; err != nil {
		return nil, err
	}

	if startIndex+uint64(len(chunks)) != endIndex+1 {
		return nil, errors.New("number of chunks not expected in the specified range")
	}

	return chunks, nil
}

func (o *Chunk) GetChunkIndexByHash(chunkHash string) (uint64, error) {
	var chunk Chunk
	if err := o.db.Where("hash = ?", chunkHash).First(&chunk).Error; err != nil {
		return 0, err
	}
	return chunk.Index, nil
}

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

func (o *Chunk) InsertChunk(ctx context.Context, chunk *bridgeTypes.Chunk, l2BlockOrm *L2Block, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	hash, err := chunk.Hash()
	if err != nil {
		log.Error("failed to get chunk hash", "err", err)
		return err
	}

	numBlocks := len(chunk.Blocks)
	if numBlocks == 0 {
		return errors.New("chunk must contain at least one block")
	}

	var totalGasUsed uint64
	var totalTxNum uint64
	var totalPayloadSize uint64
	for _, block := range chunk.Blocks {
		totalGasUsed += block.Header.GasUsed
		totalTxNum += uint64(len(block.Transactions))
		for _, tx := range block.Transactions {
			totalPayloadSize += uint64(len(tx.Data))
		}
	}

	var chunkIndex uint64
	var lastChunk Chunk
	if err := db.Order("index desc").First(&lastChunk).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
	} else {
		chunkIndex = lastChunk.Index + 1
	}

	tmpChunk := Chunk{
		Index:            chunkIndex,
		Hash:             hex.EncodeToString(hash),
		StartBlockNumber: chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:   chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:   chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:     chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
		TotalGasUsed:     totalGasUsed,
		TotalTxNum:       totalTxNum,
		TotalPayloadSize: totalPayloadSize,
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(&tmpChunk).Error; err != nil {
		log.Error("failed to insert chunk", "hash", hash, "err", err)
		tx.Rollback()
		return err
	}

	blockNumbers := make([]uint64, numBlocks)
	for i, block := range chunk.Blocks {
		blockNumbers[i] = block.Header.Number.Uint64()
	}

	if err := l2BlockOrm.UpdateChunkHashForL2Blocks(blockNumbers, tmpChunk.Hash, tx); err != nil {
		log.Error("failed to update chunk_hash for l2_blocks", "chunk_hash", tmpChunk.Hash, "block_numbers", blockNumbers, "err", err)
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// UpdateProvingStatus update the proving status
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

func (o *Chunk) UpdateBatchHashForChunks(chunkHashes []string, batchHash string, dbTX *gorm.DB) error {
	err := dbTX.Model(&Chunk{}).
		Where("hash IN ?", chunkHashes).
		Update("batch_hash", batchHash).
		Error

	if err != nil {
		log.Error("failed to update batch_hash for chunks", "err", err)
	}
	return err
}
