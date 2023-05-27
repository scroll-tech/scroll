package orm

import (
	"time"

	"gorm.io/gorm"

	"scroll-tech/common/types"
)

type AggTask struct {
	db *gorm.DB `gorm:"-"`

	ID              string    `json:"id" gorm:"id"`
	StartBatchIndex uint64    `json:"start_batch_index" gorm:"start_batch_index"`
	StartBatchHash  string    `json:"start_batch_hash" gorm:"start_batch_hash"`
	EndBatchIndex   uint64    `json:"end_batch_index" gorm:"end_batch_index"`
	EndBatchHash    string    `json:"end_batch_hash" gorm:"end_batch_hash"`
	ProvingStatus   int       `json:"proving_status" gorm:"proving_status;default:1"`
	Proof           []byte    `json:"proof" gorm:"proof;default:NULL"`
	CreatedTime     time.Time `json:"created_time" gorm:"created_time;default:CURRENT_TIMESTAMP()"`
	UpdatedTime     time.Time `json:"updated_time" gorm:"updated_time;default:CURRENT_TIMESTAMP()"`
}

// NewAggTask creates an AggTaskOrm instance
func NewAggTask(db *gorm.DB) *AggTask {
	return &AggTask{db: db}
}

// TableName define the AggTaskOrm table name
func (*AggTask) TableName() string {
	return "agg_task"
}

// GetSubProofsByAggTaskID get sub proof by agg task id
func (a *AggTask) GetSubProofsByAggTaskID(id string) ([][]byte, error) {
	var aggTask AggTask
	err := a.db.Select("start_batch_index, end_batch_index").Where("id", id).First(&aggTask).Error
	if err != nil {
		return nil, err
	}

	var aggTaskList []AggTask
	err = a.db.Select("proof").Where("index >= ?", aggTask.StartBatchIndex).Where("index <= ?", aggTask.EndBatchIndex).
		Where("proving_status", types.ProvingTaskVerified).Find(&aggTaskList).Error
	if err != nil {
		return nil, err
	}

	var tmpProof [][]byte
	for _, v := range aggTaskList {
		tmpProof = append(tmpProof, v.Proof)
	}
	return tmpProof, nil
}

// GetUnassignedAggTasks get the agg task which proving_status is ProvingTaskUnassigned
func (a *AggTask) GetUnassignedAggTasks() ([]AggTask, error) {
	var aggTaskList []AggTask
	err := a.db.Where("proving_status", types.ProvingTaskUnassigned).Find(&aggTaskList).Error
	if err != nil {
		return nil, err
	}
	return aggTaskList, nil
}

// GetAssignedAggTasks get the agg task which proving_status is types.ProvingTaskAssigned or types.ProvingTaskProved
func (a *AggTask) GetAssignedAggTasks() ([]AggTask, error) {
	var aggTaskList []AggTask
	err := a.db.Where("proving_status IN (?)", []types.ProvingStatus{types.ProvingTaskAssigned, types.ProvingTaskProved}).Find(&aggTaskList).Error
	if err != nil {
		return nil, err
	}
	return aggTaskList, nil
}

// UpdateAggTaskStatus update agg task status
func (a *AggTask) UpdateAggTaskStatus(aggTaskID string, status types.ProvingStatus) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = status
	err := a.db.Model(&AggTask{}).Where("id", aggTaskID).Updates(updateFields).Error
	return err
}

// UpdateProofForAggTask update agg task proof
func (a *AggTask) UpdateProofForAggTask(aggTaskID string, proof []byte) error {
	updateFields := make(map[string]interface{})
	updateFields["proving_status"] = types.ProvingTaskProved
	updateFields["proof"] = proof
	err := a.db.Model(&AggTask{}).Where("id", aggTaskID).Updates(updateFields).Error
	return err
}
