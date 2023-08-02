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
	proofDumpPath = flag.String("dump", "/assets/proof_data", "the path proofs dump to")
	tracePath1    = flag.String("trace1", "/assets/traces/1_transfer.json", "chunk trace 1")
	tracePath2    = flag.String("trace2", "/assets/traces/10_transfer.json", "chunk trace 2")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	chunkProverConfig := &config.ProverCoreConfig{
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

	chunkProof1, err := chunkProverCore.ProveChunk("prover_test", chunkTrace1)
	as.NoError(err)
	t.Log("Generated chunk proof 1")

	chunkProof2, err := chunkProverCore.ProveChunk("prover_test", chunkTrace2)
	as.NoError(err)
	t.Log("Generated chunk proof 2")

	dumpChunkProof("chunk_proof1", chunkProof1, as)
	dumpChunkProof("chunk_proof2", chunkProof2, as)
	t.Log("Dumped chunk proofs")

	batchProverConfig := &config.ProverCoreConfig{
		ParamsPath: *paramsPath,
		ProofType:  message.ProofTypeBatch,
	}
	batchProverCore, err := core.NewProverCore(batchProverConfig)
	as.NoError(err)

	chunkInfos := []*message.ChunkInfo{chunkInfo1, chunkInfo2}
	chunkProofs := []*message.ChunkProof{chunkProof1, chunkProof2}
	batchProof, err := batchProverCore.ProveBatch("prover_test", chunkInfos, chunkProofs)
	as.NoError(err)
	t.Log("Generated batch proof")

	dumpBatchProof("batch_proof", batchProof, as)
	t.Log("Dumped batch proofs")
}

func dumpBatchProof(filename string, proof *message.BatchProof, as *assert.Assertions) {
	proofByt, err := json.Marshal(proof)
	as.NoError(err)
	f, err := os.Create(filepath.Join(*proofDumpPath, filename))
	as.NoError(err)
	_, err = f.Write(proofByt)
	as.NoError(err)
}

func dumpChunkProof(filename string, proof *message.ChunkProof, as *assert.Assertions) {
	proofByt, err := json.Marshal(proof)
	as.NoError(err)
	f, err := os.Create(filepath.Join(*proofDumpPath, filename))
	as.NoError(err)
	_, err = f.Write(proofByt)
	as.NoError(err)
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
