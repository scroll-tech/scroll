package orm

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

type BatchStatus int

const (
	// BatchWithoutProof represents batch is not used to compute proof
	BatchWithoutProof BatchStatus = iota

	// BatchWithProof represents batch is used to compute proof
	BatchWithProof
)

type bridgeBatchOrm struct {
	db *sqlx.DB
}

type BridgeBatch struct {
	ID               uint64      `json:"id" db:"id"`
	BatchHash        string      `json:"batch_hash" db:"batch_hash"`
	Height           uint64      `json:"height" db:"height"`
	StartBlockNumber uint64      `json:"start_block_number" db:"start_block_number"`
	EndBlockNumber   uint64      `json:"end_block_number" db:"end_block_number"`
	Status           BatchStatus `json:"status" db:"status"`
}

// NewBridgeBatchOrm create an NewBridgeBatchOrm instance
func NewBridgeBatchOrm(db *sqlx.DB) BridgeBatchOrm {
	return &bridgeBatchOrm{db: db}
}

func (b *bridgeBatchOrm) BatchInsertBridgeBatchDBTx(dbTx *sqlx.Tx, messages []*BridgeBatch) error {
	if len(messages) == 0 {
		return nil
	}
	var err error
	messageMaps := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		messageMaps[i] = map[string]interface{}{
			"height":             msg.Height,
			"batch_hash":         msg.BatchHash,
			"start_block_number": msg.StartBlockNumber,
			"end_block_number":   msg.EndBlockNumber,
		}

		_, err = dbTx.NamedExec(`insert into bridge_batch(height, batch_hash, start_block_number, end_block_number) values(:height, :batch_hash, :start_block_number, :end_block_number);`, messageMaps[i])
		if err != nil {
			log.Error("BatchInsertBridgeBatchDBTx: failed to insert batch event msgs", "height", msg.Height)
			break
		}
	}
	return err
}

func (b *bridgeBatchOrm) GetLatestBridgeBatch() (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT id, height, batch_hash, start_block_number, end_block_number FROM bridge_batch ORDER BY id DESC LIMIT 1;`)
	if err := row.StructScan(result); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) GetBridgeBatchByBlock(height uint64) (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT id, height, start_block_number, end_block_number, status FROM bridge_batch WHERE start_block_number <= $1 AND end_block_number >= $1;`, height)
	if err := row.StructScan(result); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) GetLatestBridgeBatchWithProof() (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT id, height, start_block_number, end_block_number, status FROM bridge_batch WHERE status = $1 ORDER BY id DESC LIMIT 1;`, BatchWithProof)
	if err := row.StructScan(result); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) GetBridgeBatchByIndex(index uint64) (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT id, height, start_block_number, end_block_number FROM bridge_batch WHERE id = $1;`, index)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) UpdateBridgeBatchStatusDBTx(dbTx *sqlx.Tx, batchID uint64, status BatchStatus) error {
	_, err := dbTx.Exec(`UPDATE bridge_batch SET status = $1 WHERE id = $2;`, status, batchID)
	return err
}
