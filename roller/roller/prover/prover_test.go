package prover_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"scroll-tech/go-roller/config"

	"scroll-tech/go-roller/roller/prover"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

type RpcTrace struct {
	Jsonrpc string             `json:"jsonrpc"`
	ID      int64              `json:"id"`
	Result  *types.BlockResult `json:"result"`
}

func TestFFI(t *testing.T) {
	if os.Getenv("TEST_FFI") != "true" {
		return
	}

	as := assert.New(t)
	cfg := &config.ProverConfig{
		MockMode:   false,
		ParamsPath: "../../assets/test_params",
		SeedPath:   "../../assets/test_seed",
	}
	prover, err := prover.NewProver(cfg)
	as.NoError(err)

	f, err := os.Open("../../assets/trace.json")
	as.NoError(err)
	byt, err := ioutil.ReadAll(f)
	as.NoError(err)
	rpcTrace := &RpcTrace{}
	as.NoError(json.Unmarshal(byt, rpcTrace))

	_, err = prover.Prove(rpcTrace.Result)
	as.NoError(err)
	t.Log("prove success")
}
