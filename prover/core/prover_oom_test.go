//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core_test

import (
	"flag"
	"sync"
	"testing"

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

func TestFFI(t *testing.T) {
	as := assert.New(t)

	chunkProverConfig := &config.ProverCoreConfig{
		ParamsPath: *paramsPath,
		AssetsPath: *assetsPath,
		ProofType:  message.ProofTypeChunk,
	}

	chunkProverCore, _ := core.NewProverCore(chunkProverConfig)
	chunkTrace1 := readChunkTrace(*tracePath1, as)

	for i := 1; i <= 2000; i++ {
		t.Log("Proof-", i, " BEGIN mem: ", memUsage(as))
		chunkProverCore.ProveChunk("chunk_proof1", chunkTrace1)
		t.Log("Proof-", i, " END mem: ", memUsage(as))
	}
}

var blockTracePool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 4000000000) // Adjust the capacity based on your needs
	},
}

func readChunkTrace(filePat string, as *assert.Assertions) []*types.BlockTrace {
	/* gupeng

	buf := blockTracePool.Get().([]byte)
	defer blockTracePool.Put(buf)

	byt, _ := ioutil.ReadFile(filePat)
	buf = append(buf[:0], byt...)

	trace := &types.BlockTrace{}
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	as.NoError(json.Unmarshal(buf, trace))

	return []*types.BlockTrace{trace}

	*/
	return []*types.BlockTrace{}
}
