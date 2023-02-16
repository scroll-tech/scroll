//go:build ffi

package prover_test

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/roller/config"
	"scroll-tech/roller/prover"
)

const (
	paramsPath    = "../assets/test_params"
	seedPath      = "../assets/test_seed"
	tracesPath    = "../assets/traces"
	proofDumpPath = "agg_proof"
)

var times = flag.Int("times", 1, "proving times")

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

	for i := 0; i < times; i++ {
		now := time.Now()
		proof, err := prover.Prove(traces)
		as.NoError(err)
		t.Logf("%d: prove success! cost %f sec", i, time.Since(now).Seconds())
	}

	// dump the proof
	os.RemoveAll(proofDumpPath)
	proofByt, err := json.Marshal(proof)
	as.NoError(err)
	proofFile, err := os.Create(proofDumpPath)
	as.NoError(err)
	_, err = proofFile.Write(proofByt)
	as.NoError(err)
}
