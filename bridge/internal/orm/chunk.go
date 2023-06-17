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
	Index                      uint64 `json:"index" gorm:"column:index"`
	Hash                       string `json:"hash" gorm:"column:hash"`
	StartBlockNumber           uint64 `json:"start_block_number" gorm:"column:start_block_number"`
	StartBlockHash             string `json:"start_block_hash" gorm:"column:start_block_hash"`
	EndBlockNumber             uint64 `json:"end_block_number" gorm:"column:end_block_number"`
	EndBlockHash               string `json:"end_block_hash" gorm:"column:end_block_hash"`
	TotalGasUsed               uint64 `json:"total_gas_used" gorm:"column:total_gas_used"`
	TotalTxNum                 uint64 `json:"total_tx_num" gorm:"column:total_tx_num"`
	TotalPayloadSize           uint64 `json:"total_payload_size" gorm:"column:total_payload_size"`
	TotalL1MessagePoppedBefore uint64 `json:"total_l1_messages_popped_before" gorm:"column:total_l1_messages_popped_before"`
	TotalL1Messages            uint64 `json:"total_l1_messages" gorm:"column:total_l1_messages"`

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

func (o *Chunk) GetChunksInClosedRange(ctx context.Context, startIndex uint64, endIndex uint64) ([]*Chunk, error) {
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

func (o *Chunk) GetChunkByStartBlockIndex(ctx context.Context, startBlockNumber uint64) (*Chunk, error) {
	var chunk Chunk
	if err := o.db.Where("start_block_number = ?", startBlockNumber).First(&chunk).Error; err != nil {
		return nil, err
	}
	return &chunk, nil
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

func (o *Chunk) GetTotalL1MessagePoppedByEndBlockNumber(ctx context.Context, endBlockNumber uint64) (uint64, error) {
	var chunk Chunk
	if err := o.db.Where("endBlockNumber = ?", endBlockNumber).First(&chunk).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	return chunk.TotalL1MessagePoppedBefore + chunk.TotalL1Messages, nil
}

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

func (o *Chunk) InsertChunk(ctx context.Context, chunk *bridgeTypes.Chunk, dbTX ...*gorm.DB) error {
	if chunk == nil || len(chunk.Blocks) == 0 {
		return errors.New("invalid args")
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	var totalL1MessagePoppedBefore uint64
	parentChunk, err := o.GetLatestChunk(ctx)
	if err != nil {
		log.Error("failed to get latest chunk", "err", err)
		return err
	}
	if parentChunk != nil {
		totalL1MessagePoppedBefore = parentChunk.TotalL1MessagePoppedBefore + parentChunk.TotalL1Messages
	}
	hash, err := chunk.Hash(totalL1MessagePoppedBefore)
	if err != nil {
		log.Error("failed to get chunk hash", "err", err)
		return err
	}

	// TODO: implement an exact payload size calculation.
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

	numBlocks := len(chunk.Blocks)
	tmpChunk := Chunk{
		Index:                      chunkIndex,
		Hash:                       hex.EncodeToString(hash),
		StartBlockNumber:           chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:             chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:             chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:               chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
		TotalGasUsed:               totalGasUsed,
		TotalTxNum:                 totalTxNum,
		TotalPayloadSize:           totalPayloadSize,
		TotalL1MessagePoppedBefore: totalL1MessagePoppedBefore,
		TotalL1Messages:            chunk.NumL1Messages(totalL1MessagePoppedBefore),
	}

	if err := db.Create(&tmpChunk).Error; err != nil {
		log.Error("failed to insert chunk", "hash", hash, "err", err)
		return err
	}
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
