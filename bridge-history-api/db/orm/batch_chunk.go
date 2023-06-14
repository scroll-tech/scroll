package orm

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/jmoiron/sqlx"
)

type bridgeBatchOrm struct {
	db *sqlx.DB
}

type BridgeBatch struct {
	ID               uint64 `db:"id"`
	Height           uint64 `db:"height"`
	StartBlockNumber uint64 `db:"start_block_number"`
	EndBlockNumber   uint64 `db:"end_block_number"`
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
			"start_block_number": msg.StartBlockNumber,
			"end_block_number":   msg.EndBlockNumber,
		}

		_, err = dbTx.NamedExec(`insert into bridge_batch(height, start_block_number, end_block_number) values(:height, :start_block_number, :end_block_number);`, messageMaps[i])
		if err != nil {
			log.Error("BatchInsertBridgeBatchDBTx: failed to insert batch event msgs", "height", msg.Height)
			break
		}
	}
	return err
}

func (b *bridgeBatchOrm) GetLatestBridgeBatch() (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT (id, height, start_block_number, end_block_number) FROM bridge_batch WHERE status = $1 ORDER BY id DESC LIMIT 1;`)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) GetBridgeBatchByBlock(height uint64) (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT (id, height, start_block_number, end_block_number) FROM bridge_batch WHERE start_block_number <= $1 AND end_block_number >= $1;`, height)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) IsBlockInBatch(batchIndex uint64, height uint64) (bool, error) {
	var exists bool

	err := b.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM bridge_batch WHERE id = $1 AND start_block_number <= $2 AND end_block_number >= $2 )`, batchIndex, height).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *bridgeBatchOrm) GetBridgeBatchByIndex(index uint64) (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT (id, height, start_block_number, end_block_number) FROM bridge_batch WHERE id = $1;`, index)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}
