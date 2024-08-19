package message

import (
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
)

// RespStatus represents status code from prover to scroll
type RespStatus uint32

const (
	// StatusOk means generate proof success
	StatusOk RespStatus = iota
	// StatusProofError means generate proof failed
	StatusProofError
)

// ProofType represents the type of task.
type ProofType uint8

func (r ProofType) String() string {
	switch r {
	case ProofTypeChunk:
		return "proof type chunk"
	case ProofTypeBatch:
		return "proof type batch"
	case ProofTypeBundle:
		return "proof type bundle"
	default:
		return fmt.Sprintf("illegal proof type: %d", r)
	}
}

const (
	// ProofTypeUndefined is an unknown proof type
	ProofTypeUndefined ProofType = iota
	// ProofTypeChunk generates a proof for a ZkEvm chunk, where the inputs are the execution traces for blocks contained in the chunk. ProofTypeChunk is the default proof type.
	ProofTypeChunk
	// ProofTypeBatch generates zk proof from chunk proofs
	ProofTypeBatch
	// ProofTypeBundle generates zk proof from batch proofs
	ProofTypeBundle
)

// ChunkTaskDetail is a type containing ChunkTask detail.
type ChunkTaskDetail struct {
	BlockHashes []common.Hash `json:"block_hashes"`
}

// BatchTaskDetail is a type containing BatchTask detail.
type BatchTaskDetail struct {
	ChunkInfos  []*ChunkInfo  `json:"chunk_infos"`
	ChunkProofs []*ChunkProof `json:"chunk_proofs"`
	BatchHeader interface{}   `json:"batch_header"`
	BlobBytes   []byte        `json:"blob_bytes"`
}

// BundleTaskDetail consists of all the information required to describe the task to generate a proof for a bundle of batches.
type BundleTaskDetail struct {
	BatchProofs []*BatchProof `json:"batch_proofs"`
}

// ChunkInfo is for calculating pi_hash for chunk
type ChunkInfo struct {
	ChainID       uint64      `json:"chain_id"`
	PrevStateRoot common.Hash `json:"prev_state_root"`
	PostStateRoot common.Hash `json:"post_state_root"`
	WithdrawRoot  common.Hash `json:"withdraw_root"`
	DataHash      common.Hash `json:"data_hash"`
	IsPadding     bool        `json:"is_padding"`
	TxBytes       []byte      `json:"tx_bytes"`
}

// SubCircuitRowUsage tracing info added in v0.11.0rc8
type SubCircuitRowUsage struct {
	Name      string `json:"name"`
	RowNumber uint64 `json:"row_number"`
}

// ChunkProof includes the proof info that are required for chunk verification and rollup.
type ChunkProof struct {
	StorageTrace []byte `json:"storage_trace,omitempty"`
	Protocol     []byte `json:"protocol"`
	Proof        []byte `json:"proof"`
	Instances    []byte `json:"instances"`
	Vk           []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	ChunkInfo  *ChunkInfo           `json:"chunk_info,omitempty"`
	GitVersion string               `json:"git_version,omitempty"`
	RowUsages  []SubCircuitRowUsage `json:"row_usages,omitempty"`
}

// BatchProof includes the proof info that are required for batch verification and rollup.
type BatchProof struct {
	Protocol  []byte `json:"protocol"`
	Proof     []byte `json:"proof"`
	Instances []byte `json:"instances"`
	Vk        []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	BatchHash  common.Hash `json:"batch_hash"`
	GitVersion string      `json:"git_version,omitempty"`
}

// SanityCheck checks whether a BatchProof is in a legal format
func (ap *BatchProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}

	if len(ap.Proof) == 0 {
		return errors.New("proof not ready")
	}

	if len(ap.Proof)%32 != 0 {
		return fmt.Errorf("proof buffer length must be a multiple of 32, got: %d", len(ap.Proof))
	}

	if len(ap.Instances) == 0 {
		return errors.New("instance not ready")
	}

	if len(ap.Vk) == 0 {
		return errors.New("vk not ready")
	}

	return nil
}

// BundleProof includes the proof info that are required for verification of a bundle of batch proofs.
type BundleProof struct {
	Proof     []byte `json:"proof"`
	Instances []byte `json:"instances"`
	Vk        []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	GitVersion string `json:"git_version,omitempty"`
}

// SanityCheck checks whether a BundleProof is in a legal format
func (ap *BundleProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}

	if len(ap.Proof) == 0 {
		return errors.New("proof not ready")
	}

	if len(ap.Proof)%32 != 0 {
		return fmt.Errorf("proof buffer length must be a multiple of 32, got: %d", len(ap.Proof))
	}

	if len(ap.Instances) == 0 {
		return errors.New("instance not ready")
	}

	if len(ap.Vk) == 0 {
		return errors.New("vk not ready")
	}

	return nil
}
