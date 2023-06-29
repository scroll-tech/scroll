package orm

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
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

	// proof
	ProvingStatus    int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof            []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec     int16      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`

	// batch
	BatchHash string `json:"batch_hash" gorm:"column:batch_hash;default:NULL"`

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

// GetChunks retrieves selected chunks from the database.
// The returned chunks are sorted in ascending order by their index.
func (o *Chunk) GetChunks(ctx context.Context, fields map[string]interface{}, orderByList []string, limit int) ([]*Chunk, error) {
	db := o.db.WithContext(ctx)

	for key, value := range fields {
		db = db.Where(key, value)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit > 0 {
		db = db.Limit(limit)
	}

	db = db.Order("index ASC")

	var chunks []*Chunk
	if err := db.Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetUnassignedChunks retrieves all or limited number of unassigned chunks based on the specified limit.
// The returned chunks are sorted in ascending order by their index.
func (o *Chunk) GetUnassignedChunks(ctx context.Context) ([]*Chunk, error) {
	var chunks []*Chunk
	db := o.db.WithContext(ctx)
	db = db.Where("proving_status = ?", types.ProvingTaskUnassigned).Order("index ASC")

	if err := db.Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetProofsByBatchHash retrieves the proofs associated with a specific batch hash.
// It returns a slice of decoded proofs (message.AggProof) obtained from the database.
// The returned proofs are sorted in ascending order by their associated chunk index.
func (o *Chunk) GetProofsByBatchHash(ctx context.Context, batchHash string) ([]*message.AggProof, error) {
	// The returned chunks are sorted in ascending order by their index.
	chunks, err := o.GetChunks(ctx, map[string]interface{}{"batch_hash": batchHash}, nil, 0)
	if err != nil {
		return nil, err
	}

	var proofs []*message.AggProof
	for _, chunk := range chunks {
		var proof message.AggProof
		if err := json.Unmarshal(chunk.Proof, &proof); err != nil {
			return nil, err
		}

		proofs = append(proofs, &proof)
	}

	return proofs, nil
}

// GetLatestChunk retrieves the latest chunk from the database.
func (o *Chunk) GetLatestChunk(ctx context.Context) (*Chunk, error) {
	var latestChunk Chunk
	err := o.db.WithContext(ctx).
		Order("index desc").
		First(&latestChunk).Error
	if err != nil {
		return nil, err
	}
	return &latestChunk, nil
}

// GetProvingStatusByHash retrieves the proving status of a chunk given its hash.
func (o *Chunk) GetProvingStatusByHash(ctx context.Context, hash string) (types.ProvingStatus, error) {
	var chunk Chunk
	err := o.db.WithContext(ctx).Where("hash = ?", hash).First(&chunk).Error
	if err != nil {
		return types.ProvingStatusUndefined, err
	}
	return types.ProvingStatus(chunk.ProvingStatus), nil
}

// InsertChunk inserts a new chunk into the database.
func (o *Chunk) InsertChunk(ctx context.Context, chunk *types.Chunk, dbTX ...*gorm.DB) (*Chunk, error) {
	if chunk == nil || len(chunk.Blocks) == 0 {
		return nil, errors.New("invalid args")
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	var chunkIndex uint64
	var totalL1MessagePoppedBefore uint64
	parentChunk, err := o.GetLatestChunk(ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("failed to get latest chunk", "err", err)
		return nil, err
	}

	// if parentChunk==nil then err==gorm.ErrRecordNotFound, which means there's
	// not chunk record in the db, we then use default empty values for the creating chunk;
	// if parentChunk!=nil then err=nil, then we fill the parentChunk-related data into the creating chunk
	if parentChunk != nil {
		chunkIndex = parentChunk.Index + 1
		totalL1MessagePoppedBefore = parentChunk.TotalL1MessagesPoppedBefore + parentChunk.TotalL1MessagesPoppedInChunk
	}

	hash, err := chunk.Hash(totalL1MessagePoppedBefore)
	if err != nil {
		log.Error("failed to get chunk hash", "err", err)
		return nil, err
	}

	var totalL2TxGas uint64
	var totalL2TxNum uint64
	var totalL1CommitCalldataSize uint64
	var totalL1CommitGas uint64
	for _, block := range chunk.Blocks {
		totalL2TxGas += block.Header.GasUsed
		totalL2TxNum += block.L2TxsNum()
		totalL1CommitCalldataSize += block.EstimateL1CommitCalldataSize()
		totalL1CommitGas += block.EstimateL1CommitGas()
	}

	numBlocks := len(chunk.Blocks)
	newChunk := Chunk{
		Index:                        chunkIndex,
		Hash:                         hash.Hex(),
		StartBlockNumber:             chunk.Blocks[0].Header.Number.Uint64(),
		StartBlockHash:               chunk.Blocks[0].Header.Hash().Hex(),
		EndBlockNumber:               chunk.Blocks[numBlocks-1].Header.Number.Uint64(),
		EndBlockHash:                 chunk.Blocks[numBlocks-1].Header.Hash().Hex(),
		TotalL2TxGas:                 totalL2TxGas,
		TotalL2TxNum:                 totalL2TxNum,
		TotalL1CommitCalldataSize:    totalL1CommitCalldataSize,
		TotalL1CommitGas:             totalL1CommitGas,
		StartBlockTime:               chunk.Blocks[0].Header.Time,
		TotalL1MessagesPoppedBefore:  totalL1MessagePoppedBefore,
		TotalL1MessagesPoppedInChunk: chunk.NumL1Messages(totalL1MessagePoppedBefore),
		ProvingStatus:                int16(types.ProvingTaskUnassigned),
	}

	if err := db.Create(&newChunk).Error; err != nil {
		log.Error("failed to insert chunk", "hash", hash, "err", err)
		return nil, err
	}

	return &newChunk, nil
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

// UpdateProofByHash updates the chunk proof by hash.
func (o *Chunk) UpdateProofByHash(ctx context.Context, hash string, proof *message.AggProof, proofTimeSec uint64) error {
	proofBytes, err := json.Marshal(proof)
	if err != nil {
		return err
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proofBytes
	updateFields["proof_time_sec"] = proofTimeSec
	err = o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error
	return err
}
