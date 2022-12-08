//go:build !mock_prover

//nolint:typecheck
package prover

/*
#cgo LDFLAGS: ${SRCDIR}/interface/libzkp.a -lm -ldl
#cgo gpu LDFLAGS: ${SRCDIR}/interface/libzkp.a -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart
#include <stdlib.h>
#include "./interface/zkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/core/types"

	"scroll-tech/common/message"

	"scroll-tech/roller/config"
)

// Prover sends block-traces to rust-prover through ffi and get back the zk-proof.
type Prover struct {
	cfg *config.ProverConfig
}

// NewProver inits a Prover object.
func NewProver(cfg *config.ProverConfig) (*Prover, error) {
	paramsPathStr := C.CString(cfg.ParamsPath)
	seedPathStr := C.CString(cfg.SeedPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(seedPathStr))
	}()
	C.init_prover(paramsPathStr, seedPathStr)

	return &Prover{cfg: cfg}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *Prover) Prove(traces []*types.BlockTrace) (*message.AggProof, error) {
	return p.prove(traces)
}

func (p *Prover) prove(traces []*types.BlockTrace) (*message.AggProof, error) {
	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return nil, err
	}
	tracesStr := C.CString(string(tracesByt))

	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()
	cProof := C.create_agg_proof_multi(tracesStr)
	proof := C.GoString(cProof)
	zkProof := &message.AggProof{}
	err = json.Unmarshal([]byte(proof), zkProof)
	return zkProof, err
}
