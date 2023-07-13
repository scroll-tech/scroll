//go:build ffi

package verifier

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"testing"

	"scroll-tech/common/types/message"

<<<<<<< HEAD:coordinator/internal/logic/verifier/verifier_test.go
	"scroll-tech/coordinator/config"
=======
	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/verifier"
>>>>>>> 6841ef264c163c158446d94d8ea48336aca8498e:coordinator/verifier/verifier_test.go

	"github.com/stretchr/testify/assert"
)

<<<<<<< HEAD:coordinator/internal/logic/verifier/verifier_test.go
const (
	paramsPath = "./assets/test_params"
	aggVkPath  = "./assets/agg_vk"
	proofPath  = "./assets/agg_proof"
=======
var (
	paramsPath = flag.String("params", "/assets/test_params", "params dir")
	aggVkPath  = flag.String("vk", "/assets/agg_vk", "aggregation proof verification key path")
	proofPath  = flag.String("proof", "/assets/agg_proof", "aggregation proof path")
>>>>>>> 6841ef264c163c158446d94d8ea48336aca8498e:coordinator/verifier/verifier_test.go
)

func TestFFI(t *testing.T) {
	as := assert.New(t)
	cfg := &config.VerifierConfig{
		MockMode:   false,
		ParamsPath: *paramsPath,
		AggVkPath:  *aggVkPath,
	}
	v, err := NewVerifier(cfg)
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
