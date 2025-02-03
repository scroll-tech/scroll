package euclid

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

// ChunkTaskDetail is a type containing informations require to build ChunkTask.
type ChunkTaskDetail struct {
	BlockHashes []common.Hash `json:"block_hashes"`
}

// BatchTask
type BatchTask struct {
	ChunkProofs []*ChunkProof `json:"chunk_proofs"`
	BatchHeader interface{}   `json:"batch_header"`
	BlobBytes   []byte        `json:"blob_bytes"`
}

// BundleTask
type BundleTask struct {
	BatchProofs []*BatchProof `json:"batch_proofs"`
}

// Proof for flatten VM proof
type VmProof struct {
	Proof        interface{} `json:"proofs"`
	PublicValues []uint32    `json:"public_values"`
	Vk           []byte      `json:"vk,omitempty"`
}

// Proof for flatten EVM proof
type EVmProof struct {
	Proof     []byte     `json:"proof"`
	Instances [][]string `json:"instances"`
	Vk        []byte     `json:"vk,omitempty"`
}

// ChunkInfo is for calculating pi_hash for chunk
type ChunkInfo struct {
	ChainID       uint64      `json:"chain_id"`
	PrevStateRoot common.Hash `json:"prev_state_root"`
	PostStateRoot common.Hash `json:"post_state_root"`
	WithdrawRoot  common.Hash `json:"withdraw_root"`
	DataHash      common.Hash `json:"data_hash"`
	TxBytesHash   common.Hash `json:"tx_data_digest"`
}

// ChunkProof includes the proof info that are required for chunk verification and rollup.
type ChunkProof struct {
	MetaData struct {
		ChunkInfo *ChunkInfo `json:"chunk_info"`
	} `json:"metadata"`

	Proof      *VmProof `json:"proof"`
	GitVersion string   `json:"git_version,omitempty"`
}

// BatchInfo is for calculating pi_hash for batch header
type BatchInfo struct {
	ParentBatchHash common.Hash `json:"parent_batch_hash"`
	ParentStateRoot common.Hash `json:"parent_state_root"`
	StateRoot       common.Hash `json:"state_root"`
	WithdrawRoot    common.Hash `json:"withdraw_root"`
	BatchHash       common.Hash `json:"batch_hash"`
	ChainID         uint64      `json:"chain_id"`
}

// BatchProof includes the proof info that are required for batch verification and rollup.
type BatchProof struct {
	MetaData struct {
		BatchInfo *BatchInfo  `json:"batch_info"`
		BatchHash common.Hash `json:"batch_hash"`
	} `json:"metadata"`

	Proof      *VmProof `json:"proof"`
	GitVersion string   `json:"git_version,omitempty"`
}

// SanityCheck checks whether a BatchProof is in a legal format
func (ap *BatchProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}
	if ap.MetaData.BatchInfo == nil {
		return errors.New("batch info not ready")
	}

	if ap.Proof == nil {
		return errors.New("proof not ready")
	} else {
		pf := ap.Proof
		if pf.Proof == nil {
			return errors.New("proof data not ready")
		}

		// TODO: resume this after rc2
		// if len(pf.Vk) == 0 {
		// 	return errors.New("vk not ready")
		// }
	}

	return nil
}

// BundleInfo is for calculating pi_hash for bundle header
type BundleInfo struct {
	ChainID       uint64      `json:"chain_id"`
	PrevStateRoot common.Hash `json:"prev_state_root"`
	PostStateRoot common.Hash `json:"post_state_root"`
	WithdrawRoot  common.Hash `json:"withdraw_root"`
	NumBatches    uint32      `json:"num_batches"`
	PrevBatchHash common.Hash `json:"prev_batch_hash"`
	BatchHash     common.Hash `json:"batch_hash"`
}

// BundleProof includes the proof info that are required for verification of a bundle of batch proofs.
type BundleProof struct {
	MetaData struct {
		BundleInfo    *BundleInfo `json:"bundle_info"`
		BunndlePIHash common.Hash `json:"bundle_pi_hash"`
	} `json:"metadata"`

	Proof      *EVmProof `json:"proof"`
	GitVersion string    `json:"git_version,omitempty"`
}

// SanityCheck checks whether a BundleProof is in a legal format
func (ap *BundleProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}
	// TODO: resume this after metadata of bundle info is ready
	// if ap.MetaData.BundleInfo == nil {
	// 	return errors.New("bundle info not ready")
	// }

	if ap.Proof == nil {
		return errors.New("proof not ready")
	} else {
		pf := ap.Proof
		if len(pf.Proof)%32 != 0 {
			return fmt.Errorf("proof buffer length must be a multiple of 32, got: %d", len(pf.Proof))
		}

		if len(pf.Instances) == 0 {
			return errors.New("instance not ready")
		}

		// TODO: resume this after rc2
		// if len(pf.Vk) == 0 {
		// 	return errors.New("vk not ready")
		// }
	}

	return nil
}
