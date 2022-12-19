//go:build ffi

package verifier_test

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"scroll-tech/common/message"

	"scroll-tech/coordinator/verifier"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestFFI(t *testing.T) {
	as := assert.New(t)
	viper.Set("mock_mode", false)
	viper.Set("params_path", "../assets/test_params")
	viper.Set("agg_vk_path", "../assets/agg_vk")
	v, err := verifier.NewVerifier(viper.Sub("roller_manager_config.verifier"))
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
