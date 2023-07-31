//go:build mock_prover

package core

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"

	"scroll-tech/common/types/message"
)

// ProverCore sends block-traces to rust-prover through socket and get back the zk-proof.
type ProverCore struct {
}

// NewProverCore inits a ProverCore object.
func NewProverCore(cfg interface{}) (*ProverCore, error) {
	return &ProverCore{}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *ProverCore) Prove(taskID string, traces []*types.BlockTrace) (*message.ChunkProof, error) {
	_empty := common.BigToHash(big.NewInt(0))
	return &message.ChunkProof{
		Proof:     _empty[:],
		Instance:  _empty[:],
		FinalPair: _empty[:],
	}, nil
}
