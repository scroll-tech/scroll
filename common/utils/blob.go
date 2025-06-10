package utils

import (
	"crypto/sha256"
	"fmt"

	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

// CalculateVersionedBlobHash calculate the kzg4844 versioned blob hash from a blob
func CalculateVersionedBlobHash(blob kzg4844.Blob) ([32]byte, error) {
	// calculate kzg4844 commitment from blob
	commit, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get blob commitment, err: %w", err)
	}

	// calculate kzg4844 versioned blob hash from blob commitment
	hasher := sha256.New()
	vh := kzg4844.CalcBlobHashV1(hasher, &commit)

	return vh, nil
}
