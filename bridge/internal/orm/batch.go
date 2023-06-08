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

type Batch struct {
	db *gorm.DB `gorm:"column:-"`

	BatchHash        string     `json:"batch_hash" gorm:"column:batch_hash"`
	StartChunkIndex  int        `json:"start_chunk_index" gorm:"column:start_chunk_index"`
	StartChunkHash   string     `json:"start_chunk_hash" gorm:"column:start_chunk_hash"`
	EndChunkIndex    int        `json:"end_chunk_index" gorm:"column:end_chunk_index"`
	EndChunkHash     string     `json:"end_chunk_hash" gorm:"column:end_chunk_hash"`
	Proof            []byte     `json:"proof" gorm:"column:proof"`
	ProvingStatus    int        `json:"proving_status" gorm:"column:proving_status"`
	ProofTimeSec     int        `json:"proof_time_sec" gorm:"column:proof_time_sec"`
	RollupStatus     int        `json:"rollup_status" gorm:"column:rollup_status"`
	CommitTxHash     string     `json:"commit_tx_hash" gorm:"column:commit_tx_hash"`
	FinalizeTxHash   string     `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at"`
	CommittedAt      *time.Time `json:"committed_at" gorm:"column:committed_at"`
	FinalizedAt      *time.Time `json:"finalized_at" gorm:"column:finalized_at"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt        *time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

func NewBatch(db *gorm.DB) *Batch {
	return &Batch{db: db}
}

func (*Batch) TableName() string {
	return "batch"
}

func (c *Batch) GetChunkBatch(ctx context.Context, batchHash string) (*Batch, error) {
	var chunkBatch Batch
	err := c.db.WithContext(ctx).Where("batch_hash", batchHash).First(&chunkBatch).Error
	if err != nil {
		return nil, err
	}
	return &chunkBatch, nil
}

func (c *Batch) GetBatchCount(ctx context.Context) (int64, error) {
	var count int64
	err := c.db.WithContext(ctx).Model(&Batch{}).Count(&count).Error
	return count, err
}

func (c *Batch) InsertChunkBatch(ctx context.Context, batch *types.Batch, tx ...*gorm.DB) error {
	db := c.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}

	numChunks := len(batch.Chunks)
	if numChunks == 0 {
		return errors.New("chunkBatch must contain at least one chunk")
	}

	startChunkHash, err := batch.Chunks[0].Hash()
	if err != nil {
		log.Error("failed to get start chunk hash", "err", err)
		return err
	}

	endChunkHash, err := batch.Chunks[numChunks-1].Hash()
	if err != nil {
		log.Error("failed to get end chunk hash", "err", err)
		return err
	}

	tmpChunkBatch := Batch{
		StartChunkHash: hex.EncodeToString(startChunkHash),
		EndChunkHash:   hex.EncodeToString(endChunkHash),
	}

	err = db.WithContext(ctx).Create(&tmpChunkBatch).Error
	return err
}

func (c *Batch) UpdateChunkBatch(ctx context.Context, batchHash string, updateFields map[string]interface{}, tx ...*gorm.DB) error {
	db := c.db
	if len(tx) > 0 && tx[0] != nil {
		db = tx[0]
	}
	err := db.Model(&Batch{}).WithContext(ctx).Where("batch_hash", batchHash).Updates(updateFields).Error
	return err
}
