package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/common/types"
	"scroll-tech/common/utils"
)

// Bundle represents a bundle of batches.
type Bundle struct {
	db *gorm.DB `gorm:"column:-"`

	Index           uint64 `json:"index" gorm:"column:index"`
	Hash            string `json:"hash" gorm:"column:hash"`
	StartBatchIndex uint64 `json:"start_batch_index" gorm:"column:start_batch_index"`
	StartBatchHash  string `json:"start_batch_hash" gorm:"column:start_batch_hash"`
	EndBatchIndex   uint64 `json:"end_batch_index" gorm:"column:end_batch_index"`
	EndBatchHash    string `json:"end_batch_hash" gorm:"column:end_batch_hash"`

	// proof
	BatchProofsStatus int16      `json:"batch_proofs_status" gorm:"column:batch_proofs_status;default:1"`
	ProvingStatus     int16      `json:"proving_status" gorm:"column:proving_status;default:1"`
	Proof             []byte     `json:"proof" gorm:"column:proof;default:NULL"`
	ProvedAt          *time.Time `json:"proved_at" gorm:"column:proved_at;default:NULL"`
	ProofTimeSec      int32      `json:"proof_time_sec" gorm:"column:proof_time_sec;default:NULL"`
	TotalAttempts     int16      `json:"total_attempts" gorm:"column:total_attempts;default:0"`
	ActiveAttempts    int16      `json:"active_attempts" gorm:"column:active_attempts;default:0"`

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

// GetUnassignedBundle retrieves unassigned bundle based on the specified limit.
// The returned batch sorts in ascending order by their index.
func (o *Bundle) GetUnassignedBundle(ctx context.Context, maxActiveAttempts, maxTotalAttempts uint8) (*Bundle, error) {
	var bundle Bundle
	db := o.db.WithContext(ctx)
	sql := fmt.Sprintf("SELECT * FROM bundle WHERE proving_status = %d AND total_attempts < %d AND active_attempts < %d AND batch_proofs_status = %d AND bundle.deleted_at IS NULL ORDER BY bundle.index LIMIT 1;",
		int(types.ProvingTaskUnassigned), maxTotalAttempts, maxActiveAttempts, int(types.BatchProofsStatusReady))
	err := db.Raw(sql).Scan(&bundle).Error
	if err != nil {
		return nil, fmt.Errorf("Batch.GetUnassignedBundle error: %w", err)
	}
	if bundle.StartBatchHash == "" || bundle.EndBatchHash == "" {
		return nil, nil
	}
	return &bundle, nil
}

// GetAssignedBundle retrieves assigned bundle based on the specified limit.
// The returned bundle sorts in ascending order by their index.
func (o *Bundle) GetAssignedBundle(ctx context.Context, maxActiveAttempts, maxTotalAttempts uint8) (*Bundle, error) {
	var bundle Bundle
	db := o.db.WithContext(ctx)
	sql := fmt.Sprintf("SELECT * FROM bundle WHERE proving_status = %d AND total_attempts < %d AND active_attempts < %d AND batch_proofs_status = %d AND bundle.deleted_at IS NULL ORDER BY bundle.index LIMIT 1;",
		int(types.ProvingTaskAssigned), maxTotalAttempts, maxActiveAttempts, int(types.BatchProofsStatusReady))
	err := db.Raw(sql).Scan(&bundle).Error
	if err != nil {
		return nil, fmt.Errorf("Bundle.GetAssignedBatch error: %w", err)
	}
	if bundle.StartBatchHash == "" || bundle.EndBatchHash == "" {
		return nil, nil
	}
	return &bundle, nil
}

// GetProvingStatusByHash retrieves the proving status of a bundle given its hash.
func (o *Bundle) GetProvingStatusByHash(ctx context.Context, hash string) (types.ProvingStatus, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Select("proving_status")
	db = db.Where("hash = ?", hash)

	var bundle Bundle
	if err := db.Find(&bundle).Error; err != nil {
		return types.ProvingStatusUndefined, fmt.Errorf("Bundle.GetProvingStatusByHash error: %w, batch hash: %v", err, hash)
	}
	return types.ProvingStatus(bundle.ProvingStatus), nil
}

// GetBundleByHash retrieves the given
func (o *Bundle) GetBundleByHash(ctx context.Context, bundleHash string) (*Bundle, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash = ?", bundleHash)

	var bundle Bundle
	if err := db.First(&bundle).Error; err != nil {
		return nil, fmt.Errorf("Bundle.GetBundleByHash error: %w, bundle hash: %v", err, bundleHash)
	}
	return &bundle, nil
}

// GetUnassignedAndBatchesUnreadyBundles get the bundles which is unassigned and batches are not ready
func (o *Bundle) GetUnassignedAndBatchesUnreadyBundles(ctx context.Context, offset, limit int) ([]*Bundle, error) {
	if offset < 0 || limit < 0 {
		return nil, errors.New("limit and offset must not be smaller than 0")
	}

	db := o.db.WithContext(ctx)
	db = db.Where("proving_status = ?", types.ProvingTaskUnassigned)
	db = db.Where("batch_proofs_status = ?", types.BatchProofsStatusPending)
	db = db.Order("index ASC")
	db = db.Offset(offset)
	db = db.Limit(limit)

	var bundles []*Bundle
	if err := db.Find(&bundles).Error; err != nil {
		return nil, fmt.Errorf("Bundle.GetUnassignedAndBatchesUnreadyBundles error: %w", err)
	}
	return bundles, nil
}

// UpdateBatchProofsStatusByBundleHash updates the status of batch_proofs_status field for a given bundle hash.
func (o *Bundle) UpdateBatchProofsStatusByBundleHash(ctx context.Context, bundleHash string, status types.BatchProofsStatus) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash = ?", bundleHash)

	if err := db.Update("batch_proofs_status", status).Error; err != nil {
		return fmt.Errorf("Bundle.UpdateBatchProofsStatusByBundleHash error: %w, bundle hash: %v, status: %v", err, bundleHash, status.String())
	}
	return nil
}

// UpdateProvingStatusFailed updates the proving status failed of a bundle.
func (o *Bundle) UpdateProvingStatusFailed(ctx context.Context, bundleHash string, maxAttempts uint8, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash", bundleHash)
	db = db.Where("total_attempts >= ?", maxAttempts)
	db = db.Where("proving_status != ?", int(types.ProvingTaskVerified))
	if err := db.Update("proving_status", int(types.ProvingTaskFailed)).Error; err != nil {
		return fmt.Errorf("Bundle.UpdateProvingStatus error: %w, bundle hash: %v, status: %v", err, bundleHash, types.ProvingTaskFailed.String())
	}
	return nil
}

// UpdateProofAndProvingStatusByHash updates the bundle proof and proving status by hash.
func (o *Bundle) UpdateProofAndProvingStatusByHash(ctx context.Context, hash string, proof []byte, provingStatus types.ProvingStatus, proofTimeSec uint64, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}

	updateFields := make(map[string]interface{})
	updateFields["proof"] = proof
	updateFields["proving_status"] = provingStatus
	updateFields["proof_time_sec"] = proofTimeSec
	updateFields["proved_at"] = utils.NowUTC()

	db = db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash", hash)

	if err := db.Updates(updateFields).Error; err != nil {
		return fmt.Errorf("Batch.UpdateProofByHash error: %w, batch hash: %v", err, hash)
	}
	return nil
}

// UpdateBundleAttempts atomically increments the attempts count for the earliest available bundle that meets the conditions.
func (o *Bundle) UpdateBundleAttempts(ctx context.Context, hash string, curActiveAttempts, curTotalAttempts int16) (int64, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash = ?", hash)
	db = db.Where("active_attempts = ?", curActiveAttempts)
	db = db.Where("total_attempts = ?", curTotalAttempts)
	result := db.Updates(map[string]interface{}{
		"proving_status":  types.ProvingTaskAssigned,
		"total_attempts":  gorm.Expr("total_attempts + 1"),
		"active_attempts": gorm.Expr("active_attempts + 1"),
	})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to update bundle, err:%w", result.Error)
	}
	return result.RowsAffected, nil
}

// DecreaseActiveAttemptsByHash decrements the active_attempts of a bundle given its hash.
func (o *Bundle) DecreaseActiveAttemptsByHash(ctx context.Context, bundleHash string, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&Bundle{})
	db = db.Where("hash = ?", bundleHash)
	db = db.Where("proving_status != ?", int(types.ProvingTaskVerified))
	db = db.Where("active_attempts > ?", 0)
	result := db.UpdateColumn("active_attempts", gorm.Expr("active_attempts - 1"))
	if result.Error != nil {
		return fmt.Errorf("Bundle.DecreaseActiveAttemptsByHash error: %w, bundle hash: %v", result.Error, bundleHash)
	}
	if result.RowsAffected == 0 {
		log.Warn("No rows were affected in DecreaseActiveAttemptsByHash", "bundle hash", bundleHash)
	}
	return nil
}
