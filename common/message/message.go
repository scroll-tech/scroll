package message

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// RespStatus represents status code from roller to scroll
type RespStatus uint32

const (
	// StatusOk means generate proof success
	StatusOk RespStatus = iota
	// StatusProofError means generate proof failed
	StatusProofError
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
	// Time of message creation
	Timestamp int64 `json:"timestamp"`
	// Roller public key
	PublicKey string `json:"publicKey"`
	// Version is common.Version+ZK_VERSION. Use the following to check the latest ZK_VERSION version.
	// curl -sL https://api.github.com/repos/scroll-tech/common-rs/commits | jq -r ".[0].sha"
	Version string `json:"version"`
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
	bs, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	hash := crypto.Keccak256Hash(bs)
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
	ID     string              `json:"id"`
	Traces []*types.BlockTrace `json:"blockTraces"`
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
	bs, err := json.Marshal(z)
	if err != nil {
		return nil, err
	}

	hash := crypto.Keccak256Hash(bs)
	return hash[:], nil
}

// AggProof includes the proof and public input that are required to verification and rollup.
type AggProof struct {
	Proof     []byte `json:"proof"`
	Instance  []byte `json:"instance"`
	FinalPair []byte `json:"final_pair"`
	Vk        []byte `json:"vk"`
}

// Marshal marshals the TraceProof as bytes
func (proof *AggProof) Marshal() ([]byte, error) {
	jsonByt, err := json.Marshal(proof)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(len(jsonByt)))
	return append(buf, jsonByt...), nil
}
