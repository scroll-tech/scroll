package types

import (
	"fmt"

	"scroll-tech/common/types/message"
)

// ProverType represents the type of prover.
type ProverType uint8

func (r ProverType) String() string {
	switch r {
	case ProverTypeChunk:
		return "prover type chunk"
	case ProverTypeBatch:
		return "prover type batch"
	default:
		return fmt.Sprintf("illegal prover type: %d", r)
	}
}

const (
	// ProverTypeUndefined is an unknown prover type
	ProverTypeUndefined ProverType = iota
	// ProverTypeChunk signals it's a chunk prover, which can prove chunk_tasks
	ProverTypeChunk
	// ProverTypeBatch signals it's a batch prover, which can prove batch_tasks and bundle_tasks
	ProverTypeBatch
)

// MakeProverType make ProverType from ProofType
func MakeProverType(proofType message.ProofType) ProverType {
	switch proofType {
	case message.ProofTypeChunk:
		return ProverTypeChunk
	case message.ProofTypeBatch, message.ProofTypeBundle:
		return ProverTypeBatch
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
