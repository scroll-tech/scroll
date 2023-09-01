//go:build mock_verifier

package verifier

import (
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

// NewVerifier Sets up a mock verifier.
func NewVerifier(_ *config.VerifierConfig) (*Verifier, error) {
	return &Verifier{}, nil
}

// VerifyChunkProof return a mock verification result for a ChunkProof.
func (v *Verifier) VerifyChunkProof(proof *message.ChunkProof) (bool, error) {
	if string(proof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}

// VerifyBatchProof return a mock verification result for a BatchProof.
func (v *Verifier) VerifyBatchProof(proof *message.BatchProof) (bool, error) {
	if string(proof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}
