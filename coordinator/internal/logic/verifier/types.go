package verifier

import (
	"scroll-tech/coordinator/internal/config"
)

// InvalidTestProof invalid proof used in tests
const InvalidTestProof = "this is a invalid proof"

// Verifier represents a rust ffi to a verifier.
type Verifier struct {
	cfg         *config.VerifierConfig
	OpenVMVkMap map[string]struct{}
	ChunkVk     map[string][]byte
	BatchVk     map[string][]byte
	BundleVk    map[string][]byte
}
