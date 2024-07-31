// Package types defines the table schema data structure used in the database tables
package types

import (
	"fmt"
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

	// GasOracleImportedFailed represents the gas oracle status is imported failed
	GasOracleImportedFailed
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
	case GasOracleImportedFailed:
		return "GasOracleImportedFailed"
	default:
		return fmt.Sprintf("Undefined GasOracleStatus (%d)", int32(s))
	}
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

// ProverProveStatus is the prover prove status of a block batch (session)
type ProverProveStatus int32

const (
	// ProverProveStatusUndefined indicates an unknown prover proving status
	ProverProveStatusUndefined ProverProveStatus = iota
	// ProverAssigned indicates prover assigned but has not submitted proof
	ProverAssigned
	// ProverProofValid indicates prover has submitted valid proof
	ProverProofValid
	// ProverProofInvalid indicates prover has submitted invalid proof
	ProverProofInvalid
)

func (s ProverProveStatus) String() string {
	switch s {
	case ProverAssigned:
		return "ProverAssigned"
	case ProverProofValid:
		return "ProverProofValid"
	case ProverProofInvalid:
		return "ProverProofInvalid"
	default:
		return fmt.Sprintf("Bad Value: %d", int32(s))
	}
}

// ProverTaskFailureType the type of prover task failure
type ProverTaskFailureType int

const (
	// ProverTaskFailureTypeUndefined indicates an unknown prover failure type
	ProverTaskFailureTypeUndefined ProverTaskFailureType = iota
	// ProverTaskFailureTypeTimeout prover task failure of timeout
	ProverTaskFailureTypeTimeout
	// ProverTaskFailureTypeSubmitStatusNotOk prover task failure of submit status not ok
	ProverTaskFailureTypeSubmitStatusNotOk
	// ProverTaskFailureTypeVerifiedFailed prover task failure of verified failed by coordinator
	ProverTaskFailureTypeVerifiedFailed
	// ProverTaskFailureTypeServerError collect occur error
	ProverTaskFailureTypeServerError
	// ProverTaskFailureTypeObjectAlreadyVerified object(batch/chunk) already verified, may exists in test env when ENABLE_TEST_ENV_BYPASS_FEATURES is true
	ProverTaskFailureTypeObjectAlreadyVerified
	// ProverTaskFailureTypeReassignedByAdmin reassigned by admin, this value is used in admin-system and defined here for clarity
	ProverTaskFailureTypeReassignedByAdmin
)

func (r ProverTaskFailureType) String() string {
	switch r {
	case ProverTaskFailureTypeUndefined:
		return "prover task failure undefined"
	case ProverTaskFailureTypeTimeout:
		return "prover task failure timeout"
	case ProverTaskFailureTypeSubmitStatusNotOk:
		return "prover task failure validated submit proof status not ok"
	case ProverTaskFailureTypeVerifiedFailed:
		return "prover task failure verified failed"
	case ProverTaskFailureTypeServerError:
		return "prover task failure server exception"
	case ProverTaskFailureTypeObjectAlreadyVerified:
		return "prover task failure object already verified"
	case ProverTaskFailureTypeReassignedByAdmin:
		return "prover task failure reassigned by admin"
	default:
		return fmt.Sprintf("illegal prover task failure type (%d)", int32(r))
	}
}

// ProvingStatus block_batch proving_status (unassigned, assigned, proved, verified, submitted)
type ProvingStatus int

const (
	// ProvingStatusUndefined : undefined proving_task status
	ProvingStatusUndefined ProvingStatus = iota
	// ProvingTaskUnassigned : proving_task is not assigned to be proved
	ProvingTaskUnassigned
	// ProvingTaskAssigned : proving_task is assigned to be proved
	ProvingTaskAssigned
	// ProvingTaskProvedDEPRECATED DEPRECATED: proof has been returned by prover
	ProvingTaskProvedDEPRECATED
	// ProvingTaskVerified : proof is valid
	ProvingTaskVerified
	// ProvingTaskFailed : fail to generate proof
	ProvingTaskFailed
)

func (ps ProvingStatus) String() string {
	switch ps {
	case ProvingTaskUnassigned:
		return "unassigned"
	case ProvingTaskAssigned:
		return "assigned"
	case ProvingTaskProvedDEPRECATED:
		return "proved"
	case ProvingTaskVerified:
		return "verified"
	case ProvingTaskFailed:
		return "failed"
	default:
		return fmt.Sprintf("Undefined ProvingStatus (%d)", int32(ps))
	}
}

// ChunkProofsStatus describes the proving status of chunks that belong to a batch.
type ChunkProofsStatus int

const (
	// ChunkProofsStatusUndefined represents an undefined chunk proofs status
	ChunkProofsStatusUndefined ChunkProofsStatus = iota

	// ChunkProofsStatusPending means that some chunks that belong to this batch have not been proven
	ChunkProofsStatusPending

	// ChunkProofsStatusReady means that all chunks that belong to this batch have been proven
	ChunkProofsStatusReady
)

func (s ChunkProofsStatus) String() string {
	switch s {
	case ChunkProofsStatusPending:
		return "ChunkProofsStatusPending"
	case ChunkProofsStatusReady:
		return "ChunkProofsStatusReady"
	default:
		return fmt.Sprintf("Undefined ChunkProofsStatus (%d)", int32(s))
	}
}

// BatchProofsStatus describes the proving status of batches that belong to a bundle.
type BatchProofsStatus int

const (
	// BatchProofsStatusUndefined represents an undefined batch proofs status
	BatchProofsStatusUndefined BatchProofsStatus = iota

	// BatchProofsStatusPending means that some batches that belong to this bundle have not been proven
	BatchProofsStatusPending

	// BatchProofsStatusReady means that all batches that belong to this bundle have been proven
	BatchProofsStatusReady
)

func (s BatchProofsStatus) String() string {
	switch s {
	case BatchProofsStatusPending:
		return "BatchProofsStatusPending"
	case BatchProofsStatusReady:
		return "BatchProofsStatusReady"
	default:
		return fmt.Sprintf("Undefined BatchProofsStatus (%d)", int32(s))
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
	case RollupCommitFailed:
		return "RollupCommitFailed"
	case RollupFinalizeFailed:
		return "RollupFinalizeFailed"
	default:
		return fmt.Sprintf("Undefined RollupStatus (%d)", int32(s))
	}
}

// SenderType defines the various types of senders sending the transactions.
type SenderType int

const (
	// SenderTypeUnknown indicates an unknown sender type.
	SenderTypeUnknown SenderType = iota
	// SenderTypeCommitBatch indicates the sender is responsible for committing batches.
	SenderTypeCommitBatch
	// SenderTypeFinalizeBatch indicates the sender is responsible for finalizing batches.
	SenderTypeFinalizeBatch
	// SenderTypeL1GasOracle indicates a sender from L2 responsible for updating L1 gas prices.
	SenderTypeL1GasOracle
	// SenderTypeL2GasOracle indicates a sender from L1 responsible for updating L2 gas prices.
	SenderTypeL2GasOracle
)

// String returns a string representation of the SenderType.
func (t SenderType) String() string {
	switch t {
	case SenderTypeCommitBatch:
		return "SenderTypeCommitBatch"
	case SenderTypeFinalizeBatch:
		return "SenderTypeFinalizeBatch"
	case SenderTypeL1GasOracle:
		return "SenderTypeL1GasOracle"
	case SenderTypeL2GasOracle:
		return "SenderTypeL2GasOracle"
	default:
		return fmt.Sprintf("Unknown SenderType (%d)", int32(t))
	}
}

// TxStatus represents the current status of a transaction in the transaction lifecycle.
type TxStatus int

const (
	// TxStatusUnknown represents an undefined status of the transaction.
	TxStatusUnknown TxStatus = iota
	// TxStatusPending indicates that the transaction is yet to be processed.
	TxStatusPending
	// TxStatusReplaced indicates that the transaction has been replaced by another one, typically due to a higher gas price.
	TxStatusReplaced
	// TxStatusConfirmed indicates that the transaction has been successfully processed and confirmed.
	TxStatusConfirmed
	// TxStatusConfirmedFailed indicates that the transaction has failed during processing.
	TxStatusConfirmedFailed
)

func (s TxStatus) String() string {
	switch s {
	case TxStatusPending:
		return "TxStatusPending"
	case TxStatusReplaced:
		return "TxStatusReplaced"
	case TxStatusConfirmed:
		return "TxStatusConfirmed"
	case TxStatusConfirmedFailed:
		return "TxStatusConfirmedFailed"
	default:
		return fmt.Sprintf("Unknown TxStatus (%d)", int32(s))
	}
}
