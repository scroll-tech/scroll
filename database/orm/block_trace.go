package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	geth_types "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database/cache"

	"scroll-tech/common/utils"

	"scroll-tech/common/types"
)

type blockTraceOrm struct {
	db    *sqlx.DB
	cache cache.Cache
}

var _ BlockTraceOrm = (*blockTraceOrm)(nil)

// NewBlockTraceOrm create an blockTraceOrm instance
func NewBlockTraceOrm(db *sqlx.DB, cache cache.Cache) BlockTraceOrm {
	return &blockTraceOrm{db: db, cache: cache}
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

func (o *blockTraceOrm) GetL2BlockTracesLatestHeight() (int64, error) {
	row := o.db.QueryRow("SELECT COALESCE(MAX(number), -1) FROM block_trace;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	return height, nil
}

func (o *blockTraceOrm) GetL2BlockTraces(fields map[string]interface{}, args ...string) ([]*geth_types.BlockTrace, error) {
	query := "SELECT hash FROM block_trace WHERE 1 = 1 "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var (
		traces []*geth_types.BlockTrace
		rdb    = o.cache
	)
	for rows.Next() {
		var (
			trace *geth_types.BlockTrace
			hash  string
		)
		if err = rows.Scan(&hash); err != nil {
			break
		}
		trace, err = rdb.GetBlockTrace(context.Background(), common.HexToHash(hash))
		if err != nil {
			break
		}
		traces = append(traces, trace)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return traces, rows.Close()
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

	return blocks, rows.Close()
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

	return blocks, rows.Close()
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

func (o *blockTraceOrm) InsertL2BlockTraces(blockTraces []*geth_types.BlockTrace) error {
	traceMaps := make([]map[string]interface{}, len(blockTraces))
	rdb := o.cache
	for i, trace := range blockTraces {
		number, hash, txNum, mtime := trace.Header.Number.Int64(),
			trace.Header.Hash().String(),
			len(trace.Transactions),
			trace.Header.Time

		gasCost := utils.ComputeTraceGasCost(trace)
		traceMaps[i] = map[string]interface{}{
			"number":      number,
			"hash":        hash,
			"parent_hash": trace.Header.ParentHash.String(),
			// Empty trace content.
			"trace":           "{}",
			"tx_num":          txNum,
			"gas_used":        gasCost,
			"block_timestamp": mtime,
		}
		if rdb != nil {
			err := rdb.SetBlockTrace(context.Background(), trace)
			if err != nil {
				return err
			}
		}
	}
	_, err := o.db.NamedExec(`INSERT INTO public.block_trace (number, hash, parent_hash, trace, tx_num, gas_used, block_timestamp) VALUES (:number, :hash, :parent_hash, :trace, :tx_num, :gas_used, :block_timestamp);`, traceMaps)
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
