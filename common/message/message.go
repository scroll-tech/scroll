package message

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"golang.org/x/crypto/blake2s"
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

// AuthMessage is the first message exchanged from the Roller to the Sequencer.
// It effectively acts as a registration, and makes the Roller identification
// known to the Sequencer.
type AuthMessage struct {
	// Message fields
	Identity Identity `json:"message"`
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
}

// Sign auth message
func (a *AuthMessage) Sign(priv *ecdsa.PrivateKey) error {
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
	a.Signature = common.Bytes2Hex(sig)

	return nil
}

// Hash returns the hash of the auth message, which should be the message used
// to construct the Signature.
func (i *Identity) Hash() ([]byte, error) {
	bs, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	hash := blake2s.Sum256(bs)
	return hash[:], nil
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

// ProofMsg is the message received from rollers that contains zk proof, the status of
// the proof generation succeeded, and an error message if proof generation failed.
type ProofMsg struct {
	Status RespStatus `json:"status"`
	Error  string     `json:"error,omitempty"`
	ID     uint64     `json:"id"`
	// FIXME: Maybe we need to allow Proof omitempty? Also in scroll.
	Proof *AggProof `json:"proof"`
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
