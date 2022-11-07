package orm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

// BlockStatus blockResult status(unassigned, assigned, proved, verified, submitted)
type BlockStatus int

const (
	// BlockUndefined : undefined block status
	BlockUndefined BlockStatus = iota
	// BlockUnassigned : block is not assigned to be proved
	BlockUnassigned
	// BlockSkipped : block is skipped for proof generation
	BlockSkipped
	// BlockAssigned : block is assigned to be proved
	BlockAssigned
	// BlockProved : block proof has been returned by prover
	BlockProved
	// BlockVerified : block proof is valid
	BlockVerified
	// BlockFailed : fail to generate block proof
	BlockFailed
)

func (bs BlockStatus) String() string {
	switch bs {
	case BlockUnassigned:
		return "unassigned"
	case BlockSkipped:
		return "skipped"
	case BlockAssigned:
		return "assigned"
	case BlockProved:
		return "proved"
	case BlockVerified:
		return "undefined"
	case BlockFailed:
		return "failed"
	default:
		return "undefined"
	}
}

type blockResultOrm struct {
	db *sqlx.DB
}

var _ BlockResultOrm = (*blockResultOrm)(nil)

// NewBlockResultOrm create an blockResultOrm instance
func NewBlockResultOrm(db *sqlx.DB) BlockResultOrm {
	return &blockResultOrm{db: db}
}

func (o *blockResultOrm) Exist(number uint64) (bool, error) {
	var res int
	return res == 1, o.db.Get(&res, o.db.Rebind(`SELECT 1 from block_result where number = ? limit 1;`), number)
}

func (o *blockResultOrm) GetBlockResultsLatestHeight() (int64, error) {
	row := o.db.QueryRow("SELECT COALESCE(MAX(number), -1) FROM block_result;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	return height, nil
}

func (o *blockResultOrm) GetBlockResultsOldestHeight() (int64, error) {
	row := o.db.QueryRow("SELECT COALESCE(MIN(number), -1) FROM block_result;")

	var height int64
	if err := row.Scan(&height); err != nil {
		return -1, err
	}
	return height, nil
}

func (o *blockResultOrm) GetBlockResults(fields map[string]interface{}, args ...string) ([]*types.BlockResult, error) {
	type Result struct {
		Content string
	}

	query := "SELECT content FROM block_result WHERE 1 = 1 "
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
		err = json.Unmarshal([]byte(result.Content), &trace)
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

func (o *blockResultOrm) GetVerifiedProofAndInstanceByNumber(number uint64) ([]byte, []byte, error) {
	var proof []byte
	var instance []byte
	row := o.db.QueryRow(`SELECT proof, instance_commitments FROM block_result WHERE number = $1 and status = $2`, number, BlockVerified)

	if err := row.Scan(&proof, &instance); err != nil {
		return nil, nil, err
	}
	return proof, instance, nil
}

func (o *blockResultOrm) GetHashByNumber(number uint64) (*common.Hash, error) {
	row := o.db.QueryRow(`SELECT hash FROM block_result WHERE number = $1`, number)
	var hashStr string
	if err := row.Scan(&hashStr); err != nil {
		return nil, err
	}
	hash := common.HexToHash(hashStr)
	return &hash, nil
}

func (o *blockResultOrm) GetBlockStatusByNumber(number uint64) (BlockStatus, error) {
	row := o.db.QueryRow(`SELECT status FROM block_result WHERE number = $1`, number)
	var status BlockStatus
	if err := row.Scan(&status); err != nil {
		return BlockUndefined, err
	}
	return status, nil
}

func (o *blockResultOrm) InsertBlockResultsWithStatus(ctx context.Context, blockResults []*types.BlockResult, status BlockStatus) error {
	traceMaps := make([]map[string]interface{}, len(blockResults))
	for i, trace := range blockResults {
		number, hash, tx_num, mtime := trace.BlockTrace.Number.ToInt().Int64(),
			trace.BlockTrace.Hash.String(),
			len(trace.BlockTrace.Transactions),
			trace.BlockTrace.Time
		var data []byte
		data, err := json.Marshal(trace)
		if err != nil {
			log.Error("failed to marshal blockResult", "hash", hash, "err", err)
			return err
		}
		traceMaps[i] = map[string]interface{}{
			"number":          number,
			"hash":            hash,
			"content":         string(data),
			"status":          status,
			"tx_num":          tx_num,
			"block_timestamp": mtime,
		}
	}

	_, err := o.db.NamedExec(`INSERT INTO public.block_result (number, hash, content, status, tx_num, block_timestamp) VALUES (:number, :hash, :content, :status, :tx_num, :block_timestamp);`, traceMaps)
	if err != nil {
		log.Error("failed to insert blockResults", "err", err)
	}
	return err
}

func (o *blockResultOrm) UpdateProofByNumber(ctx context.Context, number uint64, proof, instance_commitments []byte, proofTimeSec uint64) error {
	db := o.db
	if _, err := db.ExecContext(ctx, db.Rebind(`update block_result set proof = ?, instance_commitments = ?, proof_time_sec = ? where number = ?;`), proof, instance_commitments, proofTimeSec, number); err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return nil
}

func (o *blockResultOrm) UpdateBlockStatus(number uint64, status BlockStatus) error {
	if _, err := o.db.Exec(o.db.Rebind("update block_result set status = ? where number = ?;"), status, number); err != nil {
		return err
	}
	return nil
}

func (o *blockResultOrm) DeleteTraceByNumber(number uint64) error {
	if _, err := o.db.Exec(o.db.Rebind("update block_result set content = ? where number = ?;"), "{}", number); err != nil {
		return err
	}
	return nil
}
