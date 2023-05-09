//go:build mock_prover

package prover

import (
	"scroll-tech/common/types/message"

	"scroll-tech/roller/config"
)

// Prover sends block-traces to rust-prover through socket and get back the zk-proof.
type Prover struct {
	cfg *config.ProverConfig
}

// NewProver inits a Prover object.
func NewProver(cfg *config.ProverConfig) (*Prover, error) {
	return &Prover{cfg: cfg}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *Prover) Prove(_ *message.TaskMsg) (*message.AggProof, error) {
	return &message.AggProof{
		Proof:     []byte{},
		Instance:  []byte{},
		FinalPair: []byte{},
	}, nil
}
