package types

import (
	"crypto/ecdsa"
	"encoding/hex"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rlp"
)

const (
	// PublicKey the public key for context
	PublicKey = "public_key"
	// ProverName the prover name key for context
	ProverName = "prover_name"
	// ProverVersion the prover version for context
	ProverVersion = "prover_version"
	// HardForkName the hard fork name for context
	HardForkName = "hard_fork_name"
)

// LoginSchema for /login response
type LoginSchema struct {
	Time  time.Time `json:"time"`
	Token string    `json:"token"`
}

// Message the login message struct
type Message struct {
	Challenge     string       `form:"challenge" json:"challenge" binding:"required"`
	ProverVersion string       `form:"prover_version" json:"prover_version" binding:"required"`
	ProverName    string       `form:"prover_name" json:"prover_name" binding:"required"`
	ProverTypes   []ProverType `form:"prover_types" json:"prover_types"`
	VKs           []string     `form:"vks" json:"vks"`
}

// LoginParameterWithHardForkName constructs new payload for login
type LoginParameterWithHardForkName struct {
	LoginParameter
	HardForkName string `form:"hard_fork_name" json:"hard_fork_name"`
}

// LoginParameter for /login api
type LoginParameter struct {
	Message   Message `form:"message" json:"message" binding:"required"`
	PublicKey string  `form:"public_key" json:"public_key"`
	Signature string  `form:"signature" json:"signature" binding:"required"`
}

// SignWithKey auth message with private key and set public key in auth message's Identity
func (a *LoginParameter) SignWithKey(priv *ecdsa.PrivateKey) error {
	// Hash identity content
	hash, err := a.Message.Hash()
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
func (a *LoginParameter) Verify() (bool, error) {
	hash, err := a.Message.Hash()
	if err != nil {
		return false, err
	}

	expectedPubKey, err := a.Message.DecodeAndUnmarshalPubkey(a.PublicKey)
	if err != nil {
		return false, err
	}

	sig := common.FromHex(a.Signature)
	isValid := crypto.VerifySignature(crypto.CompressPubkey(expectedPubKey), hash, sig[:len(sig)-1])
	return isValid, nil
}

// Hash returns the hash of the auth message, which should be the message used
// to construct the Signature.
func (i *Message) Hash() ([]byte, error) {
	byt, err := rlp.EncodeToBytes(i)
	if err != nil {
		return nil, err
	}
	hash := crypto.Keccak256Hash(byt)
	return hash[:], nil
}

// DecodeAndUnmarshalPubkey decodes a hex-encoded public key and unmarshal it into an ecdsa.PublicKey
func (i *Message) DecodeAndUnmarshalPubkey(pubKeyHex string) (*ecdsa.PublicKey, error) {
	// Decode hex string to bytes
	byteKey, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, err
	}

	// Unmarshal bytes to ECDSA public key
	pubKey, err := crypto.DecompressPubkey(byteKey)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}
