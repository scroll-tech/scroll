//go:build ffi

package core_test

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
	"scroll-tech/prover/core"
)

var (
	paramsPath    = flag.String("params", "/assets/test_params", "params dir")
	proofDumpPath = flag.String("dump", "/assets/proof_data", "the path proofs dump to")
	tracePath1    = flag.String("trace1", "/assets/traces/1_transfer.json", "chunk trace 1")
	tracePath2    = flag.String("trace2", "/assets/traces/10_transfer.json", "chunk trace 2")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	chunkProverConfig := &config.ProverCoreConfig{
		DumpDir:    *proofDumpPath,
		ParamsPath: *paramsPath,
		ProofType:  message.ProofTypeChunk,
	}
	chunkProverCore, err := core.NewProverCore(chunkProverConfig)
	as.NoError(err)
	t.Log("Constructed chunk prover")

	chunkTrace1 := readChunkTrace(*tracePath1, as)
	chunkTrace2 := readChunkTrace(*tracePath2, as)
	t.Log("Loaded chunk traces")

	chunkInfo1, err := chunkProverCore.TracesToChunkInfo(chunkTrace1)
	as.NoError(err)
	chunkInfo2, err := chunkProverCore.TracesToChunkInfo(chunkTrace2)
	as.NoError(err)
	t.Log("Converted to chunk infos")

	chunkProof1, err := chunkProverCore.ProveChunk("chunk_proof1", chunkTrace1)
	as.NoError(err)
	t.Log("Generated and dumped chunk proof 1")

	chunkProof2, err := chunkProverCore.ProveChunk("chunk_proof2", chunkTrace2)
	as.NoError(err)
	t.Log("Generated and dumped chunk proof 2")

	batchProverConfig := &config.ProverCoreConfig{
		DumpDir:    *proofDumpPath,
		ParamsPath: *paramsPath,
		ProofType:  message.ProofTypeBatch,
	}
	batchProverCore, err := core.NewProverCore(batchProverConfig)
	as.NoError(err)

	chunkInfos := []*message.ChunkInfo{chunkInfo1, chunkInfo2}
	chunkProofs := []*message.ChunkProof{chunkProof1, chunkProof2}
	_, err = batchProverCore.ProveBatch("batch_proof", chunkInfos, chunkProofs)
	as.NoError(err)
	t.Log("Generated and dumped batch proof")
}

func readChunkTrace(filePat string, as *assert.Assertions) []*types.BlockTrace {
	f, err := os.Open(filePat)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)

	trace := &types.BlockTrace{}
	as.NoError(json.Unmarshal(byt, trace))

	return []*types.BlockTrace{trace}
}
