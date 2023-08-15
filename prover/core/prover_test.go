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
	paramsPath          = flag.String("params", "/assets/test_params", "params dir")
	proofDumpPath       = flag.String("dump", "/assets/proof_data", "the path proofs dump to")
	batchTaskDetailPath = flag.String("batch-task-detail", "/assets/traces/full_proof_1.json", "batch-task-detail")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	batchTaskDetail := readBatchTaskDetail(*batchTaskDetailPath, as)

	batchProverConfig := &config.ProverCoreConfig{
		DumpDir:    *proofDumpPath,
		ParamsPath: *paramsPath,
		ProofType:  message.ProofTypeBatch,
	}
	batchProverCore, err := core.NewProverCore(batchProverConfig)
	as.NoError(err)

	_, err = batchProverCore.ProveBatch("batch_proof", batchTaskDetail.ChunkInfos, batchTaskDetail.ChunkProofs)
	as.NoError(err)
	t.Log("Generated and dumped batch proof")
}

func readBatchTaskDetail(filePat string, as *assert.Assertions) *message.BatchTaskDetail {
	f, err := os.Open(filePat)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)

	d := &message.BatchTaskDetail{}
	as.NoError(json.Unmarshal(byt, d))

	return d
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
