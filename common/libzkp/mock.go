//go:build mock_zkp

package libzkp

import (
	"github.com/scroll-tech/go-ethereum/core/types"

	"scroll-tech/common/message"
)

// Prover sends block-traces to rust-prover through socket and get back the zk-proof.
type Prover struct {
	cfg *ProverConfig
}

// NewProver inits a Prover object.
func NewProver(cfg *ProverConfig) (*Prover, error) {
	return &Prover{cfg: cfg}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *Prover) Prove(_ []*types.BlockTrace) (*message.AggProof, error) {
	return &message.AggProof{
		Proof:     []byte{},
		Instance:  []byte{},
		FinalPair: []byte{},
	}, nil
}

// Verifier represents a mock halo2 verifier.
type Verifier struct {
}

// NewVerifier Sets up a mock verifier.
func NewVerifier(_ *VerifierConfig) (*Verifier, error) {
	return &Verifier{}, nil
}

// VerifyProof always return true
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	return true, nil
}
