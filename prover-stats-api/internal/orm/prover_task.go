package orm

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"scroll-tech/common/types/message"
	"time"

	"gorm.io/gorm/clause"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
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

// GetProverTasksByProver get all prover tasks by the given prover's public key.
func (o *ProverTask) GetProverTasksByProver(ctx context.Context, pubkey string, offset, limit int) ([]*ProverTask, error) {
	var proverTasks []*ProverTask
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("prover_public_key", pubkey)
	db = db.Order("id desc")
	db = db.Offset(offset)
	db = db.Limit(limit)
	if err := db.Find(&proverTasks).Error; err != nil {
		return nil, fmt.Errorf("ProverTask.GetProverTasksByProver error: %w, prover %s", err, pubkey)
	}
	return proverTasks, nil
}

// GetProverTotalReward get prover all reward by the given prover's public key.
func (o *ProverTask) GetProverTotalReward(ctx context.Context, pubkey string) (*big.Int, error) {
	var totalReward decimal.Decimal
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Select("sum(reward)")
	db = db.Where("prover_public_key", pubkey)
	if err := db.Scan(&totalReward).Error; err != nil {
		return nil, fmt.Errorf("ProverTask.GetProverTotalReward error:%w, prover:%s", err, pubkey)
	}
	return totalReward.BigInt(), nil
}

// GetProverTasksByHash retrieves the ProverTask records associated with the specified hashes.
// The returned prover task objects are sorted in ascending order by their ids.
func (o *ProverTask) GetProverTasksByHash(ctx context.Context, hash string) (*ProverTask, error) {
	db := o.db.WithContext(ctx)
	db = db.Model(&ProverTask{})
	db = db.Where("task_id", hash)
	db = db.Order("id asc")

	var proverTask *ProverTask
	if err := db.Find(&proverTask).Error; err != nil {
		return nil, fmt.Errorf("ProverTask.GetProverTasksByHash error: %w, hash: %v", err, hash)
	}
	return proverTask, nil
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
