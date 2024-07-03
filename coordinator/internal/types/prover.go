package types

import "fmt"

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
