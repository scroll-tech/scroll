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

	scrollTypes "scroll-tech/common/types"
	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
	"scroll-tech/prover/core"
)

var (
	paramsPath    = flag.String("params", "/assets/test_params", "params dir")
	assetsPath    = flag.String("assets", "/assets/test_assets", "assets dir")
	proofDumpPath = flag.String("dump", "/assets/proof_data", "the path proofs dump to")
	tracePath1    = flag.String("trace1", "/assets/traces/1_transfer.json", "chunk trace 1")
	tracePath2    = flag.String("trace2", "/assets/traces/10_transfer.json", "chunk trace 2")
	batchVkPath   = flag.String("batch-vk", "/assets/test_assets/agg_vk.vkey", "batch vk")
	chunkVkPath   = flag.String("chunk-vk", "/assets/test_assets/chunk_vk.vkey", "chunk vk")
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

	wrappedBlock1 := &scrollTypes.WrappedBlock{
		Header:       chunkTrace1[0].Header,
		Transactions: chunkTrace1[0].Transactions,
		WithdrawRoot: chunkTrace1[0].WithdrawTrieRoot,
	}
	chunk1 := &scrollTypes.Chunk{Blocks: []*scrollTypes.WrappedBlock{wrappedBlock1}}
	chunkHash1, err := chunk1.Hash(0)
	as.NoError(err)
	as.Equal(chunkInfo1.PostStateRoot, wrappedBlock1.Header.Root)
	as.Equal(chunkInfo1.WithdrawRoot, wrappedBlock1.WithdrawRoot)
	as.Equal(chunkInfo1.DataHash, chunkHash1)
	t.Log("Successful to check chunk info 1")

	wrappedBlock2 := &scrollTypes.WrappedBlock{
		Header:       chunkTrace2[0].Header,
		Transactions: chunkTrace2[0].Transactions,
		WithdrawRoot: chunkTrace2[0].WithdrawTrieRoot,
	}
	chunk2 := &scrollTypes.Chunk{Blocks: []*scrollTypes.WrappedBlock{wrappedBlock2}}
	chunkHash2, err := chunk2.Hash(chunk1.NumL1Messages(0))
	as.NoError(err)
	as.Equal(chunkInfo2.PostStateRoot, wrappedBlock2.Header.Root)
	as.Equal(chunkInfo2.WithdrawRoot, wrappedBlock2.WithdrawRoot)
	as.Equal(chunkInfo2.DataHash, chunkHash2)
	t.Log("Successful to check chunk info 2")

	chunkProof1, err := chunkProverCore.ProveChunk("chunk_proof1", chunkTrace1)
	as.NoError(err)
	t.Log("Generated and dumped chunk proof 1")

	chunkProof2, err := chunkProverCore.ProveChunk("chunk_proof2", chunkTrace2)
	as.NoError(err)
	t.Log("Generated and dumped chunk proof 2")

	as.Equal(chunkProverCore.GetVk(), readVk(*chunkVkPath, as))
	t.Log("Chunk VKs are equal")

	batchProverConfig := &config.ProverCoreConfig{
		DumpDir:    *proofDumpPath,
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
		ProofType:  message.ProofTypeBatch,
	}
	batchProverCore, err := core.NewProverCore(batchProverConfig)
	as.NoError(err)

	chunkInfos := []*message.ChunkInfo{chunkInfo1, chunkInfo2}
	chunkProofs := []*message.ChunkProof{chunkProof1, chunkProof2}
	_, err = batchProverCore.ProveBatch("batch_proof", chunkInfos, chunkProofs)
	as.NoError(err)
	t.Log("Generated and dumped batch proof")

	batchVk1 := batchProverCore.GetVk()
	batchVk2 := readVk(*batchVkPath, as)
	t.Logf("gupeng - batchVk1 = %s", batchVk1)
	t.Logf("gupeng - batchVk2 = %s", batchVk2)
	// as.Equal(batchProverCore.GetVk(), readVk(*batchVkPath, as))
	as.Equal(batchVk1, batchVk2)
	t.Log("Batch VKs are equal")
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

func readVk(filePat string, as *assert.Assertions) string {
	f, err := os.Open(filePat)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)

	return base64.StdEncoding.EncodeToString(byt)
}
