//go:build ffi

package libzkp

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/message"
	"scroll-tech/common/utils"
)

const (
	s3Url = "https://circuit-release.s3.us-west-2.amazonaws.com/circuit-release"

	setupVersion = "release-1220"
	paramsDir    = "./assets/test_params"
	seedPath     = "./assets/test_seed"

	circuitVersion = "release-0215"
	vkPath         = "./assets/agg_vk"

	tracePath = "./assets/trace.json"
	traceUrl  = "https://github.com/scroll-tech/scroll-zkevm/blob/goerli-0215/zkevm/tests/traces/erc20/single.json"
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

func prepare(as *assert.Assertions) {
	flag.Parse()
	var err error
	os.MkdirAll(paramsDir, os.ModePerm)
	_, err = utils.DownloadToDir(paramsDir, paramsUrl)
	as.NoError(err)
	_, err = utils.DownloadToDir(paramsDir, aggParamsUrl)
	as.NoError(err)
	err = utils.DownloadFile(seedPath, seedUrl)
	as.NoError(err)
	err = utils.DownloadFile(vkPath, vkUrl)
	as.NoError(err)
	err = utils.DownloadFile(tracePath, traceUrl)
	as.NoError(err)
}

func TestZkp(t *testing.T) {
	as := assert.New(t)

	prepare(as)

	var proof *message.AggProof
	// test prove
	pvr, err := NewProver(pvrCfg)
	as.NoError(err)
	for i := 0; i < *times; i++ {
		now := time.Now()
		task := &message.TaskMsg{ID: "test", Traces: traces}
		proof, err = pvr.Prove(task)
		as.NoError(err)
		t.Logf("%d: prove successfully! cost %f sec", i+1, time.Since(now).Seconds())
	}

	// test verify
	vfr, err := NewVerifier(vfrCfg)
	as.NoError(err)
	for i := 0; i < *times; i++ {
		now := time.Now()
		ok, err := vfr.VerifyProof(proof)
		as.NoError(err)
		as.True(ok)
		t.Logf("%d: verify successfully! cost %f sec", i+1, time.Since(now).Seconds())
	}
}
