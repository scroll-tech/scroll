//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
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
	paramsPath      = flag.String("params", "/assets/test_params", "params dir")
	proofDumpPath   = flag.String("dump", "/assets/proof_data", "the path proofs dump to")
	tracePath1      = flag.String("trace1", "/assets/traces/1_transfer.json", "chunk trace 1")
	tracePath2      = flag.String("trace2", "/assets/traces/10_transfer.json", "chunk trace 2")
	batchDetailPath = flag.String("batch_detail", "/assets/traces/full_proof_3.json", "batch detail")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	batchProverConfig := &config.ProverCoreConfig{
		DumpDir:    *proofDumpPath,
		ParamsPath: *paramsPath,
		ProofType:  message.ProofTypeBatch,
	}
	batchProverCore, err := core.NewProverCore(batchProverConfig)
	as.NoError(err)

	// gupeng
	batchDetail := readBatchDetail(*batchDetailPath, as)
	t.Logf("gupeng - batch-detail = %+v", batchDetail)
	chunkInfos := batchDetail.ChunkInfos
	chunkProofs := batchDetail.ChunkProofs

	_, err = batchProverCore.ProveBatch("batch_proof", chunkInfos, chunkProofs)
	as.NoError(err)
	t.Log("Generated and dumped batch proof")
}

func readBatchDetail(filePat string, as *assert.Assertions) *message.BatchTaskDetail {
	f, err := os.Open(filePat)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)

	batchDetail := &message.BatchTaskDetail{}
	as.NoError(json.Unmarshal(byt, batchDetail))

	return batchDetail
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
