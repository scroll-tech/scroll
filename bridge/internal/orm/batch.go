package orm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bridgeTypes "scroll-tech/bridge/internal/types"
	"scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"
)

const defaultBatchHeaderVersion = 0

type Batch struct {
	db *gorm.DB `gorm:"column:-"`

	// batch
	Index           uint64 `json:"index" gorm:"column:index"`
	Hash            string `json:"hash" gorm:"column:hash"`
	StartChunkIndex uint64 `json:"start_chunk_index" gorm:"column:start_chunk_index"`
	StartChunkHash  string `json:"start_chunk_hash" gorm:"column:start_chunk_hash"`
	EndChunkIndex   uint64 `json:"end_chunk_index" gorm:"column:end_chunk_index"`
	EndChunkHash    string `json:"end_chunk_hash" gorm:"column:end_chunk_hash"`
	StateRoot       string `json:"state_root" gorm:"column:state_root"`
	WithdrawRoot    string `json:"withdraw_root" gorm:"column:withdraw_root"`
	BatchHeader     []byte `json:"batch_header" gorm:"column:batch_header"`

	// proof
	ProvingStatus    int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof            []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProverAssignedAt *time.Time `json:"prover_assigned_at" gorm:"column:prover_assigned_at;default:NULL"`
	ProvedAt         *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec     int        `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`

	// rollup
	RollupStatus   int16      `json:"rollup_status" gorm:"column:rollup_status;default:1"`
	CommitTxHash   string     `json:"commit_tx_hash" gorm:"column:commit_tx_hash;default:NULL"`
	CommittedAt    *time.Time `json:"committed_at" gorm:"column:committed_at;default:NULL"`
	FinalizeTxHash string     `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash;default:NULL"`
	FinalizedAt    *time.Time `json:"finalized_at" gorm:"column:finalized_at;default:NULL"`

	// gas oracle
	OracleStatus int16  `json:"oracle_status" gorm:"column:oracle_status;default:1"`
	OracleTxHash string `json:"oracle_tx_hash" gorm:"column:oracle_tx_hash;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

func NewBatch(db *gorm.DB) *Batch {
	return &Batch{db: db}
}

func (*Batch) TableName() string {
	return "batch"
}

// GetBatches retrieves selected batches from the database
func (o *Batch) GetBatches(ctx context.Context, fields map[string]interface{}, orderByList []string, limit int) ([]*Batch, error) {
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

	var batches []*Batch
	if err := db.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

func (o *Batch) GetBatchCount(ctx context.Context) (uint64, error) {
	var count int64
	err := o.db.WithContext(ctx).Model(&Batch{}).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func (o *Batch) GetVerifiedProofByHash(ctx context.Context, hash string) (*message.AggProof, error) {
	var batch Batch
	err := o.db.WithContext(ctx).Where("hash = ? AND proving_status = ?", hash, types.ProvingTaskVerified).First(&batch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	var proof message.AggProof
	err = json.Unmarshal(batch.Proof, &proof)
	if err != nil {
		return nil, err
	}

	return &proof, nil
}

func (o *Batch) GetLatestBatch(ctx context.Context) (*Batch, error) {
	var latestBatch Batch
	err := o.db.WithContext(ctx).Order("index DESC").First(&latestBatch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &latestBatch, nil
}

func (o *Batch) GetLatestBatchByRollupStatus(statuses []types.RollupStatus) (*Batch, error) {
	var batch Batch
	interfaceStatuses := make([]interface{}, len(statuses))
	for i, v := range statuses {
		interfaceStatuses[i] = v
	}
	err := o.db.Where("rollup_status IN ?", interfaceStatuses).Order("index desc").First(&batch).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

func (o *Batch) GetRollupStatusByHashList(ctx context.Context, hashes []string) ([]types.RollupStatus, error) {
	if len(hashes) == 0 {
		return []types.RollupStatus{}, nil
	}

	var batches []Batch
	err := o.db.WithContext(ctx).Where("hash IN ?", hashes).Find(&batches).Error
	if err != nil {
		return nil, err
	}

	hashToStatusMap := make(map[string]types.RollupStatus)
	for _, batch := range batches {
		hashToStatusMap[batch.Hash] = types.RollupStatus(batch.RollupStatus)
	}

	var statuses []types.RollupStatus
	for _, hash := range hashes {
		status, ok := hashToStatusMap[hash]
		if !ok {
			return nil, fmt.Errorf("hash not found in database: %s", hash)
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (o *Batch) GetPendingBatches(ctx context.Context, limit int) ([]*Batch, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be greater than zero")
	}

	var batches []*Batch
	db := o.db.WithContext(ctx)

	db = db.Where("rollup_status = ?", types.RollupPending).Order("index ASC").Limit(limit)

	if err := db.Find(&batches).Error; err != nil {
		return nil, err
	}

	if len(batches) == 0 {
		log.Warn("no pending batches in db")
		return nil, nil
	}

	return batches, nil
}

// GetBatchHeader retrieves the header of a batch with a given index
func (o *Batch) GetBatchHeader(ctx context.Context, index uint64) (*bridgeTypes.BatchHeader, error) {
	var batch Batch
	err := o.db.WithContext(ctx).Where("index = ?", index).First(&batch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	batchHeader, err := bridgeTypes.DecodeBatchHeader(batch.BatchHeader)
	if err != nil {
		return nil, err
	}
	return batchHeader, nil
}

func (o *Batch) InsertBatch(ctx context.Context, startChunkIndex, endChunkIndex uint64, startChunkHash, endChunkHash string, chunks []*bridgeTypes.Chunk, dbTX ...*gorm.DB) (string, error) {
	if chunks == nil || len(chunks) == 0 {
		return "", errors.New("invalid args")
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	lastBatch, err := o.GetLatestBatch(ctx)
	if err != nil {
		log.Error("failed to get the latest batch", "err", err)
		return "", err
	}
	var batchIndex uint64
	var parentBatchHash common.Hash
	var totalL1MessagePoppedBefore uint64
	var version uint8 = defaultBatchHeaderVersion
	if lastBatch != nil {
		batchIndex = lastBatch.Index + 1
		parentBatchHash = common.HexToHash(lastBatch.Hash)
		lastBatchHeader, err := bridgeTypes.DecodeBatchHeader(lastBatch.BatchHeader)
		if err != nil {
			return "", err
		}
		totalL1MessagePoppedBefore = lastBatchHeader.TotalL1MessagePopped()
		version = lastBatchHeader.Version()
	}

	batchHeader, err := bridgeTypes.NewBatchHeader(version, batchIndex, totalL1MessagePoppedBefore, parentBatchHash, chunks)
	if err != nil {
		log.Error("failed to create batch header", "err", err)
		return "", err
	}

	numChunks := len(chunks)
	lastChunkBlockNum := len(chunks[numChunks-1].Blocks)
	newBatch := Batch{
		Index:           batchIndex,
		Hash:            batchHeader.Hash().Hex(),
		StartChunkHash:  startChunkHash,
		StartChunkIndex: startChunkIndex,
		EndChunkHash:    endChunkHash,
		EndChunkIndex:   endChunkIndex,
		StateRoot:       chunks[numChunks-1].Blocks[lastChunkBlockNum-1].Header.Root.Hex(),
		WithdrawRoot:    chunks[numChunks-1].Blocks[lastChunkBlockNum-1].WithdrawTrieRoot.Hex(),
		BatchHeader:     batchHeader.Encode(),
	}

	if err := db.WithContext(ctx).Create(&newBatch).Error; err != nil {
		log.Error("failed to insert batch", "err", err)
		return "", err
	}
	return newBatch.Hash, nil
}

// UpdateSkippedBatches update the skipped batches
func (o *Batch) UpdateSkippedBatches(ctx context.Context) (uint64, error) {
	provingStatusList := []interface{}{
		int(types.ProvingTaskSkipped),
		int(types.ProvingTaskFailed),
	}
	result := o.db.Model(&Batch{}).Where("rollup_status", int(types.RollupCommitted)).
		Where("proving_status IN (?)", provingStatusList).Update("rollup_status", int(types.RollupFinalizationSkipped))
	if result.Error != nil {
		return 0, result.Error
	}
	return uint64(result.RowsAffected), nil
}

// UpdateL2GasOracleStatusAndOracleTxHash update the l2 gas oracle status and oracle tx hash
func (o *Batch) UpdateL2GasOracleStatusAndOracleTxHash(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
	updateFields := make(map[string]interface{})
	updateFields["oracle_status"] = int(status)
	updateFields["oracle_tx_hash"] = txHash
	if err := o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateProvingStatus update the proving status
func (o *Batch) UpdateProvingStatus(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
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

	if err := db.Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateRollupStatus update the rollup status
func (o *Batch) UpdateRollupStatus(ctx context.Context, hash string, status types.RollupStatus, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	updateFields := make(map[string]interface{})
	updateFields["rollup_status"] = int(status)

	switch status {
	case types.RollupCommitted:
		updateFields["committed_at"] = time.Now()
	case types.RollupFinalized:
		updateFields["finalized_at"] = time.Now()
	}
	if err := db.Model(&Batch{}).WithContext(ctx).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateCommitTxHashAndRollupStatus update the commit tx hash and rollup status
func (o *Batch) UpdateCommitTxHashAndRollupStatus(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["commit_tx_hash"] = commitTxHash
	updateFields["rollup_status"] = int(status)
	if status == types.RollupCommitted {
		updateFields["committed_at"] = time.Now()
	}
	if err := o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}

// UpdateFinalizeTxHashAndRollupStatus update the finalize tx hash and rollup status
func (o *Batch) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["finalize_tx_hash"] = finalizeTxHash
	updateFields["rollup_status"] = int(status)
	if status == types.RollupFinalized {
		updateFields["finalized_at"] = time.Now()
	}
	if err := o.db.WithContext(ctx).Model(&Batch{}).Where("hash", hash).Updates(updateFields).Error; err != nil {
		return err
	}
	return nil
}
