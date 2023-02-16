package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/utils"
)

// ProvingStatus block_batch proving_status (unassigned, assigned, proved, verified, submitted)
type ProvingStatus int

const (
	// ProvingStatusUndefined : undefined proving_task status
	ProvingStatusUndefined ProvingStatus = iota
	// ProvingTaskUnassigned : proving_task is not assigned to be proved
	ProvingTaskUnassigned
	// ProvingTaskSkipped : proving_task is skipped for proof generation
	ProvingTaskSkipped
	// ProvingTaskAssigned : proving_task is assigned to be proved
	ProvingTaskAssigned
	// ProvingTaskProved : proof has been returned by prover
	ProvingTaskProved
	// ProvingTaskVerified : proof is valid
	ProvingTaskVerified
	// ProvingTaskFailed : fail to generate proof
	ProvingTaskFailed
)

func (ps ProvingStatus) String() string {
	switch ps {
	case ProvingTaskUnassigned:
		return "unassigned"
	case ProvingTaskSkipped:
		return "skipped"
	case ProvingTaskAssigned:
		return "assigned"
	case ProvingTaskProved:
		return "proved"
	case ProvingTaskVerified:
		return "verified"
	case ProvingTaskFailed:
		return "failed"
	default:
		return "undefined"
	}
}

// RollupStatus block_batch rollup_status (pending, committing, committed, finalizing, finalized)
type RollupStatus int

const (
	// RollupUndefined : undefined rollup status
	RollupUndefined RollupStatus = iota
	// RollupPending : batch is pending to rollup to layer1
	RollupPending
	// RollupCommitting : rollup transaction is submitted to layer1
	RollupCommitting
	// RollupCommitted : rollup transaction is confirmed to layer1
	RollupCommitted
	// RollupFinalizing : finalize transaction is submitted to layer1
	RollupFinalizing
	// RollupFinalized : finalize transaction is confirmed to layer1
	RollupFinalized
	// RollupFinalizationSkipped : batch finalization is skipped
	RollupFinalizationSkipped
)

// BlockBatch is structure of stored block_batch
type BlockBatch struct {
	Hash                string         `json:"hash" db:"hash"`
	Index               uint64         `json:"index" db:"index"`
	ParentHash          string         `json:"parent_hash" db:"parent_hash"`
	StartBlockNumber    uint64         `json:"start_block_number" db:"start_block_number"`
	StartBlockHash      string         `json:"start_block_hash" db:"start_block_hash"`
	EndBlockNumber      uint64         `json:"end_block_number" db:"end_block_number"`
	EndBlockHash        string         `json:"end_block_hash" db:"end_block_hash"`
	StateRoot           string         `json:"state_root" db:"state_root"`
	TotalTxNum          uint64         `json:"total_tx_num" db:"total_tx_num"`
	TotalL1TxNum        uint64         `json:"total_l1_tx_num" db:"total_l1_tx_num"`
	TotalL2Gas          uint64         `json:"total_l2_gas" db:"total_l2_gas"`
	ProvingStatus       ProvingStatus  `json:"proving_status" db:"proving_status"`
	Proof               []byte         `json:"proof" db:"proof"`
	InstanceCommitments []byte         `json:"instance_commitments" db:"instance_commitments"`
	ProofTimeSec        uint64         `json:"proof_time_sec" db:"proof_time_sec"`
	RollupStatus        RollupStatus   `json:"rollup_status" db:"rollup_status"`
	CommitTxHash        sql.NullString `json:"commit_tx_hash" db:"commit_tx_hash"`
	FinalizeTxHash      sql.NullString `json:"finalize_tx_hash" db:"finalize_tx_hash"`
	CreatedAt           *time.Time     `json:"created_at" db:"created_at"`
	ProverAssignedAt    *time.Time     `json:"prover_assigned_at" db:"prover_assigned_at"`
	ProvedAt            *time.Time     `json:"proved_at" db:"proved_at"`
	CommittedAt         *time.Time     `json:"committed_at" db:"committed_at"`
	FinalizedAt         *time.Time     `json:"finalized_at" db:"finalized_at"`
}

type blockBatchOrm struct {
	db *sqlx.DB
}

var _ BlockBatchOrm = (*blockBatchOrm)(nil)

// NewBlockBatchOrm create an blockBatchOrm instance
func NewBlockBatchOrm(db *sqlx.DB) BlockBatchOrm {
	return &blockBatchOrm{db: db}
}

func (o *blockBatchOrm) GetBlockBatches(fields map[string]interface{}, args ...string) ([]*BlockBatch, error) {
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

	var batches []*BlockBatch
	for rows.Next() {
		batch := &BlockBatch{}
		if err = rows.StructScan(batch); err != nil {
			break
		}
		batches = append(batches, batch)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return batches, rows.Close()
}

func (o *blockBatchOrm) GetProvingStatusByID(hash string) (ProvingStatus, error) {
	row := o.db.QueryRow(`SELECT proving_status FROM block_batch WHERE hash = $1`, hash)
	var status ProvingStatus
	if err := row.Scan(&status); err != nil {
		return ProvingStatusUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) GetVerifiedProofAndInstanceByID(hash string) ([]byte, []byte, error) {
	var proof []byte
	var instance []byte
	row := o.db.QueryRow(`SELECT proof, instance_commitments FROM block_batch WHERE hash = $1 and proving_status = $2`, hash, ProvingTaskVerified)

	if err := row.Scan(&proof, &instance); err != nil {
		return nil, nil, err
	}
	return proof, instance, nil
}

func (o *blockBatchOrm) UpdateProofByID(ctx context.Context, hash string, proof, instanceCommitments []byte, proofTimeSec uint64) error {
	db := o.db
	if _, err := db.ExecContext(ctx,
		db.Rebind(`UPDATE block_batch set proof = ?, instance_commitments = ?, proof_time_sec = ? where hash = ?;`),
		proof, instanceCommitments, proofTimeSec, hash,
	); err != nil {
		log.Error("failed to update proof", "err", err)
	}
	return nil
}

func (o *blockBatchOrm) UpdateProvingStatus(hash string, status ProvingStatus) error {
	switch status {
	case ProvingTaskAssigned:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ?, prover_assigned_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	case ProvingTaskUnassigned:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ?, prover_assigned_at = null where hash = ?;"), status, hash)
		return err
	case ProvingTaskProved, ProvingTaskVerified:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ?, proved_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ? where hash = ?;"), status, hash)
		return err
	}
}

func (o *blockBatchOrm) ResetProvingStatusFor(before ProvingStatus) error {
	_, err := o.db.Exec(o.db.Rebind("update block_batch set proving_status = ? where proving_status = ?;"), ProvingTaskUnassigned, before)
	return err
}

func (o *blockBatchOrm) NewBatchInDBTx(dbTx *sqlx.Tx, startBlock *BlockInfo, endBlock *BlockInfo, parentHash string, totalTxNum uint64, totalL2Gas uint64) (string, error) {
	row := dbTx.QueryRow("SELECT COALESCE(MAX(index), 0) FROM block_batch;")

	// TODO: use *big.Int for this
	var index int64
	if err := row.Scan(&index); err != nil {
		return "", err
	}

	index++
	hash := utils.ComputeBatchID(common.HexToHash(endBlock.Hash), common.HexToHash(parentHash), big.NewInt(index))
	if _, err := dbTx.NamedExec(`INSERT INTO public.block_batch (hash, index, parent_hash, start_block_number, start_block_hash, end_block_number, end_block_hash, total_tx_num, total_l2_gas, state_root, total_l1_tx_num) VALUES (:hash, :index, :parent_hash, :start_block_number, :start_block_hash, :end_block_number, :end_block_hash, :total_tx_num, :total_l2_gas, :state_root, :total_l1_tx_num)`,
		map[string]interface{}{
			"hash":               hash,
			"index":              index,
			"parent_hash":        parentHash,
			"start_block_number": startBlock.Number,
			"start_block_hash":   startBlock.Hash,
			"end_block_number":   endBlock.Number,
			"end_block_hash":     endBlock.Hash,
			"total_tx_num":       totalTxNum,
			"total_l2_gas":       totalL2Gas,
			"state_root":         "state_root_ph", // TODO: add state_root
			"total_l1_tx_num":    0,               // TODO: add total_l1_tx_num
			"created_at":         time.Now(),
			// "proving_status":     ProvingTaskUnassigned, // actually no need, because we have default value in DB schema
			// "rollup_status":      RollupPending,         // actually no need, because we have default value in DB schema
		}); err != nil {
		return "", err
	}

	return hash, nil
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
	rows, err := o.db.Queryx(`SELECT hash FROM block_batch WHERE rollup_status = $1 ORDER BY index ASC LIMIT $2`, RollupPending, limit)
	if err != nil {
		return nil, err
	}

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

	return hashes, rows.Close()
}

func (o *blockBatchOrm) GetLatestBatch() (*BlockBatch, error) {
	row := o.db.QueryRowx(`select * from block_batch where index = (select max(index) from block_batch);`)
	batch := &BlockBatch{}
	if err := row.StructScan(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (o *blockBatchOrm) GetLatestCommittedBatch() (*BlockBatch, error) {
	row := o.db.QueryRowx(`select * from block_batch where index = (select max(index) from block_batch where rollup_status = $1);`, RollupCommitted)
	batch := &BlockBatch{}
	if err := row.StructScan(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (o *blockBatchOrm) GetLatestFinalizedBatch() (*BlockBatch, error) {
	row := o.db.QueryRowx(`select * from block_batch where index = (select max(index) from block_batch where rollup_status = $1);`, RollupFinalized)
	batch := &BlockBatch{}
	if err := row.StructScan(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

func (o *blockBatchOrm) GetCommittedBatches(limit uint64) ([]string, error) {
	rows, err := o.db.Queryx(`SELECT hash FROM block_batch WHERE rollup_status = $1 ORDER BY index ASC LIMIT $2`, RollupCommitted, limit)
	if err != nil {
		return nil, err
	}

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

	return hashes, rows.Close()
}

func (o *blockBatchOrm) GetRollupStatus(hash string) (RollupStatus, error) {
	row := o.db.QueryRow(`SELECT rollup_status FROM block_batch WHERE hash = $1`, hash)
	var status RollupStatus
	if err := row.Scan(&status); err != nil {
		return RollupUndefined, err
	}
	return status, nil
}

func (o *blockBatchOrm) GetRollupStatusByIDList(hashes []string) ([]RollupStatus, error) {
	if len(hashes) == 0 {
		return make([]RollupStatus, 0), nil
	}

	query, args, err := sqlx.In("SELECT hash, rollup_status FROM block_batch WHERE hash IN (?);", hashes)
	if err != nil {
		return make([]RollupStatus, 0), err
	}
	// sqlx.In returns queries with the `?` bindvar, we can rebind it for our backend
	query = o.db.Rebind(query)

	rows, err := o.db.Query(query, args...)

	statusMap := make(map[string]RollupStatus)
	for rows.Next() {
		var hash string
		var status RollupStatus
		if err = rows.Scan(&hash, &status); err != nil {
			break
		}
		statusMap[hash] = status
	}
	var statuses []RollupStatus
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

func (o *blockBatchOrm) UpdateRollupStatus(ctx context.Context, hash string, status RollupStatus) error {
	switch status {
	case RollupCommitted:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ?, committed_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	case RollupFinalized:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ?, finalized_at = ? where hash = ?;"), status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ? where hash = ?;"), status, hash)
		return err
	}
}

func (o *blockBatchOrm) UpdateCommitTxHashAndRollupStatus(ctx context.Context, hash string, commitTxHash string, status RollupStatus) error {
	switch status {
	case RollupCommitted:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set commit_tx_hash = ?, rollup_status = ?, committed_at = ? where hash = ?;"), commitTxHash, status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set commit_tx_hash = ?, rollup_status = ? where hash = ?;"), commitTxHash, status, hash)
		return err
	}
}

func (o *blockBatchOrm) UpdateFinalizeTxHashAndRollupStatus(ctx context.Context, hash string, finalizeTxHash string, status RollupStatus) error {
	switch status {
	case RollupFinalized:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set finalize_tx_hash = ?, rollup_status = ?, finalized_at = ? where hash = ?;"), finalizeTxHash, status, time.Now(), hash)
		return err
	default:
		_, err := o.db.Exec(o.db.Rebind("update block_batch set finalize_tx_hash = ?, rollup_status = ? where hash = ?;"), finalizeTxHash, status, hash)
		return err
	}
}

func (o *blockBatchOrm) GetAssignedBatchIDs() ([]string, error) {
	rows, err := o.db.Queryx(`SELECT hash FROM block_batch WHERE proving_status IN ($1, $2)`, ProvingTaskAssigned, ProvingTaskProved)
	if err != nil {
		return nil, err
	}

	var ids []string
	for rows.Next() {
		var hash string
		if err = rows.Scan(&hash); err != nil {
			break
		}
		ids = append(ids, hash)
	}

	return ids, rows.Close()
}

func (o *blockBatchOrm) UpdateSkippedBatches() (int64, error) {
	res, err := o.db.Exec(o.db.Rebind("update block_batch set rollup_status = ? where (proving_status = ? or proving_status = ?) and rollup_status = ?;"), RollupFinalizationSkipped, ProvingTaskSkipped, ProvingTaskFailed, RollupCommitted)
	if err != nil {
		return 0, err
	}

	count, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, nil
}
