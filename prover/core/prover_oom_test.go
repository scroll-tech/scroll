//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core

import (
	"encoding/json"
	"flag"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
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
	ballast := make([]byte, 100*1024*1024*1024) // 10G
	initPyroscopse()

	chunkProverConfig := &config.ProverCoreConfig{
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
		ProofType:  message.ProofTypeChunk,
	}

	chunkProverCore, _ := NewProverCore(chunkProverConfig)

	for {
		chunkProverCore.proveChunk()
	}
	runtime.KeepAlive(ballast)
}

func readChunkTrace(t *testing.T, filePat string) []*types.BlockTrace {
	byt, err := ioutil.ReadFile(filePat)
	assert.NoError(t, err)

	trace := &types.BlockTrace{}
	assert.NoError(t, json.Unmarshal(byt, trace))

	return []*types.BlockTrace{trace}
}
