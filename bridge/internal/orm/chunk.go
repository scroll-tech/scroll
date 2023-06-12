package orm

import (
	"context"
	"encoding/hex"
	"errors"
	"scroll-tech/bridge/internal/types"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
)

type Chunk struct {
	db *gorm.DB `gorm:"-"`

	Hash             string     `json:"hash" gorm:"column:hash"`
	StartBlockNumber uint64     `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash   string     `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber   uint64     `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash     string     `json:"end_block_hash" gorm:"column:end_block_hash"`
	ChunkProof       []byte     `json:"chunk_proof" gorm:"column:chunk_proof"`
	ProofTimeSec     int        `json:"proof_time_sec" gorm:"column:proof_time_sec"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at"`
	ProvingStatus    int        `json:"proving_status" gorm:"column:proving_status"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at"`
	BatchHash        string     `json:"batch_hash" gorm:"column:batch_hash"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

func NewChunk(db *gorm.DB) *Chunk {
	return &Chunk{db: db}
}

func (*Chunk) TableName() string {
	return "chunk"
}

func (c *Chunk) GetChunk(ctx context.Context, hash string) (*Chunk, error) {
	var chunk Chunk
	err := c.db.WithContext(ctx).Where("hash", hash).First(&chunk).Error
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

func (c *Chunk) GetChunksInRange(ctx context.Context, startIndex int, endIndex int) ([]*Chunk, error) {
	if startIndex > endIndex {
		return nil, errors.New("start index should be less than or equal to end index")
	}

	var chunks []*Chunk
	db := c.db.WithContext(ctx)
	db = db.Where("index >= ? AND index <= ?", startIndex, endIndex)

	if err := db.Find(&chunks).Error; err != nil {
		return nil, err
	}

	if len(chunks) != endIndex-startIndex+1 {
		return nil, errors.New("number of chunks not expected in the specified range")
	}

	return chunks, nil
}

func (c *Chunk) GetUnbatchedChunks(ctx context.Context) ([]*Chunk, error) {
	var chunks []*Chunk
	err := c.db.WithContext(ctx).
		Where("batch_hash IS NULL OR batch_hash = ''").
		Order("start_block_number asc").
		Find(&chunks).Error
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

func (c *Chunk) InsertChunk(ctx context.Context, chunk *types.Chunk, l2BlockOrm *L2Block, dbTX ...*gorm.DB) error {
	db := c.db
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

	tmpChunk := Chunk{
		Hash:             hex.EncodeToString(hash),
		StartBlockNumber: chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:   chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:   chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:     chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
	}

	// Start a new transaction
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

	// Update the chunk_hash for all blocks in the chunk
	if err := l2BlockOrm.UpdateChunkHashForL2Blocks(blockNumbers, tmpChunk.Hash, tx); err != nil {
		log.Error("failed to update chunk_hash for l2_blocks", "chunk_hash", tmpChunk.Hash, "block_numbers", blockNumbers, "err", err)
		tx.Rollback()
		return err
	}

	// If all operations succeed, then commit the transaction
	tx.Commit()
	return nil
}

func (c *Chunk) UpdateChunk(ctx context.Context, hash string, updateFields map[string]interface{}, dbTX ...*gorm.DB) error {
	db := c.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	err := db.Model(&Chunk{}).WithContext(ctx).Where("hash", hash).Updates(updateFields).Error
	return err
}

func (c *Chunk) UpdateBatchHashForChunks(chunkHashes []string, batchHash string, dbTX *gorm.DB) error {
	err := dbTX.Model(&Chunk{}).
		Where("hash IN ?", chunkHashes).
		Update("batch_hash", batchHash).
		Error

	if err != nil {
		log.Error("failed to update batch_hash for chunks", "err", err)
	}
	return err
}
