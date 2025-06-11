package types

import (
	"fmt"

	"scroll-tech/common/types/message"
)

// RespStatus represents status code from prover to scroll
type RespStatus uint32

const (
	// StatusOk means generate proof success
	StatusOk RespStatus = iota
	// StatusProofError means generate proof failed
	StatusProofError
)

// ProverType represents the type of prover.
type ProverType uint8

func (r ProverType) String() string {
	switch r {
	case ProverTypeChunkDeprecated:
		return "prover type chunk (deprecated)"
	case ProverTypeBatchDeprecated:
		return "prover type batch (deprecated)"
	case ProverTypeOpenVM:
		return "prover type openvm"
	default:
		return fmt.Sprintf("illegal prover type: %d", r)
	}
}

const (
	// ProverTypeUndefined is an unknown prover type
	ProverTypeUndefined ProverType = iota
	// ProverTypeChunk signals it's a chunk prover, which can prove chunk_tasks, which is deprecated
	ProverTypeChunkDeprecated
	// ProverTypeBatch signals it's a batch prover, which can prove batch_tasks and bundle_tasks, which is deprecated
	ProverTypeBatchDeprecated
	// ProverTypeOpenVM
	ProverTypeOpenVM
)

// MakeProverType make ProverType from ProofType
func MakeProverType(proofType message.ProofType) ProverType {
	switch proofType {
	case message.ProofTypeChunk:
		return ProverTypeChunkDeprecated
	case message.ProofTypeBatch, message.ProofTypeBundle:
		return ProverTypeBatchDeprecated
	default:
		return ProverTypeUndefined
	}
}

// ProverProviderType represents the type of prover provider.
type ProverProviderType uint8

func (r ProverProviderType) String() string {
	switch r {
	case ProverProviderTypeInternal:
		return "prover provider type internal"
	case ProverProviderTypeExternal:
		return "prover provider type external"
	default:
		return fmt.Sprintf("prover provider type: %d", r)
	}
}

const (
	// ProverProviderTypeUndefined is an unknown prover provider type
	ProverProviderTypeUndefined ProverProviderType = iota
	// ProverProviderTypeInternal is an internal prover provider type
	ProverProviderTypeInternal
	// ProverProviderTypeExternal is an external prover provider type
	ProverProviderTypeExternal
)
