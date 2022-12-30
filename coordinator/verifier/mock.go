//go:build mock_verifier

package verifier

import (
	"scroll-tech/common/message"
)

// Verifier represents a mock halo2 verifier.
type Verifier struct {
}

// NewVerifier Sets up a mock verifier.
func NewVerifier(_ *config.VerifierConfig) (*Verifier, error) {
	return &Verifier{}, nil
}

// VerifyProof always return true
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	return true, nil
}
