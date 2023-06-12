package orm

import (
	"scroll-tech/common/types"

	"github.com/jmoiron/sqlx"
)

type bridgeBatchOrm struct {
	db *sqlx.DB
}

type BridgeBatch struct {
	Index            uint64 `db:"index"`
	StartBlockNumber uint64 `db:"start_block_number"`
	EndBlockNumber   uint64 `db:"end_block_number"`
}

// NewBridgeBatchOrm create an NewBridgeBatchOrm instance
func NewBridgeBatchOrm(db *sqlx.DB) BridgeBatchOrm {
	return &bridgeBatchOrm{db: db}
}

func (b *bridgeBatchOrm) GetLatestBridgeBatch() (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT (index, start_block_number, end_block_number) FROM block_batch WHERE status = $1 DESC LIMIT 1;`, types.ProvingTaskVerified)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) GetBridgeBatchByBlock(height uint64) (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT (index, start_block_number, end_block_number) FROM block_batch WHERE start_block_number <= $1 AND end_block_number >= $1 AND status = $2;`, height, types.ProvingTaskVerified)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (b *bridgeBatchOrm) IsBlockInBatch(batchIndex uint64, height uint64) (bool, error) {
	var exists bool

	err := b.db.QueryRow(`SELECT EXISTS (SELECT 1 FROM block_batch WHERE batch_index = $1 AND start_block_number <= $2 AND end_block_number >= $2 )`, batchIndex, height).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (b *bridgeBatchOrm) GetBridgeBatchByIndex(index uint64) (*BridgeBatch, error) {
	result := &BridgeBatch{}
	row := b.db.QueryRowx(`SELECT (index, start_block_number, end_block_number) FROM block_batch WHERE index = $1;`, index)
	if err := row.StructScan(result); err != nil {
		return nil, err
	}
	return result, nil
}
