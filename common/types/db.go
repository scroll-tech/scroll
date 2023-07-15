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
		return fmt.Sprintf("Undefined (%d)", int32(s))
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

// RollerProveStatus is the roller prove status of a block batch (session)
type RollerProveStatus int32

const (
	// RollerProveStatusUndefined indicates an unknown roller proving status
	RollerProveStatusUndefined RollerProveStatus = iota
	// RollerAssigned indicates roller assigned but has not submitted proof
	RollerAssigned
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

// RollerFailureType is the type of a roller session's failure
type RollerFailureType int

const (
	// RollerFailureTypeUndefined indicates an unknown roller failure type
	RollerFailureTypeUndefined RollerFailureType = iota
)

func (s RollerFailureType) String() string {
	switch s {
	default:
		return fmt.Sprintf("Undefined (%d)", int32(s))
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
	case ProvingTaskAssigned:
		return "assigned"
	case ProvingTaskProved:
		return "proved"
	case ProvingTaskVerified:
		return "verified"
	case ProvingTaskFailed:
		return "failed"
	default:
		return fmt.Sprintf("Undefined (%d)", int32(ps))
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
		return fmt.Sprintf("Undefined (%d)", int32(s))
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
		return fmt.Sprintf("Undefined (%d)", int32(s))
	}
}
