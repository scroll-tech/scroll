//go:build mock_verifier

package verifier

import (
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

const InvalidTestProof = "this is a invalid proof"

// Verifier represents a mock halo2 verifier.
type Verifier struct{}

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

// OldVerifier represents a mock halo2 verifier.
type OldVerifier struct{}

// NewVerifier Sets up a mock verifier.
func NewOldVerifier(_ *config.VerifierConfig) (*OldVerifier, error) {
	return &OldVerifier{}, nil
}

// VerifyChunkProof return a mock verification result for a ChunkProof.
func (v *OldVerifier) VerifyChunkProof(proof *message.ChunkProof) (bool, error) {
	if string(proof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}

// VerifyBatchProof return a mock verification result for a BatchProof.
func (v *OldVerifier) VerifyBatchProof(proof *message.BatchProof) (bool, error) {
	if string(proof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}
