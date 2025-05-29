package utils

import "crypto/sha256"

// CalculateVersionedBlobHash computes the versioned hash for blob data
// Following Ethereum's approach where:
// version = 0x01
// hash = sha256(blob)
// versionedHash = version + hash[1:]
func CalculateVersionedBlobHash(blobData []byte) [32]byte {
	// Step 1: Compute SHA-256 hash of the blob data
	hash := sha256.Sum256(blobData)
	
	// Step 2: Create versioned hash (version byte + hash[1:])
	var versionedHash [32]byte
	versionedHash[0] = 0x01 // Version byte
	copy(versionedHash[1:], hash[1:])
	
	return versionedHash
}