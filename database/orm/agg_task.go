package orm

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"

	"scroll-tech/common/message"
)

// AggTask is a wrapper type around db AggProveTask type.
type AggTask struct {
	ID     string              `json:"id"`
	Proofs []*message.AggProof `json:"proofs"`
}

type aggTaskOrm struct {
	db *sqlx.DB
}

var _ AggTaskOrm = (*aggTaskOrm)(nil)

// NewAggTaskOrm creates an AggTaskOrm instance
func NewAggTaskOrm(db *sqlx.DB) AggTaskOrm {
	return &aggTaskOrm{db: db}
}

func (a *aggTaskOrm) GetUnassignedAggTasks() ([]*AggTask, error) {
	rows, err := a.db.Queryx("SELECT task FROM agg_task where roller = null")
	if err != nil {
		return nil, err
	}
	var tasks []*AggTask
	for rows.Next() {
		var byt []byte
		err = rows.Scan(&byt)
		if err != nil {
			return nil, err
		}

		task := new(AggTask)
		err = json.Unmarshal(byt, task)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (a *aggTaskOrm) InsertAggTask(task *AggTask) error {
	byt, err := json.Marshal(task)
	if err != nil {
		return err
	}
	sqlStr := "INSERT INTO agg_task (hash, task) VALUES ($1, $2) ON CONFLICT (hash) DO UPDATE SET task = EXCLUDED.task;"
	_, err = a.db.Exec(sqlStr, task.ID, byt)
	return err
}

func (a *aggTaskOrm) UpdateProofForAggTask(aggTaskID, roller string, proof *message.AggProof) error {
	byt, err := json.Marshal(proof)
	if err != nil {
		return err
	}
	_, err = a.db.Exec(a.db.Rebind("update agg_task set roller = ?, proof = ? where hash = ?;"), roller, byt, aggTaskID)
	return err
}
