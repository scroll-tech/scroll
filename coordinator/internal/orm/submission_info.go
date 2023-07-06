package orm

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// SubmissionInfo is assigned rollers info of chunk/batch proof submission
type SubmissionInfo struct {
	db *gorm.DB `gorm:"column:-"`

	ID              int64          `json:"id" gorm:"column:id"`
	TaskID          string         `json:"task_id" gorm:"column:task_id"`
	RollerPublicKey string         `json:"roller_public_key" gorm:"column:roller_public_key"`
	RollerName      string         `json:"roller_name" gorm:"column:roller_name"`
	ProofType       int16          `json:"proof_type" gorm:"column:proof_type;default:0"`
	ProvingStatus   int16          `json:"proving_status" gorm:"column:proving_status;default:0"`
	FailureType     int16          `json:"failure_type" gorm:"column:failure_type;default:0"`
	Reward          uint64         `json:"reward" gorm:"column:reward;default:0"`
	Proof           []byte         `json:"proof" gorm:"column:proof;default:NULL"`
	CreatedAt       time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at"`
}

// NewSubmissionInfo creates a new SubmissionInfo instance.
func NewSubmissionInfo(db *gorm.DB) *SubmissionInfo {
	return &SubmissionInfo{db: db}
}

// TableName returns the name of the "submission_info" table.
func (*SubmissionInfo) TableName() string {
	return "submission_info"
}

// GetSubmissionInfosByHashes retrieves the SubmissionInfo records associated with the specified hashes.
// The returned session info objects are sorted in ascending order by their ids.
func (o *SubmissionInfo) GetSubmissionInfosByHashes(ctx context.Context, hashes []string) ([]*SubmissionInfo, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	var sessionInfos []*SubmissionInfo
	db := o.db.WithContext(ctx)
	db = db.Where("task_id IN ?", hashes)
	db = db.Order("id asc")

	if err := db.Find(&sessionInfos).Error; err != nil {
		return nil, err
	}
	return sessionInfos, nil
}

// SetSubmissionInfo updates or inserts a SubmissionInfo record.
func (o *SubmissionInfo) SetSubmissionInfo(ctx context.Context, sessionInfo *SubmissionInfo) error {
	db := o.db.WithContext(ctx)
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_id"}, {Name: "roller_public_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"proving_status"}),
	})
	return db.Create(&sessionInfo).Error
}

// UpdateSubmissionInfoProvingStatus updates the proving_status of a specific SubmissionInfo record.
func (o *SubmissionInfo) UpdateSubmissionInfoProvingStatus(ctx context.Context, proofType message.ProofType, taskID string, pk string, status types.RollerProveStatus) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&SubmissionInfo{})
	db = db.Where("proof_type = ? AND task_id = ? AND roller_public_key = ?", proofType, taskID, pk)

	return db.Update("proving_status", status).Error
}
