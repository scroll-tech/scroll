package types

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// ProofMsg is the data structure sent to the coordinator.
type ProofMsg struct {
	*ProofDetail `json:"zkProof"`
	Signature    string `json:"signature"`
	publicKey    string // assign publicKey from authMsg
}

// Sign signs the ProofMsg.
func (p *ProofMsg) Sign(priv *ecdsa.PrivateKey) error {
	hash, err := p.ProofDetail.Hash()
	if err != nil {
		return err
	}
	sig, err := crypto.Sign(hash, priv)
	if err != nil {
		return err
	}
	p.Signature = hexutil.Encode(sig)
	return nil
}

// Verify verifies ProofMsg.Signature.
func (p *ProofMsg) Verify() (bool, error) {
	if p.publicKey == "" {
		return false, errors.New("public key is empty")
	}

	hash, err := p.ProofDetail.Hash()
	if err != nil {
		return false, err
	}

	sig := common.FromHex(p.Signature)
	pk, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return false, err
	}

	isValid := crypto.VerifySignature(crypto.CompressPubkey(pk), hash, sig[:len(sig)-1])
	if !isValid {
		return false, nil
	}

	expectedPubKey, err := p.ProofDetail.DecodeAndUnmarshalPubkey(p.publicKey)
	if err != nil {
		return false, err
	}
	expectedAddr := crypto.PubkeyToAddress(*expectedPubKey)
	recoveredAddr := crypto.PubkeyToAddress(*pk)
	return recoveredAddr == expectedAddr, nil
}

// ProofDetail is the message received from provers that contains zk proof, the status of
// the proof generation succeeded, and an error message if proof generation failed.
type ProofDetail struct {
	UUID        string `form:"uuid" json:"uuid"`
	TaskID      string `form:"task_id" json:"task_id" binding:"required"`
	TaskType    int    `form:"task_type" json:"task_type" binding:"required"`
	Status      int    `form:"status" json:"status"`
	Proof       string `form:"proof" json:"proof"`
	FailureType int    `form:"failure_type" json:"failure_type"`
	FailureMsg  string `form:"failure_msg" json:"failure_msg"`
}

// Hash return proofMsg content hash.
func (p *ProofDetail) Hash() ([]byte, error) {
	byt, err := rlp.EncodeToBytes(p)
	if err != nil {
		return nil, err
	}

	hash := crypto.Keccak256Hash(byt)
	return hash[:], nil
}

// DecodeAndUnmarshalPubkey decodes a hex-encoded public key and unmarshal it into an ecdsa.PublicKey
func (p *ProofDetail) DecodeAndUnmarshalPubkey(pubKeyHex string) (*ecdsa.PublicKey, error) {
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
