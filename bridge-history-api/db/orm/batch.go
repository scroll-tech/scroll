package orm

import (
	"database/sql"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

type rollupBatchOrm struct {
	db *sqlx.DB
}

type RollupBatch struct {
	ID               uint64 `json:"id" db:"id"`
	BatchIndex       uint64 `json:"batch_index" db:"batch_index"`
	BatchHash        string `json:"batch_hash" db:"batch_hash"`
	CommitHeight     uint64 `json:"commit_height" db:"commit_height"`
	StartBlockNumber uint64 `json:"start_block_number" db:"start_block_number"`
	EndBlockNumber   uint64 `json:"end_block_number" db:"end_block_number"`
}

// NewRollupBatchOrm create an NewRollupBatchOrm instance
func NewRollupBatchOrm(db *sqlx.DB) RollupBatchOrm {
	return &rollupBatchOrm{db: db}
}

func (b *rollupBatchOrm) BatchInsertRollupBatchDBTx(dbTx *sqlx.Tx, batches []*RollupBatch) error {
	if len(batches) == 0 {
		return nil
	}
	var err error
	messageMaps := make([]map[string]interface{}, len(batches))
	for i, msg := range batches {
		messageMaps[i] = map[string]interface{}{
			"commit_height":      msg.CommitHeight,
			"batch_index":        msg.BatchIndex,
			"batch_hash":         msg.BatchHash,
			"start_block_number": msg.StartBlockNumber,
			"end_block_number":   msg.EndBlockNumber,
		}
		var exists bool
		err = dbTx.QueryRow(`SELECT EXISTS(SELECT 1 FROM rollup_batch WHERE batch_index = $1 AND NOT is_deleted)`, msg.BatchIndex).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("BatchInsertRollupBatchDBTx: batch index %v already exists at height %v", msg.BatchIndex, msg.CommitHeight)
		}
	}
	_, err = dbTx.NamedExec(`insert into rollup_batch(commit_height, batch_index, batch_hash, start_block_number, end_block_number) values(:commit_height, :batch_index, :batch_hash, :start_block_number, :end_block_number);`, messageMaps)
	if err != nil {
		log.Error("BatchInsertRollupBatchDBTx: failed to insert batch event msgs", "err", err)
		return err
	}
	return nil
}

func (b *rollupBatchOrm) GetLatestRollupBatch() (*RollupBatch, error) {
	result := &RollupBatch{}
	row := b.db.QueryRowx(`SELECT id, batch_index, commit_height, batch_hash, start_block_number, end_block_number FROM rollup_batch ORDER BY batch_index DESC LIMIT 1;`)
	if err := row.StructScan(result); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (b *rollupBatchOrm) GetRollupBatchByIndex(index uint64) (*RollupBatch, error) {
	result := &RollupBatch{}
	row := b.db.QueryRowx(`SELECT id, batch_index, batch_hash, commit_height, start_block_number, end_block_number FROM rollup_batch WHERE batch_index = $1;`, index)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}
