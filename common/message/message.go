package message

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
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

// RollerType represents the type of roller.
type RollerType int8

func (r RollerType) String() string {
	switch r {
	case CommonRoller:
		return "Common Roller"
	case AggregatorRoller:
		return "Aggregator Roller"
	default:
		return "illegal Roller type"
	}
}

const (
	// CommonRoller is default roller, it only generates zk proof from traces.
	CommonRoller RollerType = iota
	// AggregatorRoller generates zk proof from other zk proofs and aggregate them into one proof.
	AggregatorRoller
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
	// Roller Type
	Type RollerType `json:"type"`
	// Unverified Unix timestamp of message creation
	Timestamp uint32 `json:"timestamp"`
	// Roller public key
	PublicKey string `json:"publicKey"`
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

// Sign auth message
func (a *AuthMsg) Sign(priv *ecdsa.PrivateKey) error {
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
	// recover public key
	if a.Identity.PublicKey == "" {
		pk, err := crypto.SigToPub(hash, sig)
		if err != nil {
			return false, err
		}
		a.Identity.PublicKey = common.Bytes2Hex(crypto.CompressPubkey(pk))
	}

	return crypto.VerifySignature(common.FromHex(a.Identity.PublicKey), hash, sig[:len(sig)-1]), nil
}

// PublicKey return public key from signature
func (a *AuthMsg) PublicKey() (string, error) {
	if a.Identity.PublicKey == "" {
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
		a.Identity.PublicKey = common.Bytes2Hex(crypto.CompressPubkey(pk))
		return a.Identity.PublicKey, nil
	}

	return a.Identity.PublicKey, nil
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
	ID string `json:"id"`
	// Only common rollers need traces, aggregator rollers don't!
	Traces []*types.BlockTrace `json:"blockTraces,omitempty"`
	// Only aggregator rollers need proofs, common rollers don't!
	Proofs []*AggProof `json:"proofs,omitempty"`
}

// ProofDetail is the message received from rollers that contains zk proof, the status of
// the proof generation succeeded, and an error message if proof generation failed.
type ProofDetail struct {
	ID     string     `json:"id"`
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
