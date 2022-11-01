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

// ProvingStatus block_batch proving_status (unassigned, assigned, proved, verified, submitted)
type ProvingStatus int

const (
	// ProvingStatusUndefined : undefined proving_task status
	ProvingStatusUndefined ProvingStatus = iota
	// ProvingTaskUnassigned : proving_task is not assigned to be proved
	ProvingTaskUnassigned
	// ProvingTaskSkipped : proving_task is skipped for proof generation
	ProvingTaskSkipped
	// ProvingTaskAssigned : proving_task is assigned to be proved
	ProvingTaskAssigned
	// ProvingTaskProved : proof has been returned by prover
	ProvingTaskProved
	// ProvingTaskVerified : proof is valid
	ProvingTaskVerified
	// ProvingTaskFailed : fail to generate proof
	ProvingTaskFailed
)

func (ps ProvingStatus) String() string {
	switch ps {
	case ProvingTaskUnassigned:
		return "unassigned"
	case ProvingTaskSkipped:
		return "skipped"
	case ProvingTaskAssigned:
		return "assigned"
	case ProvingTaskProved:
		return "proved"
	case ProvingTaskVerified:
		return "verified"
	case ProvingTaskFailed:
		return "failed"
	default:
		return "undefined"
	}
}

// RollupStatus block_batch rollup_status (pending, committing, committed, finalizing, finalized)
type RollupStatus int

const (
	// RollupUndefined : undefined rollup status
	RollupUndefined RollupStatus = iota
	// RollupPending : batch is pending to rollup to layer1
	RollupPending
	// RollupCommitting : rollup transaction is submitted to layer1
	RollupCommitting
	// RollupCommitted : rollup transaction is confirmed to layer1
	RollupCommitted
	// RollupFinalizing : finalize transaction is submitted to layer1
	RollupFinalizing
	// RollupFinalized : finalize transaction is confirmed to layer1
	RollupFinalized
	// RollupFinalizationSkipped : batch finalization is skipped
	RollupFinalizationSkipped
)

type blockBatchOrm struct {
	db *sqlx.DB
}

var _ BlockBatchOrm = (*blockBatchOrm)(nil)

// NewBlockBatchOrm create an blockBatchOrm instance
func NewBlockBatchOrm(db *sqlx.DB) BlockBatchOrm {
	return &blockBatchOrm{db: db}
}

func (o *blockBatchOrm) GetProveTasks(fields map[string]interface{}, args ...string) ([]*ProveTask, error) {
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

func (o *blockBatchOrm) GetTaskStatusByID(id uint64) (TaskStatus, error) {
	row := o.db.QueryRow(`SELECT status FROM prove_task WHERE id = $1`, id)
	var status TaskStatus
	if err := row.Scan(&status); err != nil {
		return TaskUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) GetVerifiedProofAndInstanceByID(id uint64) ([]byte, []byte, error) {
	var proof []byte
	var instance []byte
	row := o.db.QueryRow(`SELECT proof, instance_commitments FROM prove_task WHERE id = $1 and status = $2`, id, TaskVerified)

	if err := row.Scan(&proof, &instance); err != nil {
		return nil, nil, err
	}
	return proof, instance, nil
}

func (o *blockBatchOrm) UpdateProofByID(ctx context.Context, id uint64, proof, instance_commitments []byte, proofTimeSec uint64) error {
	db := o.db
	if _, err := db.ExecContext(ctx, db.Rebind(`update prove_task set proof = ?, instance_commitments = ?, proof_time_sec = ? where id = ?;`), proof, instance_commitments, proofTimeSec, id); err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return nil
}

func (o *blockBatchOrm) UpdateTaskStatus(id uint64, status TaskStatus) error {
	if _, err := o.db.Exec(o.db.Rebind("update prove_task set status = ? where id = ?;"), status, id); err != nil {
		return err
	}
	return nil
}

func (o *blockBatchOrm) NewBatchInDBTx(dbTx *sqlx.Tx, total_l2_gas uint64) (uint64, error) {
	row := dbTx.QueryRow("SELECT MAX(id) FROM prove_task;")

	var id int64 // 0 by default for sql.ErrNoRows
	if err := row.Scan(&id); err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	id += 1
	if _, err := dbTx.NamedExec(`INSERT INTO public.prove_task (id, total_l2_gas) VALUES (:id, :total_l2_gas)`,
		map[string]interface{}{
			"id":           id,
			"total_l2_gas": total_l2_gas,
		}); err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func (o *blockBatchOrm) RollupRecordExist(id uint64) (bool, error) {
	var res int
	return res == 1, o.db.Get(&res, o.db.Rebind(`SELECT 1 from rollup_result where id = ? limit 1;`), id)
}

func (o *blockBatchOrm) GetPendingBatches() ([]uint64, error) {
	rows, err := o.db.Queryx(`SELECT id FROM rollup_result WHERE status = $1 ORDER BY id ASC`, RollupPending)
	if err != nil {
		return nil, err
	}

	var ids []uint64
	for rows.Next() {
		var id uint64
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no pending batches in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return ids, rows.Close()
}

func (o *blockBatchOrm) GetLatestFinalizedBatch() (uint64, error) {
	row := o.db.QueryRow(`SELECT MAX(id) FROM rollup_result WHERE status = $1;`, RollupFinalized)
	var id uint64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (o *blockBatchOrm) GetCommittedBatches() ([]uint64, error) {
	rows, err := o.db.Queryx(`SELECT id FROM rollup_result WHERE status = $1 ORDER BY id ASC`, RollupCommitted)
	if err != nil {
		return nil, err
	}

	var ids []uint64
	for rows.Next() {
		var id uint64
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no committed batches in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return ids, rows.Close()
}

func (o *blockBatchOrm) GetRollupStatus(id uint64) (RollupStatus, error) {
	row := o.db.QueryRow(`SELECT status FROM rollup_result WHERE id = $1`, id)
	var status RollupStatus
	if err := row.Scan(&status); err != nil {
		return RollupUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) InsertPendingBatches(ctx context.Context, batches []uint64) error {
	batchMaps := make([]map[string]interface{}, len(batches))
	for i, id := range batches {
		batchMaps[i] = map[string]interface{}{
			"id": id,
		}
	}

	_, err := o.db.NamedExec(`INSERT INTO public.rollup_result (id) VALUES (:id);`, batchMaps)
	if err != nil {
		log.Error("failed to insert rollupResults", "err", err)
	}
	return err
}

func (o *blockBatchOrm) UpdateRollupStatus(ctx context.Context, id uint64, status RollupStatus) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set status = ? where id = ?;"), status, id); err != nil {
		return err
	}
	return nil
}

func (o *blockBatchOrm) UpdateRollupTxHash(ctx context.Context, id uint64, rollup_tx_hash string) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set rollup_tx_hash = ? where id = ?;"), rollup_tx_hash, id); err != nil {
		return err
	}
	return nil
}

func (o *blockBatchOrm) UpdateFinalizeTxHash(ctx context.Context, id uint64, finalize_tx_hash string) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set finalize_tx_hash = ? where id = ?;"), finalize_tx_hash, id); err != nil {
		return err
	}
	return nil
}

func (o *blockBatchOrm) UpdateRollupTxHashAndStatus(ctx context.Context, id uint64, rollup_tx_hash string, status RollupStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update rollup_result set rollup_tx_hash = ?, status = ? where id = ?;"), rollup_tx_hash, status, id)
	return err
}

func (o *blockBatchOrm) UpdateFinalizeTxHashAndStatus(ctx context.Context, id uint64, finalize_tx_hash string, status RollupStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update rollup_result set finalize_tx_hash = ?, status = ? where id = ?;"), finalize_tx_hash, status, id)
	return err
}
