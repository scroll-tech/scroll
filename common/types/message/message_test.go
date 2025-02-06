package message

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
)

func TestDeserializeOpenVMProof(t *testing.T) {
	// Read the batch.json file located in the same directory.
	data, err := os.ReadFile("batch-proof-sample.json")
	if err != nil {
		t.Fatalf("failed to read batch proof sample.json: %v", err)
	}

	// Decode the JSON data into an BatchTask instance.
	batchProof := NewBatchProof("euclid")
	if err = json.Unmarshal(data, &batchProof); err != nil {
		t.Fatalf("failed to unmarshal JSON into Batch Proof: %v", err)
	}
	if err = batchProof.SanityCheck(); err != nil {
		t.Fatalf("failed to sanity check for Batch Proof: %v", err)
	}

	ovmbatchProof := batchProof.(*OpenVMBatchProof)

	if ovmbatchProof.MetaData.BatchInfo.ParentStateRoot !=
		common.HexToHash("0xe3440bcf882852bb1a9d6ba941e53a645220fee2c531ed79fa60481be8078c12") {
		t.Fatalf("get unexpected bundle info, parent state root is %v", ovmbatchProof.MetaData.BatchInfo.ParentStateRoot)
	}

	// Read the batch.json file located in the same directory.
	data, err = os.ReadFile("bundle-proof-sample.json")
	if err != nil {
		t.Fatalf("failed to read bundle proof sample.json: %v", err)
	}

	// Decode the JSON data into an BatchTask instance.
	bundleProof := NewBundleProof("euclid")
	if err = json.Unmarshal(data, &bundleProof); err != nil {
		t.Fatalf("failed to unmarshal JSON into Bundle Proof: %v", err)
	}
	if err = bundleProof.SanityCheck(); err != nil {
		t.Fatalf("failed to sanity check for Bundle Proof: %v", err)
	}
	ovmbundleProof := bundleProof.(*OpenVMBundleProof)

	if ovmbundleProof.MetaData.BundleInfo.PostStateRoot !=
		common.HexToHash("0x9e8b9928c55ccbc933911283175842fa515e49dd3f2fe0192c4346095695d741") {
		t.Fatalf("get unexpected bundle info, post state root is %v", ovmbundleProof.MetaData.BundleInfo.PostStateRoot)
	}
}
