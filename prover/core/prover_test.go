//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core_test

import (
	"encoding/base64"
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
	assetsPath    = flag.String("assets", "/assets/test_assets", "assets dir")
	proofDumpPath = flag.String("dump", "/assets/proof_data", "the path proofs dump to")
	tracePath1    = flag.String("trace1", "/assets/traces/1_transfer.json", "chunk trace 1")
	batchVkPath   = flag.String("batch-vk", "/assets/test_assets/agg_vk.vkey", "batch vk")
	chunkVkPath   = flag.String("chunk-vk", "/assets/test_assets/chunk_vk.vkey", "chunk vk")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	chunkProverConfig := &config.ProverCoreConfig{
		DumpDir:    *proofDumpPath,
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
		ProofType:  message.ProofTypeChunk,
	}
	chunkProverCore, err := core.NewProverCore(chunkProverConfig)
	as.NoError(err)
	t.Log("Constructed chunk prover")

	as.Equal(chunkProverCore.VK, readVk(*chunkVkPath, as))
	t.Log("Chunk VK must be available when init")

	chunkTrace1 := readChunkTrace(*tracePath1, as)
	// chunkTrace2 := readChunkTrace(*tracePath2, as)
	t.Log("Loaded chunk traces")

	chunkInfo1, err := chunkProverCore.TracesToChunkInfo(chunkTrace1)
	as.NoError(err)
	// chunkInfo2, err := chunkProverCore.TracesToChunkInfo(chunkTrace2)
	// as.NoError(err)
	t.Log("Converted to chunk infos")

	chunkProof1, err := chunkProverCore.ProveChunk("chunk_proof1", chunkTrace1)
	as.NoError(err)
	t.Log("Generated and dumped chunk proof 1")

	/*
		chunkProof2, err := chunkProverCore.ProveChunk("chunk_proof2", chunkTrace2)
		as.NoError(err)
		t.Log("Generated and dumped chunk proof 2")
	*/

	as.Equal(chunkProverCore.VK, readVk(*chunkVkPath, as))
	t.Log("Chunk VKs must be equal after proving")

	batchProverConfig := &config.ProverCoreConfig{
		DumpDir:    *proofDumpPath,
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
		ProofType:  message.ProofTypeBatch,
	}
	batchProverCore, err := core.NewProverCore(batchProverConfig)
	as.NoError(err)

	as.Equal(batchProverCore.VK, readVk(*batchVkPath, as))
	t.Log("Batch VK must be available when init")

	chunkInfos := []*message.ChunkInfo{chunkInfo1}
	chunkProofs := []*message.ChunkProof{chunkProof1}
	_, err = batchProverCore.ProveBatch("batch_proof", chunkInfos, chunkProofs)
	as.NoError(err)
	t.Log("Generated and dumped batch proof")

	as.Equal(batchProverCore.VK, readVk(*batchVkPath, as))
	t.Log("Batch VKs must be equal after proving")
}

func readChunkTrace(filePat string, as *assert.Assertions) []*types.BlockTrace {
	f, err := os.Open(filePat)
	as.NoError(err)
	defer func() {
		as.NoError(f.Close())
	}()
	byt, err := io.ReadAll(f)
	as.NoError(err)

	trace := &types.BlockTrace{}
	as.NoError(json.Unmarshal(byt, trace))

	return []*types.BlockTrace{trace}
}

func readVk(filePat string, as *assert.Assertions) string {
	f, err := os.Open(filePat)
	as.NoError(err)
	defer func() {
		as.NoError(f.Close())
	}()
	byt, err := io.ReadAll(f)
	as.NoError(err)

	return base64.StdEncoding.EncodeToString(byt)
}
