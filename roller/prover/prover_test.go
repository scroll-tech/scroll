//go:build ffi

package prover_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/roller/config"
	"scroll-tech/roller/prover"
)

const (
	paramsPath = "../assets/test_params"
	seedPath   = "../assets/test_seed"
	tracesPath = "../assets/traces"
)

type RPCTrace struct {
	Jsonrpc string            `json:"jsonrpc"`
	ID      int64             `json:"id"`
	Result  *types.BlockTrace `json:"result"`
}

func TestFFI(t *testing.T) {
	as := assert.New(t)
	cfg := &config.ProverConfig{
		ParamsPath: paramsPath,
		SeedPath:   seedPath,
	}
	prover, err := prover.NewProver(cfg)
	as.NoError(err)

	files, err := os.ReadDir(tracesPath)
	as.NoError(err)

	traces := make([]*types.BlockTrace, 0)
	for _, file := range files {
		var (
			f   *os.File
			byt []byte
		)
		f, err = os.Open(filepath.Join(tracesPath, file.Name()))
		as.NoError(err)
		byt, err = io.ReadAll(f)
		as.NoError(err)
		rpcTrace := &RPCTrace{}
		as.NoError(json.Unmarshal(byt, rpcTrace))
		traces = append(traces, rpcTrace.Result)
	}
	_, err = prover.Prove(traces)
	as.NoError(err)
	t.Log("prove success")
}
