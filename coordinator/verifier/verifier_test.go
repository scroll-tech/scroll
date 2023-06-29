//go:build ffi

package verifier_test

import (
	"flag"
	"testing"
)

var (
	paramsPath = flag.String("params", "/assets/test_params", "params dir")
	aggVkPath  = flag.String("vk", "/assets/agg_vk", "aggregator verify key")
	proofPath  = flag.String("proof", "/assets/agg_proof", "aggregator proof")
)

func TestFFI(t *testing.T) {
	t.Log("paramsPath: ", paramsPath)
	t.Log("aggVkPath", aggVkPath)
	t.Log("proofPath", proofPath)
	//as := assert.New(t)
	//cfg := &config.VerifierConfig{
	//	MockMode:   false,
	//	ParamsPath: *paramsPath,
	//	AggVkPath:  *aggVkPath,
	//}
	//v, err := verifier.NewVerifier(cfg)
	//as.NoError(err)
	//
	//f, err := os.Open(*proofPath)
	//as.NoError(err)
	//byt, err := io.ReadAll(f)
	//as.NoError(err)
	//aggProof := &message.AggProof{}
	//as.NoError(json.Unmarshal(byt, aggProof))
	//
	//ok, err := v.VerifyProof(aggProof)
	//as.NoError(err)
	//as.True(ok)
}
