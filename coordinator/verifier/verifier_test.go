//go:build ffi

package verifier_test

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"testing"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"

	"github.com/stretchr/testify/assert"
)

var (
	paramsPath = flag.String("params", "/assets/test_params", "params dir")
	aggVkPath  = flag.String("vk", "/assets/agg_vk", "aggregation proof verification key path")
	proofPath  = flag.String("proof", "/assets/agg_proof", "aggregation proof path")
)

func TestFFI(t *testing.T) {
	as := assert.New(t)
	cfg := &config.VerifierConfig{
		MockMode:   false,
		ParamsPath: *paramsPath,
		AggVkPath:  *aggVkPath,
	}
	v, err := verifier.NewVerifier(cfg)
	as.NoError(err)

	f, err := os.Open(*proofPath)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)
	aggProof := &message.AggProof{}
	as.NoError(json.Unmarshal(byt, aggProof))

	ok, err := v.VerifyProof(aggProof)
	as.NoError(err)
	as.True(ok)
}
