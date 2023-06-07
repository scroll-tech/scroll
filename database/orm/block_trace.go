package orm

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"
)

type blockTraceOrm struct {
	db *sqlx.DB
}

var _ BlockTraceOrm = (*blockTraceOrm)(nil)

// NewBlockTraceOrm create an blockTraceOrm instance
func NewBlockTraceOrm(db *sqlx.DB) BlockTraceOrm {
	return &blockTraceOrm{db: db}
}

func (o *blockTraceOrm) IsL2BlockExists(number uint64) (bool, error) {
	var res int
	err := o.db.QueryRow(o.db.Rebind(`SELECT 1 from block_trace where number = ? limit 1;`), number).Scan(&res)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (o *blockTraceOrm) GetL2BlocksLatestHeight() (int64, error) {
	row := o.db.QueryRow("SELECT COALESCE(MAX(number), -1) FROM block_trace;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	return height, nil
}

func (o *blockTraceOrm) GetL2WrappedBlocks(fields map[string]interface{}, args ...string) ([]*types.WrappedBlock, error) {
	type Result struct {
		Trace string
	}

	query := "SELECT trace FROM block_trace WHERE 1 = 1 "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var wrappedBlocks []*types.WrappedBlock
	for rows.Next() {
		result := &Result{}
		if err = rows.StructScan(result); err != nil {
			break
		}
		wrappedBlock := types.WrappedBlock{}
		err = json.Unmarshal([]byte(result.Trace), &wrappedBlock)
		if err != nil {
			break
		}
		wrappedBlocks = append(wrappedBlocks, &wrappedBlock)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return wrappedBlocks, nil
}

func (o *blockTraceOrm) GetL2BlockInfos(fields map[string]interface{}, args ...string) ([]*types.BlockInfo, error) {
	query := "SELECT number, hash, parent_hash, batch_hash, tx_num, gas_used, block_timestamp FROM block_trace WHERE 1 = 1 "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var blocks []*types.BlockInfo
	for rows.Next() {
		block := &types.BlockInfo{}
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

func (o *blockTraceOrm) GetUnbatchedL2Blocks(fields map[string]interface{}, args ...string) ([]*types.BlockInfo, error) {
	query := "SELECT number, hash, parent_hash, batch_hash, tx_num, gas_used, block_timestamp FROM block_trace WHERE batch_hash is NULL "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var blocks []*types.BlockInfo
	for rows.Next() {
		block := &types.BlockInfo{}
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

func (o *blockTraceOrm) GetL2BlockHashByNumber(number uint64) (*common.Hash, error) {
	row := o.db.QueryRow(`SELECT hash FROM block_trace WHERE number = $1`, number)
	var hashStr string
	if err := row.Scan(&hashStr); err != nil {
		return nil, err
	}
	hash := common.HexToHash(hashStr)
	return &hash, nil
}

func (o *blockTraceOrm) InsertWrappedBlocks(blocks []*types.WrappedBlock) error {
	blockMaps := make([]map[string]interface{}, len(blocks))
	for i, block := range blocks {
		number, hash, txNum, mtime := block.Header.Number.Int64(),
			block.Header.Hash().String(),
			len(block.Transactions),
			block.Header.Time

		gasCost := block.Header.GasUsed
		data, err := json.Marshal(block)
		if err != nil {
			log.Error("failed to marshal block", "hash", hash, "err", err)
			return err
		}
		blockMaps[i] = map[string]interface{}{
			"number":          number,
			"hash":            hash,
			"parent_hash":     block.Header.ParentHash.String(),
			"trace":           string(data),
			"tx_num":          txNum,
			"gas_used":        gasCost,
			"block_timestamp": mtime,
		}
	}
	_, err := o.db.NamedExec(`INSERT INTO public.block_trace (number, hash, parent_hash, trace, tx_num, gas_used, block_timestamp) VALUES (:number, :hash, :parent_hash, :trace, :tx_num, :gas_used, :block_timestamp);`, blockMaps)
	if err != nil {
		log.Error("failed to insert blockTraces", "err", err)
	}
	return err
}

func (o *blockTraceOrm) DeleteTracesByBatchHash(batchHash string) error {
	if _, err := o.db.Exec(o.db.Rebind("update block_trace set trace = ? where batch_hash = ?;"), "{}", batchHash); err != nil {
		return err
	}
	return nil
}

// http://jmoiron.github.io/sqlx/#inQueries
// https://stackoverflow.com/questions/56568799/how-to-update-multiple-rows-using-sqlx
func (o *blockTraceOrm) SetBatchHashForL2BlocksInDBTx(dbTx *sqlx.Tx, numbers []uint64, batchHash string) error {
	query := "UPDATE block_trace SET batch_hash=? WHERE number IN (?)"

	qry, args, err := sqlx.In(query, batchHash, numbers)
	if err != nil {
		return err
	}

	if _, err := dbTx.Exec(dbTx.Rebind(qry), args...); err != nil {
		return err
	}

	return nil
}
