//go:build mock_verifier

package verifier

import (
	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

// NewVerifier Sets up a mock verifier.
func NewVerifier(cfg *config.VerifierConfig) (*Verifier, error) {
	return &Verifier{
		cfg:         cfg,
		OpenVMVkMap: map[string]struct{}{"mock_vk": {}},
		ChunkVk:     map[string][]byte{"euclidV2": []byte("mock_vk")},
		BatchVk:     map[string][]byte{"euclidV2": []byte("mock_vk")},
		BundleVk:    map[string][]byte{},
	}, nil
}

// VerifyChunkProof return a mock verification result for a ChunkProof.
func (v *Verifier) VerifyChunkProof(proof *message.OpenVMChunkProof, forkName string) (bool, error) {
	if proof.VmProof != nil && string(proof.VmProof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}

// VerifyBatchProof return a mock verification result for a BatchProof.
func (v *Verifier) VerifyBatchProof(proof *message.OpenVMBatchProof, forkName string) (bool, error) {
	if proof.VmProof != nil && string(proof.VmProof.Proof) == InvalidTestProof {
		return false, nil
	}
	return true, nil
}

// VerifyBundleProof return a mock verification result for a BundleProof.
func (v *Verifier) VerifyBundleProof(proof *message.OpenVMBundleProof, forkName string) (bool, error) {
	return true, nil
}
