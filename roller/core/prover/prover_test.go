package prover_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"

	"scroll-tech/go-roller/config"
	"scroll-tech/go-roller/core/prover"
)

type RPCTrace struct {
	Jsonrpc string             `json:"jsonrpc"`
	ID      int64              `json:"id"`
	Result  *types.BlockResult `json:"result"`
}

func TestFFI(t *testing.T) {
	as := assert.New(t)
	cfg := &config.ProverConfig{
		MockMode:   true,
		ParamsPath: "../../assets/test_params",
		SeedPath:   "../../assets/test_seed",
	}
	prover, err := prover.NewProver(cfg)
	as.NoError(err)

	f, err := os.Open("../../assets/trace.json")
	as.NoError(err)
	byt, err := ioutil.ReadAll(f)
	as.NoError(err)
	rpcTrace := &RPCTrace{}
	as.NoError(json.Unmarshal(byt, rpcTrace))

	_, err = prover.Prove(rpcTrace.Result)
	as.NoError(err)
	t.Log("prove success")
}
