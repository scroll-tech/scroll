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

func (o *rollupResultOrm) RollupRecordExist(number uint64) (bool, error) {
	var res int
	return res == 1, o.db.Get(&res, o.db.Rebind(`SELECT 1 from rollup_result where number = ? limit 1;`), number)
}

func (o *rollupResultOrm) GetPendingBatches() ([]uint64, error) {
	rows, err := o.db.Queryx(`SELECT number FROM rollup_result WHERE status = $1 ORDER BY number ASC`, RollupPending)
	if err != nil {
		return nil, err
	}

	var blocks []uint64
	for rows.Next() {
		var number uint64
		if err = rows.Scan(&number); err != nil {
			break
		}
		blocks = append(blocks, number)
	}
	if len(blocks) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no pending blocks in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return blocks, rows.Close()
}

func (o *rollupResultOrm) GetLatestFinalizedBlock() (uint64, error) {
	row := o.db.QueryRow(`SELECT MAX(number) FROM rollup_result WHERE status = $1;`, RollupFinalized)
	var number uint64
	if err := row.Scan(&number); err != nil {
		return 0, err
	}
	return number, nil
}

func (o *rollupResultOrm) GetCommittedBatches() ([]uint64, error) {
	rows, err := o.db.Queryx(`SELECT number FROM rollup_result WHERE status = $1 ORDER BY number ASC`, RollupCommitted)
	if err != nil {
		return nil, err
	}

	var blocks []uint64
	for rows.Next() {
		var number uint64
		if err = rows.Scan(&number); err != nil {
			break
		}
		blocks = append(blocks, number)
	}
	if len(blocks) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no committed blocks in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return blocks, rows.Close()
}

func (o *rollupResultOrm) GetRollupStatus(number uint64) (RollupStatus, error) {
	row := o.db.QueryRow(`SELECT status FROM rollup_result WHERE number = $1`, number)
	var status RollupStatus
	if err := row.Scan(&status); err != nil {
		return RollupUndefined, err
	}
	return status, nil
}

func (o *rollupResultOrm) InsertPendingBlocks(ctx context.Context, blocks []uint64) error {
	blockMaps := make([]map[string]interface{}, len(blocks))
	for i, number := range blocks {
		blockMaps[i] = map[string]interface{}{
			"number": number,
		}
	}

	_, err := o.db.NamedExec(`INSERT INTO public.rollup_result (number) VALUES (:number);`, blockMaps)
	if err != nil {
		log.Error("failed to insert rollupResults", "err", err)
	}
	return err
}

func (o *rollupResultOrm) UpdateRollupStatus(ctx context.Context, number uint64, status RollupStatus) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set status = ? where number = ?;"), status, number); err != nil {
		return err
	}
	return nil
}

func (o *rollupResultOrm) UpdateRollupTxHash(ctx context.Context, number uint64, rollup_tx_hash string) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set rollup_tx_hash = ? where number = ?;"), rollup_tx_hash, number); err != nil {
		return err
	}
	return nil
}

func (o *rollupResultOrm) UpdateFinalizeTxHash(ctx context.Context, number uint64, finalize_tx_hash string) error {
	if _, err := o.db.Exec(o.db.Rebind("update rollup_result set finalize_tx_hash = ? where number = ?;"), finalize_tx_hash, number); err != nil {
		return err
	}
	return nil
}

func (o *rollupResultOrm) UpdateRollupTxHashAndStatus(ctx context.Context, number uint64, rollup_tx_hash string, status RollupStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update rollup_result set rollup_tx_hash = ?, status = ? where number = ?;"), rollup_tx_hash, status, number)
	return err
}

func (o *rollupResultOrm) UpdateFinalizeTxHashAndStatus(ctx context.Context, number uint64, finalize_tx_hash string, status RollupStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update rollup_result set finalize_tx_hash = ?, status = ? where number = ?;"), finalize_tx_hash, status, number)
	return err
}
