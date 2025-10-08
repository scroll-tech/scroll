//go:build mock_verifier

package libzkp

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

// GenerateWrappedProof returns a fixed dummy proof string in the mock.
func GenerateWrappedProof(proofJSON, metadata string, vkData []byte) string {
	return "mock-wrapped-proof"
}

// DumpVk is a no-op and returns nil in the mock.
func DumpVk(forkName, filePath string) error {
	return nil
}

// SetDynamicFeature is a no-op in the mock.
func SetDynamicFeature(feats string) {}
