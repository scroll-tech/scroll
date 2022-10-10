//go:build mock_prover

//nolint:typecheck
package prover

import (
	"time"

	"scroll-tech/go-roller/config"
	"scroll-tech/go-roller/message"

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
func (p *Prover) Prove(traces *types.BlockResult) (*message.AggProof, error) {
	proof, err := p.prove(traces)
	if err != nil {
		return p.prove(traces)
	}
	return proof, nil
}

func (p *Prover) prove(traces *types.BlockResult) (*message.AggProof, error) {
	time.Sleep(5 * time.Second)
	return &message.AggProof{}, nil
}
