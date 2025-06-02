package message

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

const (
	EuclidV2Fork = "euclidV2"

	EuclidV2ForkNameForProver = "euclidv2"
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

// ChunkTaskDetail is a type containing ChunkTask detail for chunk task.
type ChunkTaskDetail struct {
	// use one of the string of "euclidv1" / "euclidv2"
	ForkName         string        `json:"fork_name"`
	BlockHashes      []common.Hash `json:"block_hashes"`
	PrevMsgQueueHash common.Hash   `json:"prev_msg_queue_hash"`
}

// it is a hex encoded big with fixed length on 48 bytes
type Byte48 struct {
	hexutil.Big
}

func (e Byte48) MarshalText() ([]byte, error) {
	i := e.ToInt()
	// overrite encode big
	if sign := i.Sign(); sign < 0 {
		// sanity check
		return nil, errors.New("Byte48 must be positive integer")
	} else {
		s := i.Text(16)
		if len(s) > 96 {
			return nil, errors.New("integer Exceed 384bit")
		}
		return []byte(fmt.Sprintf("0x%0*s", 96, s)), nil
	}
}

func isString(input []byte) bool {
	return len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"'
}

// hexutil.Big has limition of 256bit so we have to override it ...
func (e *Byte48) UnmarshalJSON(input []byte) error {
	if !isString(input) {
		return errors.New("not hex string")
	}

	b, err := hexutil.Decode(string(input[1 : len(input)-1]))
	if err != nil {
		return err
	}
	if len(b) != 48 {
		return fmt.Errorf("not a 48 bytes hex string: %d", len(b))
	}
	var dec big.Int
	dec.SetBytes(b)
	*e = Byte48{(hexutil.Big)(dec)}
	return nil
}

// BatchTaskDetail is a type containing BatchTask detail.
type BatchTaskDetail struct {
	// use one of the string of "euclidv1" / "euclidv2"
	ForkName        string              `json:"fork_name"`
	ChunkInfos      []*ChunkInfo        `json:"chunk_infos"`
	ChunkProofs     []*OpenVMChunkProof `json:"chunk_proofs"`
	BatchHeader     interface{}         `json:"batch_header"`
	BlobBytes       []byte              `json:"blob_bytes"`
	KzgProof        Byte48              `json:"kzg_proof,omitempty"`
	KzgCommitment   Byte48              `json:"kzg_commitment,omitempty"`
	ChallengeDigest common.Hash         `json:"challenge_digest,omitempty"`
}

// BundleTaskDetail consists of all the information required to describe the task to generate a proof for a bundle of batches.
type BundleTaskDetail struct {
	// use one of the string of "euclidv1" / "euclidv2"
	ForkName    string              `json:"fork_name"`
	BatchProofs []*OpenVMBatchProof `json:"batch_proofs"`
	BundleInfo  *OpenVMBundleInfo   `json:"bundle_info,omitempty"`
}

// ChunkInfo is for calculating pi_hash for chunk
type ChunkInfo struct {
	ChainID            uint64           `json:"chain_id"`
	PrevStateRoot      common.Hash      `json:"prev_state_root"`
	PostStateRoot      common.Hash      `json:"post_state_root"`
	WithdrawRoot       common.Hash      `json:"withdraw_root"`
	DataHash           common.Hash      `json:"data_hash"`
	IsPadding          bool             `json:"is_padding"`
	TxBytes            []byte           `json:"tx_bytes"`
	TxBytesHash        common.Hash      `json:"tx_data_digest"`
	PrevMsgQueueHash   common.Hash      `json:"prev_msg_queue_hash"`
	PostMsgQueueHash   common.Hash      `json:"post_msg_queue_hash"`
	TxDataLength       uint64           `json:"tx_data_length"`
	InitialBlockNumber uint64           `json:"initial_block_number"`
	BlockCtxs          []BlockContextV2 `json:"block_ctxs"`
}

// BlockContextV2 is the block context for euclid v2
type BlockContextV2 struct {
	Timestamp uint64      `json:"timestamp"`
	BaseFee   hexutil.Big `json:"base_fee"`
	GasLimit  uint64      `json:"gas_limit"`
	NumTxs    uint16      `json:"num_txs"`
	NumL1Msgs uint16      `json:"num_l1_msgs"`
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
	ParentBatchHash  common.Hash `json:"parent_batch_hash"`
	ParentStateRoot  common.Hash `json:"parent_state_root"`
	StateRoot        common.Hash `json:"state_root"`
	WithdrawRoot     common.Hash `json:"withdraw_root"`
	BatchHash        common.Hash `json:"batch_hash"`
	ChainID          uint64      `json:"chain_id"`
	PrevMsgQueueHash common.Hash `json:"prev_msg_queue_hash"`
	PostMsgQueueHash common.Hash `json:"post_msg_queue_hash"`
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
	MsgQueueHash  common.Hash `json:"msg_queue_hash"`
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
	proofBytes := make([]byte, 0, 384+len(p.EvmProof.Proof))
	proofBytes = append(proofBytes, p.EvmProof.Instances[:384]...)
	return append(proofBytes, p.EvmProof.Proof...)
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
