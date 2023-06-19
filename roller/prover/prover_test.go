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

// These files should be in scroll/assets/...
const (
	paramsPath    = "../../assets/test_params"
	seedPath      = "../../assets/test_seed"
	tracesPath    = "../../assets/traces"
	proofDumpPath = "../../assets/agg_proof"
)

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
		trace := &types.BlockTrace{}
		as.NoError(json.Unmarshal(byt, trace))
		traces = append(traces, trace)
	}
	proof, err := prover.Prove("test", traces)
	as.NoError(err)
	t.Log("prove success")

	// dump the proof
	os.RemoveAll(proofDumpPath)
	proofByt, err := json.Marshal(proof)
	as.NoError(err)
	proofFile, err := os.Create(proofDumpPath)
	as.NoError(err)
	_, err = proofFile.Write(proofByt)
	as.NoError(err)
}
