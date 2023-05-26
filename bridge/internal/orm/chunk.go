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
	db *gorm.DB `gorm:"column:-"`

	ChunkHash        string     `json:"chunk_hash" gorm:"column:chunk_hash"`
	BlockContexts    string     `json:"block_contexts" gorm:"column:block_contexts"`
	StartBlockNumber uint64     `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash   string     `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber   uint64     `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash     string     `json:"end_block_hash" gorm:"column:end_block_hash"`
	ZkEvmProof       []byte     `json:"zkevm_proof" gorm:"column:zkevm_proof;default:NULL"`
	ProofTimeSec     int        `json:"proof_time_sec" gorm:"column:proof_time_sec;default:0"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP()"`
}

func NewChunk(db *gorm.DB) *Chunk {
	return &Chunk{db: db}
}

func (*Chunk) TableName() string {
	return "chunk"
}

func (c *Chunk) GetChunk(ctx context.Context, chunkHash string) (*Chunk, error) {
	var chunk Chunk
	err := c.db.WithContext(ctx).Where("chunk_hash", chunkHash).First(&chunk).Error
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

	chunkHash, err := chunk.Hash()
	if err != nil {
		log.Error("failed to get chunk hash", "err", err)
		return err
	}

	blockContexts, err := chunk.Encode()
	if err != nil {
		log.Error("failed to encode chunk", "chunk hash", chunkHash, "err", err)
		return err
	}

	numBlocks := len(chunk.Blocks)
	if numBlocks == 0 {
		return errors.New("chunk must contain at least one block")
	}

	tmpChunk := Chunk{
		ChunkHash:        hex.EncodeToString(chunkHash),
		BlockContexts:    hex.EncodeToString(blockContexts),
		StartBlockNumber: chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:   chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:   chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:     chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
	}

	if err := db.WithContext(ctx).Create(&tmpChunk).Error; err != nil {
		log.Error("failed to insert chunk", "chunk hash", chunkHash, "err", err)
		return err
	}
	return nil
}

func (c *Chunk) UpdateChunk(ctx context.Context, chunkHash string, updateFields map[string]interface{}, tx ...*gorm.DB) error {
	db := c.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}
	err := db.Model(&Chunk{}).WithContext(ctx).Where("chunk_hash", chunkHash).Updates(updateFields).Error
	return err
}
