package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

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

// BlockBatch is structure of stored block_batch
type BlockBatch struct {
	ID                  uint64        `json:"id" db:"id"`
	ParentHash          string        `json:"parent_hash" db:"parent_hash"` // TODO: TODO:
	TotalL2Gas          uint64        `json:"total_l2_gas" db:"total_l2_gas"`
	ProvingStatus       ProvingStatus `json:"proving_status" db:"proving_status"`
	Proof               []byte        `json:"proof" db:"proof"`
	InstanceCommitments []byte        `json:"instance_commitments" db:"instance_commitments"`
	ProofTimeSec        uint64        `json:"proof_time_sec" db:"proof_time_sec"`
	RollupStatus        RollupStatus  `json:"rollup_status" db:"rollup_status"`
	RollupTxHash        string        `json:"rollup_tx_hash" db:"rollup_tx_hash"`
	FinalizeTxHash      string        `json:"finalize_tx_hash" db:"finalize_tx_hash"`
	CreatedAt           time.Time     `json:"created_at" db:"created_at"`
	ProvedAt            time.Time     `json:"proved_at" db:"proved_at"`
	CommittedAt         time.Time     `json:"committed_at" db:"committed_at"`
	FinalizedAt         time.Time     `json:"finalized_at" db:"finalized_at"`
}

type blockBatchOrm struct {
	db *sqlx.DB
}

var _ BlockBatchOrm = (*blockBatchOrm)(nil)

// NewBlockBatchOrm create an blockBatchOrm instance
func NewBlockBatchOrm(db *sqlx.DB) BlockBatchOrm {
	return &blockBatchOrm{db: db}
}

func (o *blockBatchOrm) GetBlockBatches(fields map[string]interface{}, args ...string) ([]*BlockBatch, error) {
	query := "SELECT * FROM block_batch WHERE 1 = 1 "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var batches []*BlockBatch
	for rows.Next() {
		batch := &BlockBatch{}
		if err = rows.StructScan(batch); err != nil {
			break
		}
		batches = append(batches, batch)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return batches, rows.Close()
}

func (o *blockBatchOrm) GetProvingStatusByID(id uint64) (ProvingStatus, error) {
	row := o.db.QueryRow(`SELECT proving_status FROM block_batch WHERE id = $1`, id)
	var status ProvingStatus
	if err := row.Scan(&status); err != nil {
		return ProvingStatusUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) GetVerifiedProofAndInstanceByID(id uint64) ([]byte, []byte, error) {
	var proof []byte
	var instance []byte
	row := o.db.QueryRow(`SELECT proof, instance_commitments FROM block_batch WHERE id = $1 and proving_status = $2`, id, ProvingTaskVerified)

	if err := row.Scan(&proof, &instance); err != nil {
		return nil, nil, err
	}
	return proof, instance, nil
}

func (o *blockBatchOrm) UpdateProofByID(ctx context.Context, id uint64, proof, instance_commitments []byte, proofTimeSec uint64) error {
	db := o.db
	if _, err := db.ExecContext(ctx,
		db.Rebind(`UPDATE block_batch set proof = ?, instance_commitments = ?, proof_time_sec = ?, proved_at = ? where id = ?;`),
		proof, instance_commitments, proofTimeSec, time.Now(), id,
	); err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return nil
}

// `proved_at` is handled in `UpdateProofByID`, so we don't need to worry about it here
func (o *blockBatchOrm) UpdateProvingStatus(id uint64, status ProvingStatus) error {
	if _, err := o.db.Exec(o.db.Rebind("UPDATE block_batch set proving_status = ? where id = ?;"), status, id); err != nil {
		return err
	}
	return nil
}

// TODO: maybe we can use "`RETURNING` clause" like in
// https://stackoverflow.com/questions/33382981/go-how-to-get-last-insert-id-on-postgresql-with-namedexec
// then we don't need to manually query and manage this ID, and can define it as `SERIAL PRIMARY KEY` for auto-increment
func (o *blockBatchOrm) NewBatchInDBTx(dbTx *sqlx.Tx, startBlock *BlockInfo, endBlock *BlockInfo, parent_hash string, total_tx_num uint64, total_l2_gas uint64) (uint64, error) {
	row := dbTx.QueryRow("SELECT MAX(id) FROM block_batch;")

	var id int64 // 0 by default for sql.ErrNoRows
	if err := row.Scan(&id); err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	id++
	if _, err := dbTx.NamedExec(`INSERT INTO public.block_batch (id, total_l2_gas) VALUES (:id, :total_l2_gas)`,
		map[string]interface{}{
			"id":           id,
			"total_l2_gas": total_l2_gas,
			"created_at":   time.Now(),
		}); err != nil {
		return 0, err
	}

	return uint64(id), nil
}

func (o *blockBatchOrm) BatchRecordExist(id uint64) (bool, error) {
	var res int
	return res == 1, o.db.Get(&res, o.db.Rebind(`SELECT 1 FROM block_batch where id = ? limit 1;`), id)
}

func (o *blockBatchOrm) GetPendingBatches() ([]uint64, error) {
	rows, err := o.db.Queryx(`SELECT id FROM block_batch WHERE status = $1 ORDER BY id ASC`, RollupPending)
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
	row := o.db.QueryRow(`SELECT MAX(id) FROM block_batch WHERE status = $1;`, RollupFinalized)
	var id uint64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (o *blockBatchOrm) GetCommittedBatches() ([]uint64, error) {
	rows, err := o.db.Queryx(`SELECT id FROM block_batch WHERE rollup_status = $1 ORDER BY id ASC`, RollupCommitted)
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
	row := o.db.QueryRow(`SELECT rollup_status FROM block_batch WHERE id = $1`, id)
	var status RollupStatus
	if err := row.Scan(&status); err != nil {
		return RollupUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) UpdateRollupStatus(ctx context.Context, id uint64, status RollupStatus) error {
	switch status {
	case RollupCommitted:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ?, committed_at = ? where id = ?;"), status, time.Now(), id)
		return err
	case RollupFinalized:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ?, finalized_at = ? where id = ?;"), status, time.Now(), id)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ? where id = ?;"), status, id)
		return err
	}
}

func (o *blockBatchOrm) UpdateRollupTxHashAndRollupStatus(ctx context.Context, id uint64, rollup_tx_hash string, status RollupStatus) error {
	switch status {
	case RollupCommitted:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_tx_hash = ?, rollup_status = ?, committed_at = ? where id = ?;"), rollup_tx_hash, status, time.Now(), id)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_tx_hash = ?, rollup_status = ? where id = ?;"), rollup_tx_hash, status, id)
		return err
	}
}

func (o *blockBatchOrm) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, id uint64, finalize_tx_hash string, status RollupStatus) error {
	switch status {
	case RollupFinalized:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set finalize_tx_hash = ?, rollup_status = ?, finalized_at = ? where id = ?;"), finalize_tx_hash, status, time.Now(), id)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set finalize_tx_hash = ?, rollup_status = ? where id = ?;"), finalize_tx_hash, status, id)
		return err
	}
}
