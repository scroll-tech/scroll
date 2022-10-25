package orm

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
)

// RollupStatus rollupResult. status(pending, committing, committed, finalizing, finalized)
type RollupStatus int

const (
	// RollupUndefined : undefined rollup status
	RollupUndefined RollupStatus = iota
	// RollupPending : block is pending to rollup to layer1
	RollupPending
	// RollupCommitting : rollup transaction is submitted to layer1
	RollupCommitting
	// RollupCommitted : rollup transaction is confirmed to layer1
	RollupCommitted
	// RollupFinalizing : finalize transaction is submitted to layer1
	RollupFinalizing
	// RollupFinalized : finalize transaction is confirmed to layer1
	RollupFinalized
	// RollupFinalizationSkipped : finalize block is skipped
	RollupFinalizationSkipped
)

type rollupResultOrm struct {
	db *sqlx.DB
}

var _ RollupResultOrm = (*rollupResultOrm)(nil)

// NewRollupResultOrm create an rollupResultOrm instance
func NewRollupResultOrm(db *sqlx.DB) RollupResultOrm {
	return &rollupResultOrm{db: db}
}

func (o *rollupResultOrm) RollupRecordExist(id uint64) (bool, error) {
	var res int
	return res == 1, o.db.Get(&res, o.db.Rebind(`SELECT 1 from rollup_result where id = ? limit 1;`), id)
}

func (o *rollupResultOrm) GetPendingBatches() ([]uint64, error) {
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

func (o *rollupResultOrm) GetLatestFinalizedBatch() (uint64, error) {
	row := o.db.QueryRow(`SELECT MAX(id) FROM rollup_result WHERE status = $1;`, RollupFinalized)
	var id uint64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (o *rollupResultOrm) GetCommittedBatches() ([]uint64, error) {
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

func (o *rollupResultOrm) GetRollupStatus(id uint64) (RollupStatus, error) {
	row := o.db.QueryRow(`SELECT status FROM rollup_result WHERE id = $1`, id)
	var status RollupStatus
	if err := row.Scan(&status); err != nil {
		return RollupUndefined, err
	}
	return status, nil
}

func (o *rollupResultOrm) InsertPendingBatches(ctx context.Context, batches []uint64) error {
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

func (o *rollupResultOrm) UpdateRollupStatus(ctx context.Context, id uint64, status RollupStatus) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set status = ? where id = ?;"), status, id); err != nil {
		return err
	}
	return nil
}

func (o *rollupResultOrm) UpdateRollupTxHash(ctx context.Context, id uint64, rollup_tx_hash string) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set rollup_tx_hash = ? where id = ?;"), rollup_tx_hash, id); err != nil {
		return err
	}
	return nil
}

func (o *rollupResultOrm) UpdateFinalizeTxHash(ctx context.Context, id uint64, finalize_tx_hash string) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set finalize_tx_hash = ? where id = ?;"), finalize_tx_hash, id); err != nil {
		return err
	}
	return nil
}

func (o *rollupResultOrm) UpdateRollupTxHashAndStatus(ctx context.Context, id uint64, rollup_tx_hash string, status RollupStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update rollup_result set rollup_tx_hash = ?, status = ? where id = ?;"), rollup_tx_hash, status, id)
	return err
}

func (o *rollupResultOrm) UpdateFinalizeTxHashAndStatus(ctx context.Context, id uint64, finalize_tx_hash string, status RollupStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update rollup_result set finalize_tx_hash = ?, status = ? where id = ?;"), finalize_tx_hash, status, id)
	return err
}
