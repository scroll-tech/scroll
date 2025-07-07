//go:build ffi

package verifier

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/config"
)

var (
	paramsPath      = flag.String("params", "/assets/test_params", "params dir")
	assetsPathLo    = flag.String("assets_lo", "/assets/test_assets_lo", "assets dir")
	assetsPathHi    = flag.String("assets", "/assets/test_assets", "assets dir")
	batchProofPath  = flag.String("batch_proof", "/assets/proof_data/batch_proof", "batch proof file path")
	chunkProofPath1 = flag.String("chunk_proof1", "/assets/proof_data/chunk_proof1", "chunk proof file path 1")
	chunkProofPath2 = flag.String("chunk_proof2", "/assets/proof_data/chunk_proof2", "chunk proof file path 2")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	cfg := &config.VerifierConfig{
		MinProverVersion: "",
		Verifiers: []config.AssetConfig{{
			AssetsPath: *assetsPathHi,
			ForkName:   "euclidV2",
		}},
	}

	v, err := NewVerifier(cfg)
	as.NoError(err)

	chunkProof1 := readChunkProof(*chunkProofPath1, as)
	chunkOk1, err := v.VerifyChunkProof(chunkProof1, "euclidV2")
	as.NoError(err)
	as.True(chunkOk1)
	t.Log("Verified chunk proof 1")

	chunkProof2 := readChunkProof(*chunkProofPath2, as)
	chunkOk2, err := v.VerifyChunkProof(chunkProof2, "euclidV2")
	as.NoError(err)
	as.True(chunkOk2)
	t.Log("Verified chunk proof 2")

	batchProof := readBatchProof(*batchProofPath, as)
	batchOk, err := v.VerifyBatchProof(batchProof, "euclidV2")
	as.NoError(err)
	as.True(batchOk)
	t.Log("Verified batch proof")
}

func readBatchProof(filePat string, as *assert.Assertions) *message.OpenVMBatchProof {
	f, err := os.Open(filePat)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)

	proof := &message.OpenVMBatchProof{}
	as.NoError(json.Unmarshal(byt, proof))

	return proof
}

func readChunkProof(filePat string, as *assert.Assertions) *message.OpenVMChunkProof {
	f, err := os.Open(filePat)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)

	proof := &message.OpenVMChunkProof{}
	as.NoError(json.Unmarshal(byt, proof))

	return proof
}
