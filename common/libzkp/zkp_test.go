//go:build ffi

package libzkp

import (
	"encoding/json"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"path/filepath"
	"scroll-tech/common/utils"
	"testing"
)

const (
	s3Url = "https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release"

	setupVersion = "release-1220"
	paramsDir    = "./assets/test_params"
	seedPath     = "./assets/test_seed"

	circuitVersion = "release-0215"
	vkPath         = "./assets/agg_vk"

	tracesDir = "./assets/traces"
	proofPath = "./assets/agg_proof"
)

var (
	paramsUrl    = filepath.Join(s3Url, setupVersion, "params19")
	aggParamsUrl = filepath.Join(s3Url, setupVersion, "params26")
	seedUrl      = filepath.Join(s3Url, setupVersion, "test_seed")
	vkUrl        = filepath.Join(s3Url, circuitVersion, "verify_circuit.vkey")

	pvrCfg = &ProverConfig{
		ParamsPath: paramsDir,
		SeedPath:   seedPath,
	}

	vfrCfg = &VerifierConfig{
		MockMode:   false,
		ParamsPath: paramsDir,
		AggVkPath:  vkPath,
	}

	times = flag.Int("times", 1, "prove and verify times")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	var err error
	// prepare files
	os.MkdirAll(paramsDir, os.ModePerm)
	err = utils.DownloadToDir(paramsDir, paramsUrl)
	as.NoError(err)
	err = utils.DownloadToDir(paramsDir, aggParamsUrl)
	as.NoError(err)
	err = utils.DownloadFile(seedPath, seedUrl)
	as.NoError(err)
	err = utils.DownloadFile(vkPath, vkUrl)
	as.NoError(err)

	files, err := os.ReadDir(tracesDir)
	as.NoError(err)

	traces := make([]*types.BlockTrace, 0)
	for _, file := range files {
		var (
			f   *os.File
			byt []byte
		)
		f, err = os.Open(filepath.Join(tracesDir, file.Name()))
		as.NoError(err)
		byt, err = io.ReadAll(f)
		as.NoError(err)
		trace := &types.BlockTrace{}
		as.NoError(json.Unmarshal(byt, trace))
		traces = append(traces, trace)
	}

	// test prove
	pvr, err := NewProver(pvrCfg)
	as.NoError(err)
	var proof *message.AggProof
	for i := 0; i < *times; i++ {
		now := time.Now()
		proof, err = pvr.Prove(traces)
		as.NoError(err)
		t.Logf("%d: prove success! cost %f sec", i+1, time.Since(now).Seconds())
	}

	// test verify
	vfr, err := NewVerifier(vfrCfg)
	as.NoError(err)

	// clean files.
	os.RemoveAll(paramsDir)
	os.RemoveAll(seedPath)
}
