//go:build mock_prover

package prover

import (
	"github.com/scroll-tech/go-ethereum/common"
	"math/big"
	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/core/types"

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
func (p *Prover) Prove(taskID string, traces []*types.BlockTrace) (*message.AggProof, error) {
	_empty := common.BigToHash(big.NewInt(0))
	return &message.AggProof{
		Proof:     _empty[:],
		Instance:  _empty[:],
		FinalPair: _empty[:],
	}, nil
}
