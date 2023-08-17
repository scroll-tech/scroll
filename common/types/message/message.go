package message

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// RespStatus represents status code from prover to scroll
type RespStatus uint32

const (
	// StatusOk means generate proof success
	StatusOk RespStatus = iota
	// StatusProofError means generate proof failed
	StatusProofError
)

// ProofType represents the type of prover.
type ProofType uint8

func (r ProofType) String() string {
	switch r {
	case ProofTypeChunk:
		return "proof type chunk"
	case ProofTypeBatch:
		return "proof type batch"
	default:
		return fmt.Sprintf("illegal proof type: %d", r)
	}
}

const (
	// ProofTypeUndefined is an unknown proof type
	ProofTypeUndefined ProofType = iota
	// ProofTypeChunk is default prover, it only generates zk proof from traces.
	ProofTypeChunk
	// ProofTypeBatch generates zk proof from other zk proofs and aggregate them into one proof.
	ProofTypeBatch
)

// AuthMsg is the first message exchanged from the Prover to the Sequencer.
// It effectively acts as a registration, and makes the Prover identification
// known to the Sequencer.
type AuthMsg struct {
	// Message fields
	Identity *Identity `json:"message"`
	// Prover signature
	Signature string `json:"signature"`
}

// Identity contains all the fields to be signed by the prover.
type Identity struct {
	// ProverName the prover name
	ProverName string `json:"prover_name"`
	// ProverVersion the prover version
	ProverVersion string `json:"prover_version"`
	// Challenge unique challenge generated by manager
	Challenge string `json:"challenge"`
}

// GenerateToken generates token
func GenerateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SignWithKey auth message with private key and set public key in auth message's Identity
func (a *AuthMsg) SignWithKey(priv *ecdsa.PrivateKey) error {
	// Hash identity content
	hash, err := a.Identity.Hash()
	if err != nil {
		return err
	}

	// Sign register message
	sig, err := crypto.Sign(hash, priv)
	if err != nil {
		return err
	}
	a.Signature = hexutil.Encode(sig)

	return nil
}

// Verify verifies the message of auth.
func (a *AuthMsg) Verify() (bool, error) {
	hash, err := a.Identity.Hash()
	if err != nil {
		return false, err
	}
	sig := common.FromHex(a.Signature)

	pk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false, err
	}
	return crypto.VerifySignature(crypto.CompressPubkey(pk), hash, sig[:len(sig)-1]), nil
}

// PublicKey return public key from signature
func (a *AuthMsg) PublicKey() (string, error) {
	hash, err := a.Identity.Hash()
	if err != nil {
		return "", err
	}
	sig := common.FromHex(a.Signature)
	// recover public key
	pk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return "", err
	}
	return common.Bytes2Hex(crypto.CompressPubkey(pk)), nil
}

// Hash returns the hash of the auth message, which should be the message used
// to construct the Signature.
func (i *Identity) Hash() ([]byte, error) {
	byt, err := rlp.EncodeToBytes(i)
	if err != nil {
		return nil, err
	}
	hash := crypto.Keccak256Hash(byt)
	return hash[:], nil
}

// ProofMsg is the data structure sent to the coordinator.
type ProofMsg struct {
	*ProofDetail `json:"zkProof"`
	// Prover signature
	Signature string `json:"signature"`

	// Prover public key
	publicKey string
}

// Sign signs the ProofMsg.
func (a *ProofMsg) Sign(priv *ecdsa.PrivateKey) error {
	hash, err := a.ProofDetail.Hash()
	if err != nil {
		return err
	}
	sig, err := crypto.Sign(hash, priv)
	if err != nil {
		return err
	}
	a.Signature = hexutil.Encode(sig)
	return nil
}

// Verify verifies ProofMsg.Signature.
func (a *ProofMsg) Verify() (bool, error) {
	hash, err := a.ProofDetail.Hash()
	if err != nil {
		return false, err
	}
	sig := common.FromHex(a.Signature)
	// recover public key
	if a.publicKey == "" {
		pk, err := crypto.SigToPub(hash, sig)
		if err != nil {
			return false, err
		}
		a.publicKey = common.Bytes2Hex(crypto.CompressPubkey(pk))
	}

	return crypto.VerifySignature(common.FromHex(a.publicKey), hash, sig[:len(sig)-1]), nil
}

// PublicKey return public key from signature
func (a *ProofMsg) PublicKey() (string, error) {
	if a.publicKey == "" {
		hash, err := a.ProofDetail.Hash()
		if err != nil {
			return "", err
		}
		sig := common.FromHex(a.Signature)
		// recover public key
		pk, err := crypto.SigToPub(hash, sig)
		if err != nil {
			return "", err
		}
		a.publicKey = common.Bytes2Hex(crypto.CompressPubkey(pk))
		return a.publicKey, nil
	}

	return a.publicKey, nil
}

// TaskMsg is a wrapper type around db ProveTask type.
type TaskMsg struct {
	ID              string           `json:"id"`
	Type            ProofType        `json:"type,omitempty"`
	BatchTaskDetail *BatchTaskDetail `json:"batch_task_detail,omitempty"`
	ChunkTaskDetail *ChunkTaskDetail `json:"chunk_task_detail,omitempty"`
}

// ChunkTaskDetail is a type containing ChunkTask detail.
type ChunkTaskDetail struct {
	BlockHashes []common.Hash `json:"block_hashes"`
}

// BatchTaskDetail is a type containing BatchTask detail.
type BatchTaskDetail struct {
	ChunkInfos  []*ChunkInfo  `json:"chunk_infos"`
	ChunkProofs []*ChunkProof `json:"chunk_proofs"`
}

// ProofDetail is the message received from provers that contains zk proof, the status of
// the proof generation succeeded, and an error message if proof generation failed.
type ProofDetail struct {
	ID         string      `json:"id"`
	Type       ProofType   `json:"type,omitempty"`
	Status     RespStatus  `json:"status"`
	ChunkProof *ChunkProof `json:"chunk_proof,omitempty"`
	BatchProof *BatchProof `json:"batch_proof,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// Hash return proofMsg content hash.
func (z *ProofDetail) Hash() ([]byte, error) {
	byt, err := rlp.EncodeToBytes(z)
	if err != nil {
		return nil, err
	}

	hash := crypto.Keccak256Hash(byt)
	return hash[:], nil
}

// ChunkInfo is for calculating pi_hash for chunk
type ChunkInfo struct {
	ChainID       uint64      `json:"chain_id"`
	PrevStateRoot common.Hash `json:"prev_state_root"`
	PostStateRoot common.Hash `json:"post_state_root"`
	WithdrawRoot  common.Hash `json:"withdraw_root"`
	DataHash      common.Hash `json:"data_hash"`
	IsPadding     bool        `json:"is_padding"`
}

// ChunkProof includes the proof info that are required for chunk verification and rollup.
type ChunkProof struct {
	StorageTrace []byte `json:"storage_trace"`
	Protocol     []byte `json:"protocol"`
	Proof        []byte `json:"proof"`
	Instances    []byte `json:"instances"`
	Vk           []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	ChunkInfo  *ChunkInfo `json:"chunk_info,omitempty"`
	GitVersion string     `json:"git_version,omitempty"`
}

// BatchProof includes the proof info that are required for batch verification and rollup.
type BatchProof struct {
	Proof     []byte `json:"proof"`
	Instances []byte `json:"instances"`
	Vk        []byte `json:"vk"`
	// cross-reference between cooridinator computation and prover compution
	GitVersion string `json:"git_version,omitempty"`
}

// SanityCheck checks whether an BatchProof is in a legal format
// TODO: change to check Proof&Instance when upgrading to snark verifier v0.4
func (ap *BatchProof) SanityCheck() error {
	if ap == nil {
		return errors.New("agg_proof is nil")
	}

	if len(ap.Proof) == 0 {
		return errors.New("proof not ready")
	}
	if len(ap.Proof)%32 != 0 {
		return fmt.Errorf("proof buffer has wrong length, expected: 32, got: %d", len(ap.Proof))
	}

	return nil
}
