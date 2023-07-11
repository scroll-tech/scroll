package orm

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/shopspring/decimal"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

// ProverTask is assigned rollers info of chunk/batch proof prover task
type ProverTask struct {
	db *gorm.DB `gorm:"column:-"`

	ID              int64           `json:"id" gorm:"column:id"`
	TaskID          string          `json:"task_id" gorm:"column:task_id"`
	ProverPublicKey string          `json:"prover_public_key" gorm:"column:prover_public_key"`
	ProverName      string          `json:"prover_name" gorm:"column:prover_name"`
	TaskType        int16           `json:"task_type" gorm:"column:task_type;default:0"`
	ProvingStatus   int16           `json:"proving_status" gorm:"column:proving_status;default:0"`
	FailureType     int16           `json:"failure_type" gorm:"column:failure_type;default:0"`
	Reward          decimal.Decimal `json:"reward" gorm:"column:reward;default:0;type:decimal(78)"`
	Proof           []byte          `json:"proof" gorm:"column:proof;default:NULL"`
	CreatedAt       time.Time       `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time       `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt       gorm.DeletedAt  `json:"deleted_at" gorm:"column:deleted_at"`
}

// NewProverTask creates a new ProverTask instance.
func NewProverTask(db *gorm.DB) *ProverTask {
	return &ProverTask{db: db}
}

// TableName returns the name of the "prover_task" table.
func (*ProverTask) TableName() string {
	return "prover_task"
}

// GetProverTasksByHashes retrieves the ProverTask records associated with the specified hashes.
// The returned prover task objects are sorted in ascending order by their ids.
func (o *ProverTask) GetProverTasksByHashes(ctx context.Context, hashes []string) ([]*ProverTask, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("task_id IN ?", hashes)
	db = db.Order("id asc")

	var proverTasks []*ProverTask
	if err := db.Find(&proverTasks).Error; err != nil {
		return nil, fmt.Errorf("ProverTask.GetProverTasksByHashes error: %w, hashes: %v", err, hashes)
	}
	return proverTasks, nil
}

// GetProverTasksByProver returns prover-tasks by the given prover's public key.
func (o *ProverTask) GetProverTasksByProver(ctx context.Context, pubkey string) ([]*ProverTask, error) {
	var proverTasks []*ProverTask
	err := o.db.WithContext(ctx).Model(&ProverTask{}).Where(&ProverTask{ProverPublicKey: pubkey}).Order("id asc").Find(&proverTasks).Error
	if err != nil {
		return nil, fmt.Errorf("ProverTask.GetProverTasksByProver error: %w, prover %s", err, pubkey)
	}
	return proverTasks, nil
}

// SetProverTask updates or inserts a ProverTask record.
func (o *ProverTask) SetProverTask(ctx context.Context, proverTask *ProverTask) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "task_type"}, {Name: "task_id"}, {Name: "prover_public_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"proving_status"}),
	})

	if err := db.Create(&proverTask).Error; err != nil {
		return fmt.Errorf("ProverTask.SetProverTask error: %w, prover task: %v", err, proverTask)
	}
	return nil
}

// UpdateProverTaskProvingStatus updates the proving_status of a specific ProverTask record.
func (o *ProverTask) UpdateProverTaskProvingStatus(ctx context.Context, proofType message.ProofType, taskID string, pk string, status types.RollerProveStatus) error {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("task_type = ? AND task_id = ? AND prover_public_key = ?", proofType, taskID, pk)

	if err := db.Update("proving_status", status).Error; err != nil {
		return fmt.Errorf("ProverTask.UpdateProverTaskProvingStatus error: %w, proof type: %v, taskID: %v, prover public key: %v, status: %v", err, proofType.String(), taskID, pk, status.String())
	}
	return nil
}
