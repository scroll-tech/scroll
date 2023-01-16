package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/database/cache"

	"scroll-tech/common/utils"
)

type blockTraceOrm struct {
	db  *sqlx.DB
	rdb cache.Cache
}

var _ BlockTraceOrm = (*blockTraceOrm)(nil)

// NewBlockTraceOrm create an blockTraceOrm instance
func NewBlockTraceOrm(db *sqlx.DB, rdb cache.Cache) BlockTraceOrm {
	return &blockTraceOrm{
		db:  db,
		rdb: rdb,
	}
}

func (o *blockTraceOrm) Exist(number uint64) (bool, error) {
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

func (o *blockTraceOrm) GetBlockTracesLatestHeight() (int64, error) {
	row := o.db.QueryRow("SELECT COALESCE(MAX(number), -1) FROM block_trace;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	return height, nil
}

func (o *blockTraceOrm) GetBlockTraces(fields map[string]interface{}, args ...string) ([]*types.BlockTrace, error) {
	query := "SELECT hash,number FROM block_trace WHERE 1 = 1 "
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
		traces []*types.BlockTrace
		rdb    = o.rdb
	)
	for rows.Next() {
		var (
			trace  *types.BlockTrace
			hash   string
			number uint64
		)
		if err = rows.Scan(&hash, &number); err != nil {
			break
		}

		trace, err = rdb.GetBlockTrace(context.Background(), big.NewInt(0).SetUint64(number), common.HexToHash(hash))
		if err != nil {
			return nil, err
		}
		traces = append(traces, trace)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return traces, rows.Close()
}

func (o *blockTraceOrm) GetBlockInfos(fields map[string]interface{}, args ...string) ([]*BlockInfo, error) {
	query := "SELECT number, hash, parent_hash, batch_id, tx_num, gas_used, block_timestamp FROM block_trace WHERE 1 = 1 "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var blocks []*BlockInfo
	for rows.Next() {
		block := &BlockInfo{}
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

func (o *blockTraceOrm) GetUnbatchedBlocks(fields map[string]interface{}, args ...string) ([]*BlockInfo, error) {
	query := "SELECT number, hash, parent_hash, batch_id, tx_num, gas_used, block_timestamp FROM block_trace WHERE batch_id is NULL "
	for key := range fields {
		query += fmt.Sprintf("AND %s=:%s ", key, key)
	}
	query = strings.Join(append([]string{query}, args...), " ")

	db := o.db
	rows, err := db.NamedQuery(db.Rebind(query), fields)
	if err != nil {
		return nil, err
	}

	var blocks []*BlockInfo
	for rows.Next() {
		block := &BlockInfo{}
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

func (o *blockTraceOrm) GetHashByNumber(number uint64) (*common.Hash, error) {
	row := o.db.QueryRow(`SELECT hash FROM block_trace WHERE number = $1`, number)
	var hashStr string
	if err := row.Scan(&hashStr); err != nil {
		return nil, err
	}
	hash := common.HexToHash(hashStr)
	return &hash, nil
}

func (o *blockTraceOrm) InsertBlockTraces(blockTraces []*types.BlockTrace) error {
	traceMaps := make([]map[string]interface{}, len(blockTraces))
	rdb := o.rdb
	for i, trace := range blockTraces {
		number, hash, txNum, mtime := trace.Header.Number.Int64(),
			trace.Header.Hash().String(),
			len(trace.Transactions),
			trace.Header.Time

		gasCost := utils.ComputeTraceGasCost(trace)
		traceMaps[i] = map[string]interface{}{
			"number":          number,
			"hash":            hash,
			"parent_hash":     trace.Header.ParentHash.String(),
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
	_, err := o.db.NamedExec(`INSERT INTO public.block_trace (number, hash, parent_hash, tx_num, gas_used, block_timestamp) VALUES (:number, :hash, :parent_hash, :tx_num, :gas_used, :block_timestamp);`, traceMaps)
	if err != nil {
		log.Error("failed to insert blockTraces", "err", err)
	}
	return err
}

func (o *blockTraceOrm) DeleteTracesByBatchID(batchID string) error {
	if _, err := o.db.Exec(o.db.Rebind("update block_trace set trace = ? where batch_id = ?;"), "{}", batchID); err != nil {
		return err
	}
	return nil
}

// http://jmoiron.github.io/sqlx/#inQueries
// https://stackoverflow.com/questions/56568799/how-to-update-multiple-rows-using-sqlx
func (o *blockTraceOrm) SetBatchIDForBlocksInDBTx(dbTx *sqlx.Tx, numbers []uint64, batchID string) error {
	query := "UPDATE block_trace SET batch_id=? WHERE number IN (?)"

	qry, args, err := sqlx.In(query, batchID, numbers)
	if err != nil {
		return err
	}

	if _, err := dbTx.Exec(dbTx.Rebind(qry), args...); err != nil {
		return err
	}

	return nil
}
