//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core_test

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
	batchDirPath  = flag.String("batch-dir", "/assets/traces/batch_24", "batch directory")
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

	// Get the list of subdirectories (chunks)
	chunkDirs, err := os.ReadDir(*batchDirPath)
	as.NoError(err)
	sort.Slice(chunkDirs, func(i, j int) bool {
		return chunkDirs[i].Name() < chunkDirs[j].Name()
	})

	chunkInfos := make([]*message.ChunkInfo, 0, len(chunkDirs))
	chunkProofs := make([]*message.ChunkProof, 0, len(chunkDirs))

	for i, dir := range chunkDirs {
		if dir.IsDir() {
			chunkPath := filepath.Join(*batchDirPath, dir.Name())

			chunkTrace := readChunkTrace(chunkPath, as)
			t.Logf("Loaded chunk trace %d", i+1)

			chunkInfo, err := chunkProverCore.TracesToChunkInfo(chunkTrace)
			as.NoError(err)
			chunkInfos = append(chunkInfos, chunkInfo)
			t.Logf("Converted to chunk info %d", i+1)

			chunkProof, err := chunkProverCore.ProveChunk(fmt.Sprintf("chunk_proof%d", i+1), chunkTrace)
			as.NoError(err)
			chunkProofs = append(chunkProofs, chunkProof)
			t.Logf("Generated and dumped chunk proof %d", i+1)
		}
	}

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

	_, err = batchProverCore.ProveBatch("batch_proof", chunkInfos, chunkProofs)
	as.NoError(err)
	t.Log("Generated and dumped batch proof")

	as.Equal(batchProverCore.VK, readVk(*batchVkPath, as))
	t.Log("Batch VKs must be equal after proving")
}
func readChunkTrace(filePat string, as *assert.Assertions) []*types.BlockTrace {
	fileInfo, err := os.Stat(filePat)
	as.NoError(err)

	var traces []*types.BlockTrace

	readFile := func(path string) {
		f, err := os.Open(path)
		as.NoError(err)
		defer func() {
			as.NoError(f.Close())
		}()
		byt, err := io.ReadAll(f)
		as.NoError(err)

		trace := &types.BlockTrace{}
		as.NoError(json.Unmarshal(byt, trace))

		traces = append(traces, trace)
	}

	if fileInfo.IsDir() {
		files, err := os.ReadDir(filePat)
		as.NoError(err)

		// Sort files alphabetically
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() < files[j].Name()
		})

		for _, file := range files {
			if !file.IsDir() {
				readFile(filepath.Join(filePat, file.Name()))
			}
		}
	} else {
		readFile(filePat)
	}

	return traces
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
