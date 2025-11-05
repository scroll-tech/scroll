//go:build mock_verifier

package libzkp

import (
	"encoding/json"
)

// // InitVerifier is a no-op in the mock.
// func InitVerifier(configJSON string) {}

// // VerifyChunkProof returns a fixed success in the mock.
// func VerifyChunkProof(proofData, forkName string) bool {
// 	return true
// }

// // VerifyBatchProof returns a fixed success in the mock.
// func VerifyBatchProof(proofData, forkName string) bool {
// 	return true
// }

// // VerifyBundleProof returns a fixed success in the mock.
// func VerifyBundleProof(proofData, forkName string) bool {
// 	return true
// }

func UniversalTaskCompatibilityFix(taskJSON string) (string, error) {
	panic("should not run here")
}

// GenerateWrappedProof returns a fixed dummy proof string in the mock.
func GenerateWrappedProof(proofJSON, metadata string, vkData []byte) string {

	payload := struct {
		Metadata   json.RawMessage `json:"metadata"`
		Proof      json.RawMessage `json:"proof"`
		GitVersion string          `json:"git_version"`
	}{
		Metadata:   json.RawMessage(metadata),
		Proof:      json.RawMessage(proofJSON),
		GitVersion: "mock-git-version",
	}

	out, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// DumpVk is a no-op and returns nil in the mock.
func DumpVk(forkName, filePath string) error {
	return nil
}

// SetDynamicFeature is a no-op in the mock.
func SetDynamicFeature(feats string) {}
