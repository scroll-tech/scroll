package orm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
)

// Bundle represents a bundle of batches.
type Bundle struct {
	db *gorm.DB `gorm:"column:-"`

	// bundle
	Index           uint64 `json:"index" gorm:"column:index;primaryKey"`
	Hash            string `json:"hash" gorm:"column:hash"`
	StartBatchIndex uint64 `json:"start_batch_index" gorm:"column:start_batch_index"`
	EndBatchIndex   uint64 `json:"end_batch_index" gorm:"column:end_batch_index"`
	StartBatchHash  string `json:"start_batch_hash" gorm:"column:start_batch_hash"`
	EndBatchHash    string `json:"end_batch_hash" gorm:"column:end_batch_hash"`
	CodecVersion    int16  `json:"codec_version" gorm:"column:codec_version"`

	// proof
	BatchProofsStatus int16      `json:"batch_proofs_status" gorm:"column:batch_proofs_status;default:1"`
	ProvingStatus     int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof             []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProvedAt          *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec      int32      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`

	// rollup
	RollupStatus   int16      `json:"rollup_status" gorm:"column:rollup_status;default:1"`
	FinalizeTxHash string     `json:"finalize_tx_hash" gorm:"column:finalize_tx_hash;default:NULL"`
	FinalizedAt    *time.Time `json:"finalized_at" gorm:"column:finalized_at;default:NULL"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;default:NULL"`
}

// NewBundle creates a new Bundle database instance.
func NewBundle(db *gorm.DB) *Bundle {
	return &Bundle{db: db}
}

// TableName returns the table name for the Bundle model.
func (*Bundle) TableName() string {
	return "bundle"
}

// getLatestBundle retrieves the latest bundle from the database.
func (o *Bundle) getLatestBundle(ctx context.Context) (*Bundle, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Order("index desc")

	var latestBundle Bundle
	if err := db.First(&latestBundle).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getLatestBundle error: %w", err)
	}
	return &latestBundle, nil
}

// GetBundles retrieves selected bundles from the database.
// The returned bundles are sorted in ascending order by their index.
// only used in unit tests.
func (o *Bundle) GetBundles(ctx context.Context, fields map[string]interface{}, orderByList []string, limit int) ([]*Bundle, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})

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

	var bundles []*Bundle
	if err := db.Find(&bundles).Error; err != nil {
		return nil, fmt.Errorf("Bundle.GetBundles error: %w, fields: %v, orderByList: %v", err, fields, orderByList)
	}
	return bundles, nil
}

// GetFirstUnbundledBatchIndex retrieves the first unbundled batch index.
func (o *Bundle) GetFirstUnbundledBatchIndex(ctx context.Context) (uint64, error) {
	// Get the latest bundle
	latestBundle, err := o.getLatestBundle(ctx)
	if err != nil {
		return 0, fmt.Errorf("Bundle.GetFirstUnbundledBatchIndex error: %w", err)
	}
	if latestBundle == nil {
		return 0, nil
	}
	return latestBundle.EndBatchIndex + 1, nil
}

// GetFirstPendingBundle retrieves the first pending bundle from the database.
func (o *Bundle) GetFirstPendingBundle(ctx context.Context) (*Bundle, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("rollup_status = ?", types.RollupPending)
	db = db.Order("index asc")

	var pendingBundle Bundle
	if err := db.First(&pendingBundle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("Bundle.GetFirstPendingBundle error: %w", err)
	}
	return &pendingBundle, nil
}

// GetVerifiedProofByHash retrieves the verified aggregate proof for a bundle with the given hash.
func (o *Bundle) GetVerifiedProofByHash(ctx context.Context, hash string) (*message.BundleProof, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Select("proof")
	db = db.Where("hash = ? AND proving_status = ?", hash, types.ProvingTaskVerified)

	var bundle Bundle
	if err := db.Find(&bundle).Error; err != nil {
		return nil, fmt.Errorf("Bundle.GetVerifiedProofByHash error: %w, bundle hash: %v", err, hash)
	}

	var proof message.BundleProof
	if err := json.Unmarshal(bundle.Proof, &proof); err != nil {
		return nil, fmt.Errorf("Bundle.GetVerifiedProofByHash error: %w, bundle hash: %v", err, hash)
	}
	return &proof, nil
}

// InsertBundle inserts a new bundle into the database.
// Assuming input batches are ordered by index.
func (o *Bundle) InsertBundle(ctx context.Context, batches []*Batch, codecVersion encoding.CodecVersion, dbTX ...*gorm.DB) (*Bundle, error) {
	if len(batches) == 0 {
		return nil, errors.New("Bundle.InsertBundle error: no batches provided")
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Bundle{})

	newBundle := Bundle{
		StartBatchHash:    batches[0].Hash,
		StartBatchIndex:   batches[0].Index,
		EndBatchHash:      batches[len(batches)-1].Hash,
		EndBatchIndex:     batches[len(batches)-1].Index,
		BatchProofsStatus: int16(types.BatchProofsStatusPending),
		ProvingStatus:     int16(types.ProvingTaskUnassigned),
		RollupStatus:      int16(types.RollupPending),
		CodecVersion:      int16(codecVersion),
	}

	// Not part of DA hash, used for SQL query consistency and ease of use.
	// Derived using keccak256(concat(start_batch_hash_bytes, end_batch_hash_bytes)).
	newBundle.Hash = hex.EncodeToString(crypto.Keccak256(append(common.Hex2Bytes(newBundle.StartBatchHash[2:]), common.Hex2Bytes(newBundle.EndBatchHash[2:])...)))

	if err := db.Create(&newBundle).Error; err != nil {
		return nil, fmt.Errorf("Bundle.InsertBundle Create error: %w, bundle hash: %v", err, newBundle.Hash)
	}

	return &newBundle, nil
}

// UpdateFinalizeTxHashAndRollupStatus updates the finalize transaction hash and rollup status for a bundle.
func (o *Bundle) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["finalize_tx_hash"] = finalizeTxHash
	updateFields["rollup_status"] = int(status)
	if status == types.RollupFinalized {
		updateFields["finalized_at"] = time.Now()
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Bundle.UpdateFinalizeTxHashAndRollupStatus error: %w, bundle hash: %v, status: %v, finalizeTxHash: %v", err, hash, status.String(), finalizeTxHash)
	}
	return nil
}

// UpdateProvingStatus updates the proving status of a bundle.
func (o *Bundle) UpdateProvingStatus(ctx context.Context, hash string, status types.ProvingStatus, dbTX ...*gorm.DB) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = int(status)

	switch status {
	case types.ProvingTaskVerified:
		updateFields["proved_at"] = time.Now()
	}

	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Bundle.UpdateProvingStatus error: %w, bundle hash: %v, status: %v", err, hash, status.String())
	}
	return nil
}

// UpdateRollupStatus updates the rollup status for a bundle.
// only used in unit tests.
func (o *Bundle) UpdateRollupStatus(ctx context.Context, hash string, status types.RollupStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["rollup_status"] = int(status)
	if status == types.RollupFinalized {
		updateFields["finalized_at"] = time.Now()
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Bundle.UpdateRollupStatus error: %w, bundle hash: %v, status: %v", err, hash, status.String())
	}
	return nil
}

// UpdateProofAndProvingStatusByHash updates the bundle proof and proving status by hash.
// only used in unit tests.
func (o *Bundle) UpdateProofAndProvingStatusByHash(ctx context.Context, hash string, proof *message.BundleProof, provingStatus types.ProvingStatus, proofTimeSec uint64, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	proofBytes, err := json.Marshal(proof)
	if err != nil {
		return err
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proofBytes
	updateFields["proving_status"] = provingStatus
	updateFields["proof_time_sec"] = proofTimeSec
	updateFields["proved_at"] = utils.NowUTC()

	db = db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Bundle.UpdateProofAndProvingStatusByHash error: %w, bundle hash: %v", err, hash)
	}
	return nil
}
