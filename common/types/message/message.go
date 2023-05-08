package message

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"github.com/scroll-tech/go-ethereum/core/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// RespStatus represents status code from roller to scroll
type RespStatus uint32

const (
	// StatusOk means generate proof success
	StatusOk RespStatus = iota
	// StatusProofError means generate proof failed
	StatusProofError
)

// ProveType represents the type of roller.
type ProveType uint8

func (r ProveType) String() string {
	switch r {
	case BasicProve:
		return "Basic Prove"
	case AggregatorProve:
		return "Aggregator Prove"
	default:
		return "Illegal Prove type"
	}
}

const (
	// BasicProve is default roller, it only generates zk proof from traces.
	BasicProve ProveType = iota
	// AggregatorProve generates zk proof from other zk proofs and aggregate them into one proof.
	AggregatorProve
)

// AuthMsg is the first message exchanged from the Roller to the Sequencer.
// It effectively acts as a registration, and makes the Roller identification
// known to the Sequencer.
type AuthMsg struct {
	// Message fields
	Identity *Identity `json:"message"`
	// Roller signature
	Signature string `json:"signature"`
}

// Identity contains all the fields to be signed by the roller.
type Identity struct {
	// Roller name
	Name string `json:"name"`
	// Roller RollerType
	RollerType ProveType `json:"roller_type,omitempty"`
	// Unverified Unix timestamp of message creation
	Timestamp uint32 `json:"timestamp"`
	// Version is common.Version+ZkVersion. Use the following to check the latest ZkVersion version.
	// curl -sL https://api.github.com/repos/scroll-tech/scroll-zkevm/commits | jq -r ".[0].sha"
	Version string `json:"version"`
	// Random unique token generated by manager
	Token string `json:"token"`
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
	// Roller signature
	Signature string `json:"signature"`

	// Roller public key
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
	ID   string    `json:"id"`
	Type ProveType `json:"type,omitempty"`
	// Only basic rollers need blockHashes, aggregator rollers don't!
	BlockHashes []common.Hash `json:"block_hashes,omitempty"`
	// In BasicProve, we encourage coordinator pass blockHashes to rollers, not traces!
	Traces []*types.BlockTrace `json:"blockTraces,omitempty"`
	// Only aggregator rollers need proofs to aggregate, basic rollers don't!
	SubProofs [][]byte `json:"sub_proofs,omitempty"`
}

// ProofDetail is the message received from rollers that contains zk proof, the status of
// the proof generation succeeded, and an error message if proof generation failed.
type ProofDetail struct {
	ID     string     `json:"id"`
	Type   ProveType  `json:"type,omitempty"`
	Status RespStatus `json:"status"`
	Proof  *AggProof  `json:"proof"`
	Error  string     `json:"error,omitempty"`
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

// AggProof includes the proof and public input that are required to verification and rollup.
type AggProof struct {
	Proof      []byte `json:"proof"`
	Instance   []byte `json:"instance"`
	FinalPair  []byte `json:"final_pair"`
	Vk         []byte `json:"vk"`
	BlockCount uint   `json:"block_count"`
}
