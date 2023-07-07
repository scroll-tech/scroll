// Package types defines the table schema data structure used in the database tables
package types

import (
	"database/sql"
	"fmt"
)

// L1BlockStatus represents current l1 block processing status
type L1BlockStatus int

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

// GasOracleStatus represents current gas oracle processing status
type GasOracleStatus int

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

func (s GasOracleStatus) String() string {
	switch s {
	case GasOracleUndefined:
		return "GasOracleUndefined"
	case GasOraclePending:
		return "GasOraclePending"
	case GasOracleImporting:
		return "GasOracleImporting"
	case GasOracleImported:
		return "GasOracleImported"
	case GasOracleFailed:
		return "GasOracleFailed"
	default:
		return fmt.Sprintf("Bad Value: %d", int32(s))
	}
}

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

func (s RollupStatus) String() string {
	switch s {
	case RollupPending:
		return "RollupPending"
	case RollupCommitting:
		return "RollupCommitting"
	case RollupCommitted:
		return "RollupCommitted"
	case RollupFinalizing:
		return "RollupFinalizing"
	case RollupFinalized:
		return "RollupFinalized"
	case RollupFinalizationSkipped:
		return "RollupFinalizationSkipped"
	case RollupCommitFailed:
		return "RollupCommitFailed"
	case RollupFinalizeFailed:
		return "RollupFinalizeFailed"
	default:
		return "undefined"
	}
}
