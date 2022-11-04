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
	"scroll-tech/roller/core/prover"
)

const (
	ParamsPath = "../../assets/test_params"
	SeedPath   = "../../assets/test_seed"
	TracesPath = "../../assets/traces"
)

type RPCTrace struct {
	Jsonrpc string             `json:"jsonrpc"`
	ID      int64              `json:"id"`
	Result  *types.BlockResult `json:"result"`
}

func TestFFI(t *testing.T) {
	if os.Getenv("TEST_FFI") != "true" {
		t.Skip("Skipping testing FFI")
	}

	as := assert.New(t)
	cfg := &config.ProverConfig{
		MockMode:   false,
		ParamsPath: ParamsPath,
		SeedPath:   SeedPath,
	}
	prover, err := prover.NewProver(cfg)
	as.NoError(err)

	files, err := os.ReadDir(TracesPath)
	as.NoError(err)

	traces := make([]*types.BlockResult, 0)
	for _, file := range files {
		t.Log("add trace: ", file.Name())
		f, err := os.Open(filepath.Join(TracesPath, file.Name()))
		as.NoError(err)
		byt, err := io.ReadAll(f)
		as.NoError(err)
		rpcTrace := &RPCTrace{}
		as.NoError(json.Unmarshal(byt, rpcTrace))
		traces = append(traces, rpcTrace.Result)
	}
	_, err = prover.Prove(traces)
	as.NoError(err)
	t.Log("prove success")
}
