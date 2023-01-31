package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"scroll-tech/common/message"
	"scroll-tech/coordinator/config"
	"scroll-tech/coordinator/verifier"
)

const (
	paramsPath = "./test_params"
	aggVkPath  = "./agg_vk"
	proofPath  = "./agg_proof"
)

func main() {
	cfg := &config.VerifierConfig{
		MockMode:   false,
		ParamsPath: paramsPath,
		AggVkPath:  aggVkPath,
	}

	pwd, _ := os.Getwd()
	fns, err := filepath.Glob(filepath.Join(pwd, "*"))
	if err != nil {
		panic(err)
	}
	for _, fn := range fns {
		fmt.Println(fn)
	}

	v, err := verifier.NewVerifier(cfg)
	if err != nil {
		panic(err)
	}

	f, err := os.Open(proofPath)
	if err != nil {
		panic(err)
	}
	byt, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	aggProof := &message.AggProof{}
	err = json.Unmarshal(byt, aggProof)
	if err != nil {
		panic(err)
	}

	ok, err := v.VerifyProof(aggProof)
	if err != nil {
		panic(err)
	}
	fmt.Println("--------------verify result is ", ok)
}
