//go:build ffi

package core_test

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
	"scroll-tech/prover/core"
)

var (
	paramsPath    = flag.String("params", "/assets/test_params", "params dir")
	tracesPath    = flag.String("traces", "/assets/traces", "traces dir")
	proofDumpPath = flag.String("dump", "/assets/proof_data", "the path proofs dump to")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	chunkProverConfig := &config.ProverCoreConfig{
		ParamsPath: *paramsPath,
		ProofType:  message.ProofTypeChunk,
	}
	chunkProverCore, err := core.NewProverCore(chunkProverConfig)
	as.NoError(err)

	files, err := os.ReadDir(*tracesPath)
	as.NoError(err)

	traces := make([]*types.BlockTrace, 0)
	for _, file := range files {
		var (
			f   *os.File
			byt []byte
		)
		f, err = os.Open(filepath.Join(*tracesPath, file.Name()))
		as.NoError(err)
		byt, err = io.ReadAll(f)
		as.NoError(err)
		trace := &types.BlockTrace{}
		as.NoError(json.Unmarshal(byt, trace))
		traces = append(traces, trace)
	}

	chunkInfo, err := chunkProverCore.TracesToChunkInfo(traces)
	as.NoError(err)
	t.Log("Generated chunk hash")

	chunkProof, err := chunkProverCore.ProveChunk("test", traces)
	as.NoError(err)
	t.Log("Generated chunk proof")

	chunkProofByt, err := json.Marshal(chunkProof)
	as.NoError(err)
	chunkProofFile, err := os.Create(filepath.Join(*proofDumpPath, "chunk_proof"))
	as.NoError(err)
	_, err = chunkProofFile.Write(chunkProofByt)
	as.NoError(err)
	t.Log("Dumped chunk proof")

	batchProverConfig := &config.ProverCoreConfig{
		ParamsPath: *paramsPath,
		ProofType:  message.ProofTypeBatch,
	}
	batchProverCore, err := core.NewProverCore(batchProverConfig)
	as.NoError(err)

	chunkInfos := make([]*message.ChunkInfo, 0)
	chunkInfos = append(chunkInfos, chunkInfo)
	chunkProofs := make([]*message.ChunkProof, 0)
	chunkProofs = append(chunkProofs, chunkProof)
	batchProof, err := batchProverCore.ProveBatch("test", chunkInfos, chunkProofs)
	as.NoError(err)
	t.Log("Generated batch proof")

	batchProofByt, err := json.Marshal(batchProof)
	as.NoError(err)
	batchProofFile, err := os.Create(filepath.Join(*proofDumpPath, "batch_proof"))
	as.NoError(err)
	_, err = batchProofFile.Write(batchProofByt)
	as.NoError(err)
	t.Log("Dumped batch proof")
}
