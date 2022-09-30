package message

import (
	"crypto/ecdsa"
	"encoding/json"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

// BlockTraces is a wrapper type around types.BlockResult which adds an ID
// that identifies which proof generation session these block traces are
// associated to. This then allows the roller to add the ID back to their
// proof message once generated, and in turn helps the sequencer understand
// where to handle the proof.
type BlockTraces struct {
	ID     uint64             `json:"id"`
	Traces *types.BlockResult `json:"blockTraces"`
}

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

type RespStatus uint32

const (
	StatusOk RespStatus = iota
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
	*Identity `json:"message"`
	// Roller signature
	Signature string `json:"signature"`
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

func (a *AuthMessage) Verify() (bool, error) {
	hash, err := a.Hash()
	if err != nil {
		return false, err
	}
	sig := common.FromHex(a.Signature)
	return crypto.VerifySignature(common.FromHex(a.PublicKey), hash, sig[:len(sig)-1]), nil
}

// Identity contains all the fields to be signed by the roller.
type Identity struct {
	// Roller name
	Name      string `json:"name"`
	Timestamp int64
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

type AuthZkProof struct {
	*ProofMsg `json:"zkProof"`
	// Roller public key
	PublicKey string `json:"publicKey"`
	// Roller signature
	Signature string `json:"signature"`
}

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

func (a *AuthZkProof) Verify() (bool, error) {
	hash, err := a.Hash()
	if err != nil {
		return false, err
	}
	sig := common.FromHex(a.Signature)
	return crypto.VerifySignature(common.FromHex(a.PublicKey), hash, sig[:len(sig)-1]), nil
}

// ProofMsg is the message received from rollers that contains zk proof, the status of
// the proof generation succeeded, and an error message if proof generation failed.
type ProofMsg struct {
	ID     uint64     `json:"id"`
	Status RespStatus `json:"status"`
	Proof  *AggProof  `json:"proof"`
	Error  string     `json:"error,omitempty"`
}

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
