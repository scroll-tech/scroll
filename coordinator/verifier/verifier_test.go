package verifier_test

import (
	"encoding/json"
	"io"
	"os"
	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/message"
)

func TestFFI(t *testing.T) {
	if os.Getenv("TEST_FFI") != "true" {
		return
	}

	as := assert.New(t)
	cfg := &config.VerifierConfig{
		MockMode:   false,
		ParamsPath: "../../assets/test_params",
		AggVkPath:  "../../assets/agg_vk",
	}
	v, err := verifier.NewVerifier(cfg)
	as.NoError(err)

	f, err := os.Open("../../assets/agg_proof")
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)
	aggProof := &message.AggProof{}
	as.NoError(json.Unmarshal(byt, aggProof))

	ok, err := v.VerifyProof(aggProof)
	as.NoError(err)
	as.True(ok)
}