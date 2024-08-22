//go:build mock_verifier

package verifier

import (
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

// NewVerifier Sets up a mock verifier.
func NewVerifier(cfg *config.VerifierConfig) (*Verifier, error) {
	batchVKMap := map[string]struct{}{"mock_vk": {}}
	chunkVKMap := map[string]struct{}{"mock_vk": {}}
	return &Verifier{cfg: cfg, ChunkVKMap: chunkVKMap, BatchVKMap: batchVKMap}, nil
}

// VerifyChunkProof return a mock verification result for a ChunkProof.
func (v *Verifier) VerifyChunkProof(proof *message.ChunkProof, forkName string) (bool, error) {
	if string(proof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}

// VerifyBatchProof return a mock verification result for a BatchProof.
func (v *Verifier) VerifyBatchProof(proof *message.BatchProof, forkName string) (bool, error) {
	if string(proof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}

// VerifyBundleProof return a mock verification result for a BundleProof.
func (v *Verifier) VerifyBundleProof(proof *message.BundleProof, forkName string) (bool, error) {
	if string(proof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}
