package orm

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// SessionInfo is assigned rollers info of a block batch (session).
type SessionInfo struct {
	db *gorm.DB `gorm:"column:-"`

	ID              int            `json:"id" gorm:"column:id"`
	TaskID          string         `json:"task_id" gorm:"column:task_id"`
	RollerPublicKey string         `json:"roller_public_key" gorm:"column:roller_public_key"`
	RollerName      string         `json:"roller_name" gorm:"column:roller_name"`
	ProveType       int16          `json:"prove_type" gorm:"column:prove_type;default:0"`
	ProvingStatus   int16          `json:"proving_status" gorm:"column:proving_status;default:0"`
	FailureType     int16          `json:"failure_type" gorm:"column:failure_type;default:0"`
	Reward          uint64         `json:"reward" gorm:"column:reward;default:0"`
	Proof           []byte         `json:"proof" gorm:"column:proof;default:NULL"`
	CreatedAt       time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at"`
}

// NewSessionInfo creates a new SessionInfo instance.
func NewSessionInfo(db *gorm.DB) *SessionInfo {
	return &SessionInfo{db: db}
}

// TableName returns the name of the "session_info" table.
func (*SessionInfo) TableName() string {
	return "session_info"
}

// GetSessionInfosByHashes retrieves the SessionInfo records associated with the specified hashes.
func (o *SessionInfo) GetSessionInfosByHashes(ctx context.Context, hashes []string) ([]*SessionInfo, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	var sessionInfos []*SessionInfo
	db := o.db.WithContext(ctx)
	if err := db.Where("task_id IN ?", hashes).Find(&sessionInfos).Error; err != nil {
		return nil, err
	}
	return sessionInfos, nil
}

// SetSessionInfo updates or inserts a SessionInfo record.
func (o *SessionInfo) SetSessionInfo(ctx context.Context, rollersInfo *SessionInfo) error {
	sessionInfo := SessionInfo{
		TaskID:          rollersInfo.TaskID,
		RollerPublicKey: rollersInfo.RollerPublicKey,
		ProveType:       rollersInfo.ProveType,
		RollerName:      rollersInfo.RollerName,
		ProvingStatus:   rollersInfo.ProvingStatus,
		FailureType:     rollersInfo.FailureType,
		Reward:          rollersInfo.Reward,
		Proof:           rollersInfo.Proof,
		CreatedAt:       rollersInfo.CreatedAt,
	}

	db := o.db.WithContext(ctx)
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_id"}, {Name: "roller_public_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"proving_status"}),
	}).Create(&sessionInfo).Error; err != nil {
		return err
	}
	return nil
}

// UpdateSessionInfoProvingStatus updates the proving_status of a specific SessionInfo record.
func (o *SessionInfo) UpdateSessionInfoProvingStatus(ctx context.Context, proveType message.ProveType, taskID string, pk string, status types.RollerProveStatus) error {
	db := o.db.WithContext(ctx)
	if err := db.Model(&SessionInfo{}).Where("prove_type = ? AND task_id = ? AND roller_public_key = ?", proveType, taskID, pk).Update("proving_status", status).Error; err != nil {
		return err
	}
	return nil
}
