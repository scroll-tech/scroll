//go:build ffi

package verifier_test

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"scroll-tech/common/message"

	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"

	"github.com/stretchr/testify/assert"
)

const (
	ParamsPath = "./assets/test_params"
	AggVkPath  = "./assets/agg_vk"
	ProofPath  = "./assets/agg_proof"
)

func TestFFI(t *testing.T) {
	as := assert.New(t)
	cfg := &config.VerifierConfig{
		MockMode:   false,
		ParamsPath: ParamsPath,
		AggVkPath:  AggVkPath,
	}
	v, err := verifier.NewVerifier(cfg)
	as.NoError(err)

	f, err := os.Open(ProofPath)
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)
	aggProof := &message.AggProof{}
	as.NoError(json.Unmarshal(byt, aggProof))

	ok, err := v.VerifyProof(aggProof)
	as.NoError(err)
	as.True(ok)
}
