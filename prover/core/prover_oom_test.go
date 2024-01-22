//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core_test

import (
	"flag"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/grafana/pyroscope-go"
	_ "net/http/pprof"

	"github.com/json-iterator/go"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

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
	pyroscope.Start(pyroscope.Config{
		ApplicationName: "prover-idc-us-19",

		ServerAddress: "http://127.0.0.1:4040",

		// you can disable logging by setting this to nil
		Logger: pyroscope.StandardLogger,

		// you can provide static tags via a map:
		Tags: map[string]string{"hostname": os.Getenv("HOSTNAME")},

		ProfileTypes: []pyroscope.ProfileType{
			// these profile types are enabled by default:
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,

			// these profile types are optional:
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
}

func TestFFI(t *testing.T) {
	initPyroscopse()

	as := assert.New(t)

	chunkProverConfig := &config.ProverCoreConfig{
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
		ProofType:  message.ProofTypeChunk,
	}

	chunkProverCore, _ := core.NewProverCore(chunkProverConfig)
	chunkTrace1 := readChunkTrace(*tracePath1, as)

	for {
		chunkProverCore.ProveChunk("chunk_proof1", chunkTrace1)
		time.Sleep(time.Millisecond * 100)
	}
}

var blockTracePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 4000000000) // Adjust the capacity based on your needs
	},
}

func readChunkTrace(filePat string, as *assert.Assertions) []*types.BlockTrace {
	buf := blockTracePool.Get().([]byte)
	defer blockTracePool.Put(buf)

	byt, _ := ioutil.ReadFile(filePat)
	buf = append(buf[:0], byt...)

	trace := &types.BlockTrace{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	as.NoError(json.Unmarshal(buf, trace))

	return []*types.BlockTrace{trace}
}
