//go:build mock_verifier

package verifier

import (
	"scroll-tech/common/message"

	"scroll-tech/coordinator/config"
)

// Verifier represents a mock halo2 verifier.
type Verifier struct {
}

// NewVerifier Sets up a mock verifier.
func NewVerifier(_ *config.VerifierConfig) (*Verifier, error) {
	return &Verifier{}, nil
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	return true, nil
}
