package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/scroll-tech/go-ethereum/common"
)

const (
	euclidFork = "euclid"
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
	ChunkInfos  []*ChunkInfo `json:"chunk_infos"`
	ChunkProofs []ChunkProof `json:"chunk_proofs"`
	BatchHeader interface{}  `json:"batch_header"`
	BlobBytes   []byte       `json:"blob_bytes"`
}

// BundleTaskDetail consists of all the information required to describe the task to generate a proof for a bundle of batches.
type BundleTaskDetail struct {
	BatchProofs []BatchProof `json:"batch_proofs"`
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
	TxBytesHash   common.Hash `json:"tx_data_digest"`
}

// SubCircuitRowUsage tracing info added in v0.11.0rc8
type SubCircuitRowUsage struct {
	Name      string `json:"name"`
	RowNumber uint64 `json:"row_number"`
}

// ChunkProof
type ChunkProof interface {
	Proof() []byte
}

// NewChunkProof creates a new ChunkProof instance.
func NewChunkProof(hardForkName string) ChunkProof {
	switch hardForkName {
	case euclidFork:
		return &OpenVMChunkProof{}
	default:
		return &Halo2ChunkProof{}
	}
}

// Halo2ChunkProof includes the proof info that are required for chunk verification and rollup.
type Halo2ChunkProof struct {
	StorageTrace []byte `json:"storage_trace,omitempty"`
	Protocol     []byte `json:"protocol"`
	RawProof     []byte `json:"proof"`
	Instances    []byte `json:"instances"`
	Vk           []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	ChunkInfo  *ChunkInfo           `json:"chunk_info,omitempty"`
	GitVersion string               `json:"git_version,omitempty"`
	RowUsages  []SubCircuitRowUsage `json:"row_usages,omitempty"`
}

// Proof returns the proof bytes of a ChunkProof
func (ap *Halo2ChunkProof) Proof() []byte {
	return ap.RawProof
}

// BatchProof
type BatchProof interface {
	SanityCheck() error
	Proof() []byte
}

// NewBatchProof creates a new BatchProof instance.
func NewBatchProof(hardForkName string) BatchProof {
	switch hardForkName {
	case euclidFork:
		return &OpenVMBatchProof{}
	default:
		return &Halo2BatchProof{}
	}
}

// Halo2BatchProof includes the proof info that are required for batch verification and rollup.
type Halo2BatchProof struct {
	Protocol  []byte `json:"protocol"`
	RawProof  []byte `json:"proof"`
	Instances []byte `json:"instances"`
	Vk        []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	BatchHash  common.Hash `json:"batch_hash"`
	GitVersion string      `json:"git_version,omitempty"`
}

// Proof returns the proof bytes of a BatchProof
func (ap *Halo2BatchProof) Proof() []byte {
	return ap.RawProof
}

// SanityCheck checks whether a BatchProof is in a legal format
func (ap *Halo2BatchProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}

	if len(ap.RawProof) == 0 {
		return errors.New("proof not ready")
	}

	if len(ap.RawProof)%32 != 0 {
		return fmt.Errorf("proof buffer length must be a multiple of 32, got: %d", len(ap.RawProof))
	}

	if len(ap.Instances) == 0 {
		return errors.New("instance not ready")
	}

	if len(ap.Vk) == 0 {
		return errors.New("vk not ready")
	}

	return nil
}

// BundleProof
type BundleProof interface {
	SanityCheck() error
	Proof() []byte
}

// NewBundleProof creates a new BundleProof instance.
func NewBundleProof(hardForkName string) BundleProof {
	switch hardForkName {
	case euclidFork:
		return &OpenVMBundleProof{}
	default:
		return &Halo2BundleProof{}
	}
}

// BundleProof includes the proof info that are required for verification of a bundle of batch proofs.
type Halo2BundleProof struct {
	RawProof  []byte `json:"proof"`
	Instances []byte `json:"instances"`
	Vk        []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	GitVersion string `json:"git_version,omitempty"`
}

// Proof returns the proof bytes of a BundleProof
func (ap *Halo2BundleProof) Proof() []byte {
	return ap.RawProof
}

// SanityCheck checks whether a BundleProof is in a legal format
func (ap *Halo2BundleProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}

	if len(ap.RawProof) == 0 {
		return errors.New("proof not ready")
	}

	if len(ap.RawProof)%32 != 0 {
		return fmt.Errorf("proof buffer length must be a multiple of 32, got: %d", len(ap.RawProof))
	}

	if len(ap.Instances) == 0 {
		return errors.New("instance not ready")
	}

	if len(ap.Vk) == 0 {
		return errors.New("vk not ready")
	}

	return nil
}

// Proof for flatten VM proof
type OpenVMProof struct {
	Proof        []byte `json:"proofs"`
	PublicValues []byte `json:"public_values"`
}

// Proof for flatten EVM proof
type OpenVMEvmProof struct {
	Proof     []byte `json:"proof"`
	Instances []byte `json:"instances"`
}

// OpenVMChunkProof includes the proof info that are required for chunk verification and rollup.
type OpenVMChunkProof struct {
	MetaData struct {
		ChunkInfo *ChunkInfo `json:"chunk_info"`
	} `json:"metadata"`

	VmProof    *OpenVMProof `json:"proof"`
	Vk         []byte       `json:"vk,omitempty"`
	GitVersion string       `json:"git_version,omitempty"`
}

func (p *OpenVMChunkProof) Proof() []byte {
	proofJson, err := json.Marshal(p.VmProof)
	if err != nil {
		panic(fmt.Sprint("marshaling error", err))
	}

	return proofJson
}

// OpenVMBatchInfo is for calculating pi_hash for batch header
type OpenVMBatchInfo struct {
	ParentBatchHash common.Hash `json:"parent_batch_hash"`
	ParentStateRoot common.Hash `json:"parent_state_root"`
	StateRoot       common.Hash `json:"state_root"`
	WithdrawRoot    common.Hash `json:"withdraw_root"`
	BatchHash       common.Hash `json:"batch_hash"`
	ChainID         uint64      `json:"chain_id"`
}

// BatchProof includes the proof info that are required for batch verification and rollup.
type OpenVMBatchProof struct {
	MetaData struct {
		BatchInfo *OpenVMBatchInfo `json:"batch_info"`
		BatchHash common.Hash      `json:"batch_hash"`
	} `json:"metadata"`

	VmProof    *OpenVMProof `json:"proof"`
	Vk         []byte       `json:"vk,omitempty"`
	GitVersion string       `json:"git_version,omitempty"`
}

func (p *OpenVMBatchProof) Proof() []byte {
	proofJson, err := json.Marshal(p.VmProof)
	if err != nil {
		panic(fmt.Sprint("marshaling error", err))
	}

	return proofJson
}

// SanityCheck checks whether a BatchProof is in a legal format
func (ap *OpenVMBatchProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}
	if ap.MetaData.BatchInfo == nil {
		return errors.New("batch info not ready")
	}

	if ap.VmProof == nil {
		return errors.New("proof not ready")
	} else {
		if len(ap.Vk) == 0 {
			return errors.New("vk not ready")
		}
		pf := ap.VmProof
		if pf.Proof == nil {
			return errors.New("proof data not ready")
		}
		if len(pf.PublicValues) == 0 {
			return errors.New("proof public value not ready")
		}
	}

	return nil
}

// OpenVMBundleInfo is for calculating pi_hash for bundle header
type OpenVMBundleInfo struct {
	ChainID       uint64      `json:"chain_id"`
	PrevStateRoot common.Hash `json:"prev_state_root"`
	PostStateRoot common.Hash `json:"post_state_root"`
	WithdrawRoot  common.Hash `json:"withdraw_root"`
	NumBatches    uint32      `json:"num_batches"`
	PrevBatchHash common.Hash `json:"prev_batch_hash"`
	BatchHash     common.Hash `json:"batch_hash"`
}

// OpenVMBundleProof includes the proof info that are required for verification of a bundle of batch proofs.
type OpenVMBundleProof struct {
	MetaData struct {
		BundleInfo    *OpenVMBundleInfo `json:"bundle_info"`
		BunndlePIHash common.Hash       `json:"bundle_pi_hash"`
	} `json:"metadata"`

	EvmProof   *OpenVMEvmProof `json:"proof"`
	Vk         []byte          `json:"vk,omitempty"`
	GitVersion string          `json:"git_version,omitempty"`
}

// Proof returns the proof bytes that are eventually passed as calldata for on-chain bundle proof verification.
//
// There are 12 accumulators for a SNARK proof. The accumulators are the first 12 elements of the EvmProof's
// Instances field. The remaining items in Instances are supplied on-chain by the ScrollChain contract.
//
// The structure of these bytes is:
// | byte index start | byte length    | value    | description         |
// |------------------|----------------|----------|---------------------|
// | 0                | 32             | accs[0]  | accumulator 1       |
// | 32               | 32             | accs[1]  | accumulator 2       |
// | 32*i ...         | 32             | accs[i]  | accumulator i ...   |
// | 352              | 32             | accs[11] | accumulator 12      |
// | 384              | dynamic        | proof    | proof bytes         |
func (p *OpenVMBundleProof) Proof() []byte {
	append(p.EvmProof.Instances[0:384], p.EvmProof.Proof)
}

// SanityCheck checks whether a BundleProof is in a legal format
func (ap *OpenVMBundleProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}

	if ap.MetaData.BundleInfo == nil {
		return errors.New("bundle info not ready")
	}

	if ap.EvmProof == nil {
		return errors.New("proof not ready")
	} else {
		if len(ap.Vk) == 0 {
			return errors.New("vk not ready")
		}
		pf := ap.EvmProof
		if len(pf.Proof)%32 != 0 {
			return fmt.Errorf("proof buffer length must be a multiple of 32, got: %d", len(pf.Proof))
		}

		if len(pf.Instances) == 0 {
			return errors.New("instance not ready")
		}
	}

	return nil
}
