package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
)

// TaskStatus proveTask status(unassigned, assigned, proved, verified, submitted)
type TaskStatus int

const (
	// TaskUndefined : undefined block status
	TaskUndefined TaskStatus = iota
	// TaskUnassigned : task is not assigned to be proved
	TaskUnassigned
	// TaskSkipped : task is skipped for proof generation
	TaskSkipped
	// TaskAssigned : task is assigned to be proved
	TaskAssigned
	// TaskProved : proof has been returned by prover
	TaskProved
	// TaskVerified : proof is valid
	TaskVerified
	// TaskFailed : fail to generate proof
	TaskFailed
)

func (ts TaskStatus) String() string {
	switch ts {
	case TaskUnassigned:
		return "unassigned"
	case TaskSkipped:
		return "skipped"
	case TaskAssigned:
		return "assigned"
	case TaskProved:
		return "proved"
	case TaskVerified:
		return "undefined"
	case TaskFailed:
		return "failed"
	default:
		return "undefined"
	}
}

type proveTaskOrm struct {
	db *sqlx.DB
}

var _ ProveTaskOrm = (*proveTaskOrm)(nil)

// NewProveTaskOrm create an proveTaskOrm instance
func NewProveTaskOrm(db *sqlx.DB) ProveTaskOrm {
	return &proveTaskOrm{db: db}
}

func (o *proveTaskOrm) GetProveTasks(fields map[string]interface{}, args ...string) ([]*ProveTask, error) {
	query := "SELECT id, proof, instance_commitments, status, proof_time_sec FROM prove_task WHERE 1 = 1 "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var tasks []*ProveTask
	for rows.Next() {
		task := &ProveTask{}
		if err = rows.StructScan(task); err != nil {
			break
		}
		tasks = append(tasks, task)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return tasks, rows.Close()
}

func (o *proveTaskOrm) GetTaskStatusByID(id uint64) (TaskStatus, error) {
	row := o.db.QueryRow(`SELECT status FROM prove_task WHERE id = $1`, id)
	var status TaskStatus
	if err := row.Scan(&status); err != nil {
		return TaskUndefined, err
	}
	return status, nil
}

func (o *proveTaskOrm) GetVerifiedProofAndInstanceByID(id uint64) ([]byte, []byte, error) {
	var proof []byte
	var instance []byte
	row := o.db.QueryRow(`SELECT proof, instance_commitments FROM prove_task WHERE id = $1 and status = $2`, id, TaskVerified)

	if err := row.Scan(&proof, &instance); err != nil {
		return nil, nil, err
	}
	return proof, instance, nil
}

func (o *proveTaskOrm) UpdateProofByID(ctx context.Context, id uint64, proof, instance_commitments []byte, proofTimeSec uint64) error {
	db := o.db
	if _, err := db.ExecContext(ctx, db.Rebind(`update prove_task set proof = ?, instance_commitments = ?, proof_time_sec = ? where id = ?;`), proof, instance_commitments, proofTimeSec, id); err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return nil
}

func (o *proveTaskOrm) UpdateTaskStatus(id uint64, status TaskStatus) error {
	if _, err := o.db.Exec(o.db.Rebind("update prove_task set status = ? where id = ?;"), status, id); err != nil {
		return err
	}
	return nil
}
