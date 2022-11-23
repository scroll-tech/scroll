//go:build mock_prover

//nolint:typecheck
package prover

import (
	"scroll-tech/common/message"

	"scroll-tech/roller/config"

	"github.com/scroll-tech/go-ethereum/core/types"
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
func (p *Prover) Prove(_ []*types.BlockTrace) (*message.AggProof, error) {
	return &message.AggProof{}, nil
}
