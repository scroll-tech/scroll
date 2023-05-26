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

type ChunkBatch struct {
	db *gorm.DB `gorm:"column:-"`

	BatchHash        string     `json:"batch_hash" gorm:"column:batch_hash"`
	StartChunkHash   string     `json:"start_chunk_hash" gorm:"column:start_chunk_hash"`
	EndChunkHash     string     `json:"end_chunk_hash" gorm:"column:end_chunk_hash"`
	StartBlockNumber uint64     `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash   string     `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber   uint64     `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash     string     `json:"end_block_hash" gorm:"column:end_block_hash"`
	AggProof         []byte     `json:"agg_proof" gorm:"column:agg_proof;default:NULL"`
	ProvingStatus    int        `json:"proving_status" gorm:"column:proving_status;default:1"`
	ProofTimeSec     int        `json:"proof_time_sec" gorm:"column:proof_time_sec;default:0"`
	RollupStatus     int        `json:"rollup_status" gorm:"column:rollup_status;default:1"`
	CommitTxHash     string     `json:"commit_tx_hash" gorm:"column:commit_tx_hash;default:NULL"`
	FinalizeTxHash   string     `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash;default:NULL"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	CommittedAt      *time.Time `json:"committed_at" gorm:"column:committed_at;default:NULL"`
	FinalizedAt      *time.Time `json:"finalized_at" gorm:"column:finalized_at;default:NULL"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at"`
}

func NewChunkBatch(db *gorm.DB) *ChunkBatch {
	return &ChunkBatch{db: db}
}

func (*ChunkBatch) TableName() string {
	return "chunk_batch"
}

func (c *ChunkBatch) GetChunkBatch(ctx context.Context, batchHash string) (*ChunkBatch, error) {
	var chunkBatch ChunkBatch
	err := c.db.WithContext(ctx).Where("batch_hash", batchHash).First(&chunkBatch).Error
	if err != nil {
		return nil, err
	}
	return &chunkBatch, nil
}

func (c *ChunkBatch) GetBatchCount(ctx context.Context) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&ChunkBatch{}).Count(&count).Error
	return count, err
}

func (c *ChunkBatch) InsertChunkBatch(ctx context.Context, chunkBatch *types.Batch, tx ...*gorm.DB) error {
	db := c.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}

	numChunks := len(chunkBatch.Chunks)
	if numChunks == 0 {
		return errors.New("chunkBatch must contain at least one chunk")
	}

	startChunkHash, err := chunkBatch.Chunks[0].Hash()
	if err != nil {
		log.Error("failed to get start chunk hash", "err", err)
		return err
	}

	endChunkHash, err := chunkBatch.Chunks[numChunks-1].Hash()
	if err != nil {
		log.Error("failed to get end chunk hash", "err", err)
		return err
	}

	tmpChunkBatch := ChunkBatch{
		StartChunkHash:   hex.EncodeToString(startChunkHash),
		EndChunkHash:     hex.EncodeToString(endChunkHash),
		StartBlockNumber: chunkBatch.Chunks[0].Blocks[0].Header.Number.Uint64(),
		StartBlockHash:   chunkBatch.Chunks[0].Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:   chunkBatch.Chunks[numChunks-1].Blocks[len(chunkBatch.Chunks[numChunks-1].Blocks)-1].Header.Number.Uint64(),
		EndBlockHash:     chunkBatch.Chunks[numChunks-1].Blocks[len(chunkBatch.Chunks[numChunks-1].Blocks)-1].Header.Hash().Hex(),
	}

	err = db.WithContext(ctx).Create(&tmpChunkBatch).Error
	return err
}

func (c *ChunkBatch) UpdateChunkBatch(ctx context.Context, batchHash string, updateFields map[string]interface{}, tx ...*gorm.DB) error {
	db := c.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}
	err := db.Model(&ChunkBatch{}).WithContext(ctx).Where("batch_hash", batchHash).Updates(updateFields).Error
	return err
}
