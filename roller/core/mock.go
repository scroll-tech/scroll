//go:build mock_prover

package core

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"

	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
)

// ProverCore sends block-traces to rust-prover through socket and get back the zk-proof.
type ProverCore struct {
	cfg *config.ProverConfig
}

// NewProver inits a ProverCore object.
func NewProver(cfg *config.ProverConfig) (*ProverCore, error) {
	return &ProverCore{cfg: cfg}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *ProverCore) Prove(taskID string, traces []*types.BlockTrace) (*message.AggProof, error) {
	_empty := common.BigToHash(big.NewInt(0))
	return &message.AggProof{
		Proof:     _empty[:],
		Instance:  _empty[:],
		FinalPair: _empty[:],
	}, nil
}
