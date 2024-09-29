package orm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
	"scroll-tech/common/utils"
)

// ProverTask is assigned provers info of chunk/batch proof prover task
type ProverTask struct {
	db *gorm.DB `gorm:"column:-"`

	ID   int64     `json:"id" gorm:"column:id"`
	UUID uuid.UUID `json:"uuid" gorm:"column:uuid;type:uuid;default:gen_random_uuid()"`

	// prover
	ProverPublicKey string `json:"prover_public_key" gorm:"column:prover_public_key"`
	ProverName      string `json:"prover_name" gorm:"column:prover_name"`
	ProverVersion   string `json:"prover_version" gorm:"column:prover_version"`

	// task
	TaskID   string `json:"task_id" gorm:"column:task_id"`
	TaskType int16  `json:"task_type" gorm:"column:task_type;default:0"`

	// status
	ProvingStatus int16           `json:"proving_status" gorm:"column:proving_status;default:0"`
	FailureType   int16           `json:"failure_type" gorm:"column:failure_type;default:0"`
	Reward        decimal.Decimal `json:"reward" gorm:"column:reward;default:0;type:decimal(78)"`
	Proof         []byte          `json:"proof" gorm:"column:proof;default:NULL"`
	AssignedAt    time.Time       `json:"assigned_at" gorm:"assigned_at"`

	// metadata
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at"`
}

// NewProverTask creates a new ProverTask instance.
func NewProverTask(db *gorm.DB) *ProverTask {
	return &ProverTask{db: db}
}

// TableName returns the name of the "prover_task" table.
func (*ProverTask) TableName() string {
	return "prover_task"
}

// IsProverAssigned checks if a prover with the given public key has been assigned a task.
func (o *ProverTask) IsProverAssigned(ctx context.Context, publicKey string) (bool, error) {
	db := o.db.WithContext(ctx)
	var task ProverTask
	err := db.Where("prover_public_key = ? AND proving_status = ?", publicKey, types.ProverAssigned).First(&task).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetProverTasks get prover tasks
func (o *ProverTask) GetProverTasks(ctx context.Context, fields map[string]interface{}, orderByList []string, offset, limit int) ([]ProverTask, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})

	for k, v := range fields {
		db = db.Where(k, v)
	}

	for _, orderBy := range orderByList {
		db = db.Order(orderBy)
	}

	if limit != 0 {
		db = db.Limit(limit)
	}

	if offset != 0 {
		db = db.Offset(offset)
	}

	var proverTasks []ProverTask
	if err := db.Find(&proverTasks).Error; err != nil {
		return nil, err
	}
	return proverTasks, nil
}

// GetProverTasksByHashes retrieves the ProverTask records associated with the specified hashes.
// The returned prover task objects are sorted in ascending order by their ids.
func (o *ProverTask) GetProverTasksByHashes(ctx context.Context, taskType message.ProofType, hashes []string) ([]*ProverTask, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("task_type", int(taskType))
	db = db.Where("task_id IN ?", hashes)
	db = db.Order("id asc")

	var proverTasks []*ProverTask
	if err := db.Find(&proverTasks).Error; err != nil {
		return nil, fmt.Errorf("ProverTask.GetProverTasksByHashes error: %w, hashes: %v", err, hashes)
	}
	return proverTasks, nil
}

// GetProverTaskByUUIDAndPublicKey get prover task taskID by uuid and public key
func (o *ProverTask) GetProverTaskByUUIDAndPublicKey(ctx context.Context, uuid, publicKey string) (*ProverTask, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("uuid", uuid)
	db = db.Where("prover_public_key", publicKey)

	var proverTask ProverTask
	err := db.First(&proverTask).Error
	if err != nil {
		return nil, fmt.Errorf("ProverTask.GetProverTaskByUUID err:%w, uuid:%s publicKey:%s", err, uuid, publicKey)
	}
	return &proverTask, nil
}

// GetAssignedTaskOfOtherProvers get the chunk/batch task assigned other provers
func (o *ProverTask) GetAssignedTaskOfOtherProvers(ctx context.Context, taskType message.ProofType, taskID, proverPublicKey string) ([]ProverTask, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("task_type", int(taskType))
	db = db.Where("task_id", taskID)
	db = db.Where("prover_public_key != ?", proverPublicKey)
	db = db.Where("proving_status = ?", int(types.ProverAssigned))

	var proverTasks []ProverTask
	if err := db.Find(&proverTasks).Error; err != nil {
		return nil, fmt.Errorf("ProverTask.GetAssignedProverTask error: %w, taskID: %v", err, taskID)
	}
	return proverTasks, nil
}

// GetProvingStatusByTaskID retrieves the proving status of a prover task
func (o *ProverTask) GetProvingStatusByTaskID(ctx context.Context, taskType message.ProofType, taskID string) (types.ProverProveStatus, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Select("proving_status")
	db = db.Where("task_type", int(taskType))
	db = db.Where("task_id = ?", taskID)

	var proverTask ProverTask
	if err := db.Find(&proverTask).Error; err != nil {
		return types.ProverProofInvalid, fmt.Errorf("ProverTask.GetProvingStatusByTaskID error: %w, taskID: %v", err, taskID)
	}
	return types.ProverProveStatus(proverTask.ProvingStatus), nil
}

// GetTimeoutAssignedProverTasks get the timeout and assigned proving_status prover task
func (o *ProverTask) GetTimeoutAssignedProverTasks(ctx context.Context, limit int, taskType message.ProofType, timeout time.Duration) ([]ProverTask, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("proving_status", int(types.ProverAssigned))
	db = db.Where("task_type", int(taskType))
	db = db.Where("assigned_at < ?", utils.NowUTC().Add(-timeout))
	db = db.Limit(limit)

	var proverTasks []ProverTask
	err := db.Find(&proverTasks).Error
	if err != nil {
		return nil, fmt.Errorf("ProverTask.GetAssignedProverTasks error:%w", err)
	}
	return proverTasks, nil
}

// TaskTimeoutMoreThanOnce get the timeout twice task. a temp design
func (o *ProverTask) TaskTimeoutMoreThanOnce(ctx context.Context, taskType message.ProofType, taskID string) bool {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("task_type", int(taskType))
	db = db.Where("task_id", taskID)
	db = db.Where("proving_status", int(types.ProverProofInvalid))

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return true
	}

	if count >= 1 {
		return true
	}

	return false
}

// InsertProverTask insert a prover Task record
func (o *ProverTask) InsertProverTask(ctx context.Context, proverTask *ProverTask, dbTX ...*gorm.DB) error {
	db := o.db.WithContext(ctx)
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.Clauses(clause.Returning{})
	db = db.Model(&ProverTask{})
	if err := db.Create(proverTask).Error; err != nil {
		return fmt.Errorf("ProverTask.InsertProverTask error: %w, prover task: %v", err, proverTask)
	}
	return nil
}

// UpdateProverTaskProof update the prover task's proof
func (o *ProverTask) UpdateProverTaskProof(ctx context.Context, uuid uuid.UUID, proof []byte) error {
	db := o.db
	db = db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("uuid = ?", uuid)
	if err := db.Update("proof", proof).Error; err != nil {
		return fmt.Errorf("ProverTask.UpdateProverTaskProof error: %w, uuid: %v", err, uuid)
	}
	return nil
}

// UpdateProverTaskProvingStatusAndFailureType updates the proving_status of a specific ProverTask record.
func (o *ProverTask) UpdateProverTaskProvingStatusAndFailureType(ctx context.Context, uuid uuid.UUID, status types.ProverProveStatus, failureType types.ProverTaskFailureType, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("uuid = ?", uuid)

	updates := make(map[string]interface{})
	updates["proving_status"] = int(status)
	if status == types.ProverProofInvalid {
		updates["failure_type"] = int(failureType)
	}
	if err := db.Updates(updates).Error; err != nil {
		return fmt.Errorf("ProverTask.UpdateProverTaskProvingStatus error: %w, uuid:%s, status: %v", err, uuid, status.String())
	}
	return nil
}

// UpdateProverTaskFailureType update the prover task failure type
func (o *ProverTask) UpdateProverTaskFailureType(ctx context.Context, uuid uuid.UUID, failureType types.ProverTaskFailureType, dbTX ...*gorm.DB) error {
	db := o.db
	if len(dbTX) > 0 && dbTX[0] != nil {
		db = dbTX[0]
	}
	db = db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("uuid", uuid)
	if err := db.Update("failure_type", int(failureType)).Error; err != nil {
		return fmt.Errorf("ProverTask.UpdateProverTaskFailureType error: %w, uuid:%s, failure type: %v", err, uuid.String(), failureType.String())
	}
	return nil
}
