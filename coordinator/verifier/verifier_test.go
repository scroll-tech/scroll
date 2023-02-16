//go:build ffi

package verifier_test

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"scroll-tech/common/message"

	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"

	"github.com/stretchr/testify/assert"
)

const (
	paramsPath = "../assets/test_params"
	aggVkPath  = "../assets/agg_vk"
	proofPath  = "../assets/agg_proof"
)

var times = flag.Int("times", 1, "verifying times")

func TestFFI(t *testing.T) {
	as := assert.New(t)
	cfg := &config.VerifierConfig{
		MockMode:   false,
		ParamsPath: paramsPath,
		AggVkPath:  aggVkPath,
	}
	v, err := verifier.NewVerifier(cfg)
	as.NoError(err)

	f, err := os.Open(proofPath)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)
	aggProof := &message.AggProof{}
	as.NoError(json.Unmarshal(byt, aggProof))

	for i := 0; i < *times; i++ {
		now := time.Now()
		ok, err := v.VerifyProof(aggProof)
		as.NoError(err)
		as.True(ok)
		t.Logf("%d: verify success! cost %f sec", i+1, time.Since(now).Seconds())
	}

}
