package utils

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

type BlobData struct {
	VersionedBlobHash string `json:"versionedBlobHash"`
	BlobData          string `json:"blobData"`
}

// TestCalculateVersionedBlobHash tests the CalculateVersionedBlobHash function
func TestCalculateVersionedBlobHash(t *testing.T) {
	// Read the test data
	data, err := os.ReadFile("../testdata/blobdata.json")
	if err != nil {
		t.Fatalf("Failed to read blobdata.json: %v", err)
	}

	var blobData BlobData
	if err := json.Unmarshal(data, &blobData); err != nil {
		t.Fatalf("Failed to parse blobdata.json: %v", err)
	}

	blobBytes, err := hex.DecodeString(blobData.BlobData)
	if err != nil {
		t.Fatalf("Failed to decode blob data: %v", err)
	}

	// Convert []byte to kzg4844.Blob
	var blob kzg4844.Blob
	copy(blob[:], blobBytes)

	// Calculate the hash
	calculatedHashBytes, err := CalculateVersionedBlobHash(blob)
	if err != nil {
		t.Fatalf("Failed to calculate versioned blob hash: %v", err)
	}

	calculatedHash := hex.EncodeToString(calculatedHashBytes[:])

	if calculatedHash != blobData.VersionedBlobHash {
		t.Fatalf("Hash mismatch: got %s, want %s", calculatedHash, blobData.VersionedBlobHash)
	}

}
