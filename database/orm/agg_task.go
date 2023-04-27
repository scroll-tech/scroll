package orm

import (
	"encoding/json"
	"github.com/jmoiron/sqlx"

	"scroll-tech/common/message"
	"scroll-tech/common/types"
)

type aggTaskOrm struct {
	db *sqlx.DB
}

var _ AggTaskOrm = (*aggTaskOrm)(nil)

// NewAggTaskOrm creates an AggTaskOrm instance
func NewAggTaskOrm(db *sqlx.DB) AggTaskOrm {
	return &aggTaskOrm{db: db}
}

func (a *aggTaskOrm) GetSubProofsByAggTaskID(id string) ([][]byte, error) {
	row := a.db.QueryRowx("SELECT * FROM agg_task where id = $1", id)
	aggTask := new(types.AggTask)
	err := row.StructScan(aggTask)
	if err != nil {
		return nil, err
	}
	rows, err := a.db.Queryx("SELECT proof FROM block_batch WHERE index IN ($1, $2) and proving_status = $3", aggTask.StartBatchIndex, aggTask.EndBatchIndex, types.ProvingTaskVerified)
	if err != nil {
		return nil, err
	}
	var subProofs [][]byte
	for rows.Next() {
		var proofByt []byte
		err = rows.Scan(&proofByt)
		if err != nil {
			return nil, err
		}
		subProofs = append(subProofs, proofByt)
	}
	return subProofs, nil
}

func (a *aggTaskOrm) GetUnassignedAggTasks() ([]*types.AggTask, error) {
	rows, err := a.db.Queryx("SELECT * FROM agg_task where proving_status = 1;")
	if err != nil {
		return nil, err
	}
	return a.rowsToAggTask(rows)
}

func (a *aggTaskOrm) GetAssignedAggTasks() ([]*types.AggTask, error) {
	rows, err := a.db.Queryx(`SELECT * FROM agg_task WHERE proving_status IN ($1, $2)`, types.ProvingTaskAssigned, types.ProvingTaskProved)
	if err != nil {
		return nil, err
	}
	return a.rowsToAggTask(rows)
}

func (a *aggTaskOrm) InsertAggTask(task *types.AggTask) error {
	sqlStr := "INSERT INTO agg_task (id, start_batch_index, start_batch_hash, end_batch_index, end_batch_hash) VALUES ($1, $2, $3, $4, $5)"
	_, err := a.db.Exec(sqlStr, task.ID, task.StartBatchIndex, task.StartBatchHash, task.EndBatchIndex, task.EndBatchHash)
	return err
}

func (a *aggTaskOrm) UpdateAggTaskStatus(aggTaskID string, status types.ProvingStatus) error {
	_, err := a.db.Exec(a.db.Rebind("update agg_task set proving_status = ? where id = ?;"), status, aggTaskID)
	return err
}

func (a *aggTaskOrm) UpdateProofForAggTask(aggTaskID string, proof *message.AggProof) error {
	proofByt, err := json.Marshal(proof)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(a.db.Rebind("update agg_task set proving_status = ?, proof = ? where id = ?;"), types.ProvingTaskProved, proofByt, aggTaskID)
	return err
}

func (a *aggTaskOrm) rowsToAggTask(rows *sqlx.Rows) ([]*types.AggTask, error) {
	var tasks []*types.AggTask
	for rows.Next() {
		task := new(types.AggTask)
		err := rows.StructScan(task)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
