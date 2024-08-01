package verifier

import (
	"scroll-tech/coordinator/internal/config"
)

// InvalidTestProof invalid proof used in tests
const InvalidTestProof = "this is a invalid proof"

// Verifier represents a rust ffi to a halo2 verifier.
type Verifier struct {
	cfg         *config.VerifierConfig
	ChunkVKMap  map[string]string
	BatchVKMap  map[string]string
	BundleVkMap map[string]string
}
