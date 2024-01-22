//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core_test

import (
	"flag"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "net/http/pprof"

	"github.com/json-iterator/go"
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

var blockTracePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 4000000000) // Adjust the capacity based on your needs
	},
}

func readChunkTrace(t *testing.T, filePat string) []*types.BlockTrace {
	buf := blockTracePool.Get().([]byte)
	defer blockTracePool.Put(buf)

	byt, _ := ioutil.ReadFile(filePat)
	buf = append(buf[:0], byt...)

	trace := &types.BlockTrace{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	assert.NoError(t, json.Unmarshal(buf, trace))

	return []*types.BlockTrace{trace}
}
