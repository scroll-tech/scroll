package orm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

type blockTraceOrm struct {
	db *sqlx.DB
}

var _ BlockTraceOrm = (*blockTraceOrm)(nil)

// NewBlockTraceOrm create an blockTraceOrm instance
func NewBlockTraceOrm(db *sqlx.DB) BlockTraceOrm {
	return &blockTraceOrm{db: db}
}

func (o *blockTraceOrm) Exist(number uint64) (bool, error) {
	var res int
	return res == 1, o.db.Get(&res, o.db.Rebind(`SELECT 1 from block_trace where number = ? limit 1;`), number)
}

func (o *blockTraceOrm) GetBlockTracesLatestHeight() (int64, error) {
	row := o.db.QueryRow("SELECT COALESCE(MAX(number), -1) FROM block_trace;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	return height, nil
}

func (o *blockTraceOrm) GetBlockTracesOldestHeight() (int64, error) {
	row := o.db.QueryRow("SELECT COALESCE(MIN(number), -1) FROM block_trace;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	return height, nil
}

func (o *blockTraceOrm) GetBlockTraces(fields map[string]interface{}, args ...string) ([]*types.BlockResult, error) {
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

	var traces []*types.BlockResult
	for rows.Next() {
		result := &Result{}
		if err = rows.StructScan(result); err != nil {
			break
		}
		trace := types.BlockResult{}
		err = json.Unmarshal([]byte(result.Trace), &trace)
		if err != nil {
			break
		}
		traces = append(traces, &trace)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return traces, rows.Close()
}

func (o *blockTraceOrm) GetBlockInfos(fields map[string]interface{}, args ...string) ([]*BlockInfo, error) {
	query := "SELECT number, hash, batch_id, tx_num, gas_used, block_timestamp FROM block_trace WHERE 1 = 1 "
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
	query := "SELECT number, hash, batch_id, tx_num, gas_used, block_timestamp FROM block_trace WHERE batch_id is NULL "
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

func (o *blockTraceOrm) InsertBlockTraces(ctx context.Context, blockTraces []*types.BlockResult) error {
	traceMaps := make([]map[string]interface{}, len(blockTraces))
	for i, trace := range blockTraces {
		number, hash, tx_num, mtime := trace.BlockTrace.Number.ToInt().Int64(),
			trace.BlockTrace.Hash.String(),
			len(trace.BlockTrace.Transactions),
			trace.BlockTrace.Time

		var gasUsed uint64
		for _, tx := range trace.BlockTrace.Transactions {
			gasUsed += tx.Gas
		}

		data, err := json.Marshal(trace)
		if err != nil {
			log.Error("failed to marshal blockResult", "hash", hash, "err", err)
			return err
		}
		traceMaps[i] = map[string]interface{}{
			"number":          number,
			"hash":            hash,
			"trace":           string(data),
			"tx_num":          tx_num,
			"gas_used":        gasUsed,
			"block_timestamp": mtime,
		}
	}

	_, err := o.db.NamedExec(`INSERT INTO public.block_trace (number, hash, trace, tx_num, gas_used, block_timestamp) VALUES (:number, :hash, :trace, :tx_num, :gas_used, :block_timestamp);`, traceMaps)
	if err != nil {
		log.Error("failed to insert blockTraces", "err", err)
	}
	return err
}

func (o *blockTraceOrm) DeleteTracesByBatchID(batch_id string) error {
	if _, err := o.db.Exec(o.db.Rebind("update block_trace set trace = ? where batch_id = ?;"), "{}", batch_id); err != nil {
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
