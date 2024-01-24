//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core_test

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "net/http/pprof"

	"github.com/scroll-tech/go-ethereum/core/types"
	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
	"scroll-tech/prover/core"
)

var (
	paramsPath = flag.String("params", "/assets/test_params", "params dir")
	assetsPath = flag.String("assets", "/assets/test_assets", "assets dir")
	tracePath1 = flag.String("trace1", "/assets/traces/1_transfer.json", "chunk trace 1")
)

func initPyroscopse() {
	go func() {
		if runServerErr := http.ListenAndServe(":8089", nil); runServerErr != nil {
			panic(runServerErr)
		}
	}()
}

func TestFFI(t *testing.T) {
	initPyroscopse()

	chunkProverConfig := &config.ProverCoreConfig{
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
		ProofType:  message.ProofTypeChunk,
	}

	chunkProverCore, _ := core.NewProverCore(chunkProverConfig)
	chunkTrace1 := readChunkTrace(t, *tracePath1)

	for {
		chunkProverCore.ProveChunk("chunk_proof1", chunkTrace1)
		time.Sleep(time.Millisecond * 10)
	}
}

func readChunkTrace(t *testing.T, filePat string) []*types.BlockTrace {
	byt, err := ioutil.ReadFile(filePat)
	assert.NoError(t, err)

	trace := &types.BlockTrace{}
	assert.NoError(t, json.Unmarshal(byt, trace))

	return []*types.BlockTrace{trace}
}
