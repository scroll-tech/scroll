package orm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types"
	"scroll-tech/common/types/message"
)

type blockBatchOrm struct {
	db *sqlx.DB
}

var _ BlockBatchOrm = (*blockBatchOrm)(nil)

// NewBlockBatchOrm create an blockBatchOrm instance
func NewBlockBatchOrm(db *sqlx.DB) BlockBatchOrm {
	return &blockBatchOrm{db: db}
}

func (o *blockBatchOrm) GetBlockBatches(fields map[string]interface{}, args ...string) ([]*types.BlockBatch, error) {
	query := "SELECT * FROM block_batch WHERE 1 = 1 "
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

	var batches []*types.BlockBatch
	for rows.Next() {
		batch := &types.BlockBatch{}
		if err = rows.StructScan(batch); err != nil {
			break
		}
		batches = append(batches, batch)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return batches, nil
}

func (o *blockBatchOrm) GetProvingStatusByHash(hash string) (types.ProvingStatus, error) {
	row := o.db.QueryRow(`SELECT proving_status FROM block_batch WHERE hash = $1`, hash)
	var status types.ProvingStatus
	if err := row.Scan(&status); err != nil {
		return types.ProvingStatusUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) GetVerifiedProofByHash(hash string) (*message.AggProof, error) {
	var proofBytes []byte
	row := o.db.QueryRow(`SELECT proof FROM block_batch WHERE hash = $1 and proving_status = $2`, hash, types.ProvingTaskVerified)
	if err := row.Scan(&proofBytes); err != nil {
		return nil, err
	}

	var proof message.AggProof
	if err := json.Unmarshal(proofBytes, &proof); err != nil {
		return nil, err
	}

	return &proof, nil
}

func (o *blockBatchOrm) UpdateProofByHash(ctx context.Context, hash string, proof *message.AggProof, proofTimeSec uint64) error {
	proofBytes, err := json.Marshal(proof)
	if err != nil {
		return err
	}

	db := o.db
	if _, err := db.ExecContext(ctx,
		db.Rebind(`UPDATE block_batch set proof = ?, proof_time_sec = ? where hash = ?;`),
		proofBytes, proofTimeSec, hash,
	); err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return nil
}

func (o *blockBatchOrm) UpdateProvingStatus(hash string, status types.ProvingStatus) error {
	switch status {
	case types.ProvingTaskAssigned:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ?, prover_assigned_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	case types.ProvingTaskUnassigned:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ?, prover_assigned_at = null where hash = ?;"), status, hash)
		return err
	case types.ProvingTaskProved, types.ProvingTaskVerified:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ?, proved_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ? where hash = ?;"), status, hash)
		return err
	}
}

func (o *blockBatchOrm) ResetProvingStatusFor(before types.ProvingStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ? where proving_status = ?;"), types.ProvingTaskUnassigned, before)
	return err
}

// func (o *blockBatchOrm) NewBatchInDBTx(dbTx *sqlx.Tx, startBlock *BlockInfo, endBlock *BlockInfo, parentHash string, totalTxNum uint64, totalL2Gas uint64) (string, error) {
func (o *blockBatchOrm) NewBatchInDBTx(dbTx *sqlx.Tx, batchData *types.BatchData) error {
	numBlocks := len(batchData.Batch.Blocks)
	if _, err := dbTx.NamedExec(`INSERT INTO public.block_batch (hash, index, parent_hash, start_block_number, start_block_hash, end_block_number, end_block_hash, total_tx_num, total_l2_gas, state_root, total_l1_tx_num) VALUES (:hash, :index, :parent_hash, :start_block_number, :start_block_hash, :end_block_number, :end_block_hash, :total_tx_num, :total_l2_gas, :state_root, :total_l1_tx_num)`,
		map[string]interface{}{
			"hash":               batchData.Hash().Hex(),
			"index":              batchData.Batch.BatchIndex,
			"parent_hash":        batchData.Batch.ParentBatchHash.Hex(),
			"start_block_number": batchData.Batch.Blocks[0].BlockNumber,
			"start_block_hash":   batchData.Batch.Blocks[0].BlockHash.Hex(),
			"end_block_number":   batchData.Batch.Blocks[numBlocks-1].BlockNumber,
			"end_block_hash":     batchData.Batch.Blocks[numBlocks-1].BlockHash.Hex(),
			"total_tx_num":       batchData.TotalTxNum,
			"total_l1_tx_num":    batchData.TotalL1TxNum,
			"total_l2_gas":       batchData.TotalL2Gas,
			"state_root":         batchData.Batch.NewStateRoot.Hex(),
			"created_at":         time.Now(),
			// "proving_status":     ProvingTaskUnassigned, // actually no need, because we have default value in DB schema
			// "rollup_status":      RollupPending,         // actually no need, because we have default value in DB schema
		}); err != nil {
		return err
	}

	return nil
}

func (o *blockBatchOrm) BatchRecordExist(hash string) (bool, error) {
	var res int
	err := o.db.QueryRow(o.db.Rebind(`SELECT 1 FROM block_batch where hash = ? limit 1;`), hash).Scan(&res)
	if err != nil {
		if err != sql.ErrNoRows {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (o *blockBatchOrm) GetPendingBatches(limit uint64) ([]string, error) {
	rows, err := o.db.Queryx(`SELECT hash FROM block_batch WHERE rollup_status = $1 ORDER BY index ASC LIMIT $2`, types.RollupPending, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var hashes []string
	for rows.Next() {
		var hash string
		if err = rows.Scan(&hash); err != nil {
			break
		}
		hashes = append(hashes, hash)
	}
	if len(hashes) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no pending batches in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return hashes, nil
}

func (o *blockBatchOrm) GetLatestBatch() (*types.BlockBatch, error) {
	row := o.db.QueryRowx(`select * from block_batch where index = (select max(index) from block_batch);`)
	batch := &types.BlockBatch{}
	if err := row.StructScan(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (o *blockBatchOrm) GetLatestCommittedBatch() (*types.BlockBatch, error) {
	row := o.db.QueryRowx(`select * from block_batch where index = (select max(index) from block_batch where rollup_status = $1);`, types.RollupCommitted)
	batch := &types.BlockBatch{}
	if err := row.StructScan(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (o *blockBatchOrm) GetLatestFinalizedBatch() (*types.BlockBatch, error) {
	row := o.db.QueryRowx(`select * from block_batch where index = (select max(index) from block_batch where rollup_status = $1);`, types.RollupFinalized)
	batch := &types.BlockBatch{}
	if err := row.StructScan(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (o *blockBatchOrm) GetLatestFinalizingOrFinalizedBatch() (*types.BlockBatch, error) {
	row := o.db.QueryRowx(`select * from block_batch where index = (select max(index) from block_batch where rollup_status = $1 or rollup_status = $2);`, types.RollupFinalizing, types.RollupFinalized)
	batch := &types.BlockBatch{}
	if err := row.StructScan(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (o *blockBatchOrm) GetCommittedBatches(limit uint64) ([]string, error) {
	rows, err := o.db.Queryx(`SELECT hash FROM block_batch WHERE rollup_status = $1 ORDER BY index ASC LIMIT $2`, types.RollupCommitted, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var hashes []string
	for rows.Next() {
		var hash string
		if err = rows.Scan(&hash); err != nil {
			break
		}
		hashes = append(hashes, hash)
	}
	if len(hashes) == 0 || errors.Is(err, sql.ErrNoRows) {
		// log.Warn("no committed batches in db", "err", err)
	} else if err != nil {
		return nil, err
	}

	return hashes, nil
}

func (o *blockBatchOrm) GetRollupStatus(hash string) (types.RollupStatus, error) {
	row := o.db.QueryRow(`SELECT rollup_status FROM block_batch WHERE hash = $1`, hash)
	var status types.RollupStatus
	if err := row.Scan(&status); err != nil {
		return types.RollupUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) GetRollupStatusByHashList(hashes []string) ([]types.RollupStatus, error) {
	if len(hashes) == 0 {
		return make([]types.RollupStatus, 0), nil
	}

	query, args, err := sqlx.In("SELECT hash, rollup_status FROM block_batch WHERE hash IN (?);", hashes)
	if err != nil {
		return make([]types.RollupStatus, 0), err
	}
	// sqlx.In returns queries with the `?` bindvar, we can rebind it for our backend
	query = o.db.Rebind(query)

	rows, err := o.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	statusMap := make(map[string]types.RollupStatus)
	for rows.Next() {
		var hash string
		var status types.RollupStatus
		if err = rows.Scan(&hash, &status); err != nil {
			break
		}
		statusMap[hash] = status
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	var statuses []types.RollupStatus
	if err != nil {
		return statuses, err
	}

	for _, hash := range hashes {
		statuses = append(statuses, statusMap[hash])
	}

	return statuses, nil
}

func (o *blockBatchOrm) GetCommitTxHash(hash string) (sql.NullString, error) {
	row := o.db.QueryRow(`SELECT commit_tx_hash FROM block_batch WHERE hash = $1`, hash)
	var commitTXHash sql.NullString
	if err := row.Scan(&commitTXHash); err != nil {
		return sql.NullString{}, err
	}
	return commitTXHash, nil
}

func (o *blockBatchOrm) GetFinalizeTxHash(hash string) (sql.NullString, error) {
	row := o.db.QueryRow(`SELECT finalize_tx_hash FROM block_batch WHERE hash = $1`, hash)
	var finalizeTxHash sql.NullString
	if err := row.Scan(&finalizeTxHash); err != nil {
		return sql.NullString{}, err
	}
	return finalizeTxHash, nil
}

func (o *blockBatchOrm) UpdateRollupStatus(ctx context.Context, hash string, status types.RollupStatus) error {
	switch status {
	case types.RollupCommitted:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ?, committed_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	case types.RollupFinalized:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ?, finalized_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ? where hash = ?;"), status, hash)
		return err
	}
}

func (o *blockBatchOrm) UpdateCommitTxHashAndRollupStatus(ctx context.Context, hash string, commitTxHash string, status types.RollupStatus) error {
	switch status {
	case types.RollupCommitted:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set commit_tx_hash = ?, rollup_status = ?, committed_at = ? where hash = ?;"), commitTxHash, status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set commit_tx_hash = ?, rollup_status = ? where hash = ?;"), commitTxHash, status, hash)
		return err
	}
}

func (o *blockBatchOrm) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status types.RollupStatus) error {
	switch status {
	case types.RollupFinalized:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set finalize_tx_hash = ?, rollup_status = ?, finalized_at = ? where hash = ?;"), finalizeTxHash, status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set finalize_tx_hash = ?, rollup_status = ? where hash = ?;"), finalizeTxHash, status, hash)
		return err
	}
}

func (o *blockBatchOrm) GetAssignedBatchHashes() ([]string, error) {
	rows, err := o.db.Queryx(`SELECT hash FROM block_batch WHERE proving_status IN ($1, $2)`, types.ProvingTaskAssigned, types.ProvingTaskProved)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var hashes []string
	for rows.Next() {
		var hash string
		if err = rows.Scan(&hash); err != nil {
			break
		}
		hashes = append(hashes, hash)
	}

	return hashes, nil
}

func (o *blockBatchOrm) GetBatchCount() (int64, error) {
	row := o.db.QueryRow(`select count(*) from block_batch`)
	var count int64
	if err := row.Scan(&count); err != nil {
		return -1, err
	}
	return count, nil
}

func (o *blockBatchOrm) UpdateSkippedBatches() (int64, error) {
	res, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ? where (proving_status = ? or proving_status = ?) and rollup_status = ?;"), types.RollupFinalizationSkipped, types.ProvingTaskSkipped, types.ProvingTaskFailed, types.RollupCommitted)
	if err != nil {
		return 0, err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (o *blockBatchOrm) UpdateL2OracleTxHash(ctx context.Context, hash, txHash string) error {
	if _, err := o.db.ExecContext(ctx, o.db.Rebind("update block_batch set oracle_tx_hash = ? where hash = ?;"), txHash, hash); err != nil {
		return err
	}

	return nil
}

func (o *blockBatchOrm) UpdateL2GasOracleStatus(ctx context.Context, hash string, status types.GasOracleStatus) error {
	if _, err := o.db.ExecContext(ctx, o.db.Rebind("update block_batch set oracle_status = ? where hash = ?;"), status, hash); err != nil {
		return err
	}

	return nil
}

func (o *blockBatchOrm) UpdateL2GasOracleStatusAndOracleTxHash(ctx context.Context, hash string, status types.GasOracleStatus, txHash string) error {
	if _, err := o.db.ExecContext(ctx, o.db.Rebind("update block_batch set oracle_status = ?, oracle_tx_hash = ? where hash = ?;"), status, txHash, hash); err != nil {
		return err
	}

	return nil
}
