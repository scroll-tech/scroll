//go:build !mock_prover

//nolint:typecheck
package prover

/*
#cgo LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl
#cgo gpu LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart
#include <stdlib.h>
#include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/message"
	"scroll-tech/common/viper"
)

// Prover sends block-traces to rust-prover through ffi and get back the zk-proof.
type Prover struct {
	vp *viper.Viper
}

// NewProver inits a Prover object.
func NewProver(vp *viper.Viper) (*Prover, error) {
	paramsPathStr := C.CString(vp.GetString("params_path"))
	seedPathStr := C.CString(vp.GetString("seed_path"))
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(seedPathStr))
	}()
	C.init_prover(paramsPathStr, seedPathStr)

	return &Prover{vp: vp}, nil
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

	log.Info("Start to create agg proof ...")
	cProof := C.create_agg_proof_multi(tracesStr)
	log.Info("Finish creating agg proof!")

	proof := C.GoString(cProof)
	zkProof := &message.AggProof{}
	err = json.Unmarshal([]byte(proof), zkProof)
	return zkProof, err
}
