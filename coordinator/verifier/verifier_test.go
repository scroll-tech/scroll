//go:build ffi

package verifier_test

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/message"
	"scroll-tech/common/viper"

	"scroll-tech/coordinator/verifier"
)

func TestFFI(t *testing.T) {
	as := assert.New(t)

	vp := viper.New()
	vp.Set("mock_mode", false)
	vp.Set("params_path", "../assets/test_params")
	vp.Set("agg_vk_path", "../assets/agg_vk")
	v, err := verifier.NewVerifier(vp)
	as.NoError(err)

	f, err := os.Open("../assets/agg_proof")
	as.NoError(err)
	byt, err := io.ReadAll(f)
	as.NoError(err)
	aggProof := &message.AggProof{}
	as.NoError(json.Unmarshal(byt, aggProof))

	ok, err := v.VerifyProof(aggProof)
	as.NoError(err)
	as.True(ok)
}
