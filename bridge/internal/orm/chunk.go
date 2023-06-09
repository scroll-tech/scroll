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
	BatchIndex       int        `json:"batch_index" gorm:"column:batch_index"`
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

func (c *Chunk) InsertChunk(ctx context.Context, chunk *types.Chunk, tx ...*gorm.DB) error {
	db := c.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
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

	if err := db.WithContext(ctx).Create(&tmpChunk).Error; err != nil {
		log.Error("failed to insert chunk", "hash", hash, "err", err)
		return err
	}
	return nil
}

func (c *Chunk) UpdateChunk(ctx context.Context, hash string, updateFields map[string]interface{}, tx ...*gorm.DB) error {
	db := c.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}
	err := db.Model(&Chunk{}).WithContext(ctx).Where("hash", hash).Updates(updateFields).Error
	return err
}
