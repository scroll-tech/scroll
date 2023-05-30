package orm

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

type aggTaskOrm struct {
	db *sqlx.DB
}

var _ AggTaskOrm = (*aggTaskOrm)(nil)

// NewAggTaskOrm creates an AggTaskOrm instance
func NewAggTaskOrm(db *sqlx.DB) AggTaskOrm {
	return &aggTaskOrm{db: db}
}

func (a *aggTaskOrm) GetSubProofsByAggTaskID(id string) ([]*message.AggProof, error) {
	var (
		startIdx uint64
		endIdx   uint64
	)
	row := a.db.QueryRow("SELECT start_batch_index, end_batch_index FROM agg_task where id = $1", id)
	err := row.Scan(&startIdx, &endIdx)
	if err != nil {
		return nil, err
	}
	rows, err := a.db.Queryx("SELECT proof FROM block_batch WHERE index>=$1 AND index<=$2 and proving_status = $3", startIdx, endIdx, types.ProvingTaskVerified)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var subProofs []*message.AggProof
	for rows.Next() {
		var proofByt []byte
		err = rows.Scan(&proofByt)
		if err != nil {
			return nil, err
		}

		var proof message.AggProof
		if err := json.Unmarshal(proofByt, &proof); err != nil {
			return nil, err
		}

		subProofs = append(subProofs, &proof)
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

func (a *aggTaskOrm) InsertAggTask(id string, startBatchIndex uint64, startBatchHash string, endBatchIndex uint64, endBatchHash string) error {
	sqlStr := "INSERT INTO agg_task (id, start_batch_index, start_batch_hash, end_batch_index, end_batch_hash) VALUES ($1, $2, $3, $4, $5)"
	_, err := a.db.Exec(sqlStr, id, startBatchIndex, startBatchHash, endBatchIndex, endBatchHash)
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
