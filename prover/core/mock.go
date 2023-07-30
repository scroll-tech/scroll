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
	cfg *config.ProverCoreConfig
}

// NewProverCore inits a ProverCore object.
func NewProverCore(cfg *config.ProverCoreConfig) (*ProverCore, error) {
	return &ProverCore{cfg: cfg}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *ProverCore) Prove(taskID string, traces []*types.BlockTrace) (*message.BatchProof, error) {
	_empty := common.BigToHash(big.NewInt(0))
	return &message.BatchProof{
		Proof:     _empty[:],
		Instance:  _empty[:],
		FinalPair: _empty[:],
	}, nil
}
