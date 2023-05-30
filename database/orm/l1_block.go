package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"scroll-tech/common/types"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"
)

type l1BlockOrm struct {
	db *sqlx.DB
}

var _ L1BlockOrm = (*l1BlockOrm)(nil)

// NewL1BlockOrm create an l1BlockOrm instance
func NewL1BlockOrm(db *sqlx.DB) L1BlockOrm {
	return &l1BlockOrm{db: db}
}

func (l *l1BlockOrm) GetL1BlockInfos(fields map[string]interface{}, args ...string) ([]*types.L1BlockInfo, error) {
	query := "SELECT * FROM l1_block WHERE 1 = 1 "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")
	query += " ORDER BY number ASC"

	db := l.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var blocks []*types.L1BlockInfo
	for rows.Next() {
		block := &types.L1BlockInfo{}
		if err = rows.StructScan(block); err != nil {
			break
		}
		blocks = append(blocks, block)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return blocks, nil
}

func (l *l1BlockOrm) InsertL1Blocks(ctx context.Context, blocks []*types.L1BlockInfo) error {
	if len(blocks) == 0 {
		return nil
	}

	blockMaps := make([]map[string]interface{}, len(blocks))
	for i, block := range blocks {
		blockMaps[i] = map[string]interface{}{
			"number":     block.Number,
			"hash":       block.Hash,
			"header_rlp": block.HeaderRLP,
			"base_fee":   block.BaseFee,
		}
	}
	_, err := l.db.NamedExec(`INSERT INTO public.l1_block (number, hash, header_rlp, base_fee) VALUES (:number, :hash, :header_rlp, :base_fee);`, blockMaps)
	if err != nil {
		log.Error("failed to insert L1 Blocks", "err", err)
	}
	return err
}

func (l *l1BlockOrm) DeleteHeaderRLPByBlockHash(ctx context.Context, blockHash string) error {
	if _, err := l.db.Exec(l.db.Rebind("update l1_block set header_rlp = ? where hash = ?;"), "", blockHash); err != nil {
		return err
	}
	return nil
}

func (l *l1BlockOrm) UpdateImportTxHash(ctx context.Context, blockHash, txHash string) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update l1_block set import_tx_hash = ? where hash = ?;"), txHash, blockHash); err != nil {
		return err
	}

	return nil
}

func (l *l1BlockOrm) UpdateL1BlockStatus(ctx context.Context, blockHash string, status types.L1BlockStatus) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update l1_block set block_status = ? where hash = ?;"), status, blockHash); err != nil {
		return err
	}

	return nil
}

func (l *l1BlockOrm) UpdateL1BlockStatusAndImportTxHash(ctx context.Context, blockHash string, status types.L1BlockStatus, txHash string) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update l1_block set block_status = ?, import_tx_hash = ? where hash = ?;"), status, txHash, blockHash); err != nil {
		return err
	}

	return nil
}

func (l *l1BlockOrm) UpdateL1OracleTxHash(ctx context.Context, blockHash, txHash string) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update l1_block set oracle_tx_hash = ? where hash = ?;"), txHash, blockHash); err != nil {
		return err
	}

	return nil
}

func (l *l1BlockOrm) UpdateL1GasOracleStatus(ctx context.Context, blockHash string, status types.GasOracleStatus) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update l1_block set oracle_status = ? where hash = ?;"), status, blockHash); err != nil {
		return err
	}

	return nil
}

func (l *l1BlockOrm) UpdateL1GasOracleStatusAndOracleTxHash(ctx context.Context, blockHash string, status types.GasOracleStatus, txHash string) error {
	if _, err := l.db.ExecContext(ctx, l.db.Rebind("update l1_block set oracle_status = ?, oracle_tx_hash = ? where hash = ?;"), status, txHash, blockHash); err != nil {
		return err
	}

	return nil
}

func (l *l1BlockOrm) GetLatestL1BlockHeight() (uint64, error) {
	row := l.db.QueryRow("SELECT COALESCE(MAX(number), 0) FROM l1_block;")

	var height uint64
	if err := row.Scan(&height); err != nil {
		return 0, err
	}
	return height, nil
}

func (l *l1BlockOrm) GetLatestImportedL1Block() (*types.L1BlockInfo, error) {
	row := l.db.QueryRowx(`SELECT * FROM l1_block WHERE block_status = $1 ORDER BY index DESC;`, types.L1BlockImported)
	block := &types.L1BlockInfo{}
	if err := row.StructScan(block); err != nil {
		return nil, err
	}
	return block, nil
}
