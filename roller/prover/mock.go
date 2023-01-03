//go:build mock_prover

package prover

import (
	"github.com/scroll-tech/go-ethereum/core/types"

	"scroll-tech/common/message"
	"scroll-tech/common/viper"
)

// Prover sends block-traces to rust-prover through socket and get back the zk-proof.
type Prover struct {
	vp *viper.Viper
}

// NewProver inits a Prover object.
func NewProver(vp *viper.Viper) (*Prover, error) {
	return &Prover{vp: vp}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *Prover) Prove(_ []*types.BlockTrace) (*message.AggProof, error) {
	return &message.AggProof{
		Proof:     []byte{},
		Instance:  []byte{},
		FinalPair: []byte{},
	}, nil
}
