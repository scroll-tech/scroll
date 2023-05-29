// Package types defines the table schema data structure used in the database tables
package types

import (
	"database/sql"
	"fmt"
	"time"

	"scroll-tech/common/types/message"
)

// L1BlockStatus represents current l1 block processing status
type L1BlockStatus int

// GasOracleStatus represents current gas oracle processing status
type GasOracleStatus int

const (
	// L1BlockUndefined : undefined l1 block status
	L1BlockUndefined L1BlockStatus = iota

	// L1BlockPending represents the l1 block status is pending
	L1BlockPending

	// L1BlockImporting represents the l1 block status is importing
	L1BlockImporting

	// L1BlockImported represents the l1 block status is imported
	L1BlockImported

	// L1BlockFailed represents the l1 block status is failed
	L1BlockFailed
)

const (
	// GasOracleUndefined : undefined gas oracle status
	GasOracleUndefined GasOracleStatus = iota

	// GasOraclePending represents the gas oracle status is pending
	GasOraclePending

	// GasOracleImporting represents the gas oracle status is importing
	GasOracleImporting

	// GasOracleImported represents the gas oracle status is imported
	GasOracleImported

	// GasOracleFailed represents the gas oracle status is failed
	GasOracleFailed
)

// L1BlockInfo is structure of stored l1 block
type L1BlockInfo struct {
	Number    uint64 `json:"number" db:"number"`
	Hash      string `json:"hash" db:"hash"`
	HeaderRLP string `json:"header_rlp" db:"header_rlp"`
	BaseFee   uint64 `json:"base_fee" db:"base_fee"`

	BlockStatus     L1BlockStatus   `json:"block_status" db:"block_status"`
	GasOracleStatus GasOracleStatus `json:"oracle_status" db:"oracle_status"`

	ImportTxHash sql.NullString `json:"import_tx_hash" db:"import_tx_hash"`
	OracleTxHash sql.NullString `json:"oracle_tx_hash" db:"oracle_tx_hash"`
}

// MsgStatus represents current layer1 transaction processing status
type MsgStatus int

const (
	// MsgUndefined : undefined msg status
	MsgUndefined MsgStatus = iota

	// MsgPending represents the from_layer message status is pending
	MsgPending

	// MsgSubmitted represents the from_layer message status is submitted
	MsgSubmitted

	// MsgConfirmed represents the from_layer message status is confirmed
	MsgConfirmed

	// MsgFailed represents the from_layer message status is failed
	MsgFailed

	// MsgExpired represents the from_layer message status is expired
	MsgExpired

	// MsgRelayFailed represents the from_layer message status is relay failed
	MsgRelayFailed
)

// L1Message is structure of stored layer1 bridge message
type L1Message struct {
	QueueIndex uint64    `json:"queue_index" db:"queue_index"`
	MsgHash    string    `json:"msg_hash" db:"msg_hash"`
	Height     uint64    `json:"height" db:"height"`
	Sender     string    `json:"sender" db:"sender"`
	Value      string    `json:"value" db:"value"`
	Target     string    `json:"target" db:"target"`
	Calldata   string    `json:"calldata" db:"calldata"`
	GasLimit   uint64    `json:"gas_limit" db:"gas_limit"`
	Layer1Hash string    `json:"layer1_hash" db:"layer1_hash"`
	Status     MsgStatus `json:"status" db:"status"`
}

// L2Message is structure of stored layer2 bridge message
type L2Message struct {
	Nonce      uint64    `json:"nonce" db:"nonce"`
	MsgHash    string    `json:"msg_hash" db:"msg_hash"`
	Height     uint64    `json:"height" db:"height"`
	Sender     string    `json:"sender" db:"sender"`
	Value      string    `json:"value" db:"value"`
	Target     string    `json:"target" db:"target"`
	Calldata   string    `json:"calldata" db:"calldata"`
	Layer2Hash string    `json:"layer2_hash" db:"layer2_hash"`
	Status     MsgStatus `json:"status" db:"status"`
}

// BlockInfo is structure of stored `block_trace` without `trace`
type BlockInfo struct {
	Number         uint64         `json:"number" db:"number"`
	Hash           string         `json:"hash" db:"hash"`
	ParentHash     string         `json:"parent_hash" db:"parent_hash"`
	BatchHash      sql.NullString `json:"batch_hash" db:"batch_hash"`
	TxNum          uint64         `json:"tx_num" db:"tx_num"`
	GasUsed        uint64         `json:"gas_used" db:"gas_used"`
	BlockTimestamp uint64         `json:"block_timestamp" db:"block_timestamp"`
}

// RollerProveStatus is the roller prove status of a block batch (session)
type RollerProveStatus int32

const (
	// RollerAssigned indicates roller assigned but has not submitted proof
	RollerAssigned RollerProveStatus = iota
	// RollerProofValid indicates roller has submitted valid proof
	RollerProofValid
	// RollerProofInvalid indicates roller has submitted invalid proof
	RollerProofInvalid
)

func (s RollerProveStatus) String() string {
	switch s {
	case RollerAssigned:
		return "RollerAssigned"
	case RollerProofValid:
		return "RollerProofValid"
	case RollerProofInvalid:
		return "RollerProofInvalid"
	default:
		return fmt.Sprintf("Bad Value: %d", int32(s))
	}
}

// RollerStatus is the roller name and roller prove status
type RollerStatus struct {
	PublicKey string            `json:"public_key"`
	Name      string            `json:"name"`
	Status    RollerProveStatus `json:"status"`
}

// SessionInfo is assigned rollers info of a block batch (session)
type SessionInfo struct {
	ID             string                   `json:"id"`
	Rollers        map[string]*RollerStatus `json:"rollers"`
	StartTimestamp int64                    `json:"start_timestamp"`
	Attempts       uint8                    `json:"attempts,omitempty"`
	ProveType      message.ProveType        `json:"prove_type,omitempty"`
}

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

// RollupStatus block_batch rollup_status (pending, committing, committed, commit_failed, finalizing, finalized, finalize_skipped, finalize_failed)
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
	// RollupCommitFailed : rollup commit transaction confirmed but failed
	RollupCommitFailed
	// RollupFinalizeFailed : rollup finalize transaction is confirmed but failed
	RollupFinalizeFailed
)

// BlockBatch is structure of stored block_batch
type BlockBatch struct {
	Hash             string          `json:"hash" db:"hash"`
	Index            uint64          `json:"index" db:"index"`
	ParentHash       string          `json:"parent_hash" db:"parent_hash"`
	StartBlockNumber uint64          `json:"start_block_number" db:"start_block_number"`
	StartBlockHash   string          `json:"start_block_hash" db:"start_block_hash"`
	EndBlockNumber   uint64          `json:"end_block_number" db:"end_block_number"`
	EndBlockHash     string          `json:"end_block_hash" db:"end_block_hash"`
	StateRoot        string          `json:"state_root" db:"state_root"`
	TotalTxNum       uint64          `json:"total_tx_num" db:"total_tx_num"`
	TotalL1TxNum     uint64          `json:"total_l1_tx_num" db:"total_l1_tx_num"`
	TotalL2Gas       uint64          `json:"total_l2_gas" db:"total_l2_gas"`
	ProvingStatus    ProvingStatus   `json:"proving_status" db:"proving_status"`
	Proof            []byte          `json:"proof" db:"proof"`
	ProofTimeSec     uint64          `json:"proof_time_sec" db:"proof_time_sec"`
	RollupStatus     RollupStatus    `json:"rollup_status" db:"rollup_status"`
	OracleStatus     GasOracleStatus `json:"oracle_status" db:"oracle_status"`
	CommitTxHash     sql.NullString  `json:"commit_tx_hash" db:"commit_tx_hash"`
	FinalizeTxHash   sql.NullString  `json:"finalize_tx_hash" db:"finalize_tx_hash"`
	OracleTxHash     sql.NullString  `json:"oracle_tx_hash" db:"oracle_tx_hash"`
	CreatedAt        *time.Time      `json:"created_at" db:"created_at"`
	ProverAssignedAt *time.Time      `json:"prover_assigned_at" db:"prover_assigned_at"`
	ProvedAt         *time.Time      `json:"proved_at" db:"proved_at"`
	CommittedAt      *time.Time      `json:"committed_at" db:"committed_at"`
	FinalizedAt      *time.Time      `json:"finalized_at" db:"finalized_at"`
}

// AggTask is a wrapper type around db AggProveTask type.
type AggTask struct {
	ID              string        `json:"id" db:"id"`
	StartBatchIndex uint64        `json:"start_batch_index" db:"start_batch_index"`
	StartBatchHash  string        `json:"start_batch_hash" db:"start_batch_hash"`
	EndBatchIndex   uint64        `json:"end_batch_index" db:"end_batch_index"`
	EndBatchHash    string        `json:"end_batch_hash" db:"end_batch_hash"`
	ProvingStatus   ProvingStatus `json:"proving_status" db:"proving_status"`
	Proof           []byte        `json:"proof" db:"proof"`
	CreatedAt       *time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt       *time.Time    `json:"updated_at" db:"updated_at"`
}
