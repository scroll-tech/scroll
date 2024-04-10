//go:build mock_prover

package core

import (
    "math/big"

    "github.com/scroll-tech/go-ethereum/common"
    "scroll-tech/common/types/message"
)

// ProverCore sends block-traces to rust-prover through socket and get back the zk-proof.
type ProverCore struct{}

// NewProverCore inits a ProverCore object.
func NewProverCore() *ProverCore {
    return &ProverCore{}
}

func (p *ProverCore) ProveChunk(taskID string, traces []interface{}) (*message.ChunkProof, error) {
    empty := common.BigToHash(big.NewInt(0))

    return &message.ChunkProof{
        StorageTrace: empty[:],
        Protocol:     empty[:],
        Proof:        empty[:],
        Instances:    empty[:],
        Vk:           empty[:],
    }, nil
}

func (p *ProverCore) ProveBatch(taskID string, chunkInfos []message.ChunkInfo, chunkProofs []*message.ChunkProof) (*message.BatchProof, error) {
    empty := common.BigToHash(big.NewInt(0))

    return &message.BatchProof{
        Proof:     empty[:],
        Instances: empty[:],
        Vk:        empty[:],
    }, nil
}


