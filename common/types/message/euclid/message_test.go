package euclid

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDeserializeProof(t *testing.T) {
	// Read the batch.json file located in the same directory.
	data, err := os.ReadFile("batch-proof-sample.json")
	if err != nil {
		t.Fatalf("failed to read batch proof sample.json: %v", err)
	}

	// Decode the JSON data into an BatchTask instance.
	var batchProof BatchProof
	if err := json.Unmarshal(data, &batchProof); err != nil {
		t.Fatalf("failed to unmarshal JSON into Batch Proof: %v", err)
	}
	if err := batchProof.SanityCheck(); err != nil {
		t.Fatalf("failed to sanity check for Batch Proof: %v", err)
	}

	// Read the batch.json file located in the same directory.
	data, err = os.ReadFile("bundle-proof-sample.json")
	if err != nil {
		t.Fatalf("failed to read bundle proof sample.json: %v", err)
	}

	// Decode the JSON data into an BatchTask instance.
	var bundleProof BundleProof
	if err := json.Unmarshal(data, &bundleProof); err != nil {
		t.Fatalf("failed to unmarshal JSON into Bundle Proof: %v", err)
	}
	if err := bundleProof.SanityCheck(); err != nil {
		t.Fatalf("failed to sanity check for Bundle Proof: %v", err)
	}
}
