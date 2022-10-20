package message

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// MsgType denotes the type of message being sent or received.
type MsgType uint16

const (
	// Error message.
	Error MsgType = iota
	// Register message, sent by a roller when a connection is established.
	Register
	// BlockTrace message, sent by a sequencer to a roller to notify them
	// they need to generate a proof.
	BlockTrace
	// Proof message, sent by a roller to a sequencer when they have finished
	// proof generation of a given set of block traces.
	Proof
)

// RespStatus is the status of the proof generation
type RespStatus uint32

const (
	// StatusOk indicates the proof generation succeeded
	StatusOk RespStatus = iota
	// StatusProofError indicates the proof generation failed
	StatusProofError
)

// Msg is the top-level message container which contains the payload and the
// message identifier.
type Msg struct {
	// Message type
	Type MsgType `json:"type"`
	// Message payload
	Payload []byte `json:"payload"`
}

// BlockTraces is a wrapper type around types.BlockResult which adds an ID
// that identifies which proof generation session these block traces are
// associated to. This then allows the roller to add the ID back to their
// proof message once generated, and in turn helps the sequencer understand
// where to handle the proof.
type BlockTraces struct {
	ID     uint64             `json:"id"`
	Traces *types.BlockResult `json:"blockTraces"`
}

// AuthMessage is the first message exchanged from the Roller to the Sequencer.
// It effectively acts as a registration, and makes the Roller identification
// known to the Sequencer.
type AuthMessage struct {
	// Message fields
	*Identity `json:"message"`
	// Roller signature
	Signature string `json:"signature"`

	// public key
	publicKey string
}

// Sign auth message
func (a *AuthMessage) Sign(priv *ecdsa.PrivateKey) error {
	// Hash identity content
	hash, err := a.Hash()
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

// Verify auth message
func (a *AuthMessage) Verify() (bool, error) {
	hash, err := a.Hash()
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
func (a *AuthMessage) PublicKey() (string, error) {
	if a.publicKey == "" {
		hash, err := a.Hash()
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

// Identity contains all the fields to be signed by the roller.
type Identity struct {
	// Roller name
	Name string `json:"name"`
	// Time of message creation
	Timestamp int64 `json:"timestamp"`
	// Roller public key
	PublicKey string `json:"publicKey"`
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

// nolint
type AuthZkProof struct {
	*ProofMsg `json:"zkProof"`
	// Roller signature
	Signature string `json:"signature"`

	// Roller public key
	publicKey string
}

// Sign AuthZkProof
func (a *AuthZkProof) Sign(priv *ecdsa.PrivateKey) error {
	hash, err := a.Hash()
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

// Verify AuthZkProof
func (a *AuthZkProof) Verify() (bool, error) {
	hash, err := a.Hash()
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
func (a *AuthZkProof) PublicKey() (string, error) {
	if a.publicKey == "" {
		hash, err := a.Hash()
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

// ProofMsg is the message received from rollers that contains zk proof, the status of
// the proof generation succeeded, and an error message if proof generation failed.
type ProofMsg struct {
	ID     uint64     `json:"id"`
	Status RespStatus `json:"status"`
	Proof  *AggProof  `json:"proof"`
	Error  string     `json:"error,omitempty"`
}

// Hash return proofMsg content hash.
func (z *ProofMsg) Hash() ([]byte, error) {
	bs, err := json.Marshal(z)
	if err != nil {
		return nil, err
	}

	hash := crypto.Keccak256Hash(bs)
	return hash[:], nil
}

// AggProof includes the proof and public input that are required to verification and rollup.
type AggProof struct {
	Proof     string `json:"proof"`
	Instance  string `json:"instance"`
	FinalPair string `json:"final_pair"`
	Vk        string `json:"vk"`
}
