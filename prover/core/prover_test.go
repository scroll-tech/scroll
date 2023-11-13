//go:build ffi

// go test -v -race -gcflags="-l" -ldflags="-s=false" -tags ffi ./...
package core_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"runtime"
	"sync"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	jsoniter "github.com/json-iterator/go"

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

func memUsage(as *assert.Assertions) string {
	mem := "echo \"$(date '+%Y-%m-%d %H:%M:%S') $(free -g | grep Mem: | sed 's/Mem://g')\""
	cmd := exec.Command("bash", "-c", mem)

	output, err := cmd.Output()
	as.NoError(err)

	return string(output)
}

func printGC(msg string) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	fmt.Printf("%s:\n", msg)
	fmt.Printf("  Alloc: %v MB\n", bToMb(stats.Alloc))
	fmt.Printf("  TotalAlloc: %v MB\n", bToMb(stats.TotalAlloc))
	fmt.Printf("  Sys: %v MB\n", bToMb(stats.Sys))
	fmt.Printf("  NumGC: %v\n", stats.NumGC)
	fmt.Printf("  NumGoroutine: %v\n", runtime.NumGoroutine())
	fmt.Printf("  HeapAlloc: %v MB\n", bToMb(stats.HeapAlloc))
	fmt.Printf("  HeapSys: %v MB\n", bToMb(stats.HeapSys))
	fmt.Println()
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
