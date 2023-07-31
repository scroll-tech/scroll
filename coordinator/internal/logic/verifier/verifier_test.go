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
	"scroll-tech/coordinator/internal/logic/verifier"
)

var (
	paramsPath     = flag.String("params", "/assets/test_params", "params dir")
	assetsPath     = flag.String("assets", "/assets/test_assets", "assets dir")
	batchProofPath = flag.String("batch_proof", "/assets/proof_data/batch_proof", "batch proof file path")
	chunkProofPath = flag.String("chunk_proof", "/assets/proof_data/chunk_proof", "chunk proof file path")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)
	cfg := &config.VerifierConfig{
		MockMode:   false,
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
	}
	v, err := verifier.NewVerifier(cfg)
	as.NoError(err)

	chunkProofFile, err := os.Open(*chunkProofPath)
	as.NoError(err)
	chunkProofByt, err := io.ReadAll(chunkProofFile)
	as.NoError(err)
	chunkProof := &message.ChunkProof{}
	as.NoError(json.Unmarshal(chunkProofByt, chunkProof))

	chunkOk, err := v.VerifyChunkProof(chunkProof)
	as.NoError(err)
	as.True(chunkOk)

	batchProofFile, err := os.Open(*batchProofPath)
	as.NoError(err)
	batchProofByt, err := io.ReadAll(batchProofFile)
	as.NoError(err)
	batchProof := &message.BatchProof{}
	as.NoError(json.Unmarshal(batchProofByt, batchProof))

	batchOk, err := v.VerifyBatchProof(batchProof)
	as.NoError(err)
	as.True(batchOk)
}
