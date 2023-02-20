//go:build !mock

//nolint:typecheck
package libzkp

/*
#cgo LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl -lzktrie -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl -lgmp -lstdc++ -lprocps -lzktrie -L/usr/local/cuda/lib64/ -L${SRCDIR}/lib/ -lcudart -Wl,-rpath=${SRCDIR}/lib
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
)

// Prover sends block-traces to rust-prover through ffi and get back the zk-proof.
type Prover struct {
	cfg *ProverConfig
}

// NewProver inits a Prover object.
func NewProver(cfg *ProverConfig) (*Prover, error) {
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

	log.Info("Start to create agg proof ...")
	cProof := C.create_agg_proof_multi(tracesStr)
	log.Info("Finish creating agg proof!")

	proof := C.GoString(cProof)
	zkProof := &message.AggProof{}
	err = json.Unmarshal([]byte(proof), zkProof)
	return zkProof, err
}

// Verifier represents a rust ffi to a halo2 verifier.
type Verifier struct {
	cfg *VerifierConfig
}

// NewVerifier Sets up a rust ffi to call verify.
func NewVerifier(cfg *VerifierConfig) (*Verifier, error) {
	if cfg.MockMode {
		return &Verifier{cfg: cfg}, nil
	}
	paramsPathStr := C.CString(cfg.ParamsPath)
	aggVkPathStr := C.CString(cfg.AggVkPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(aggVkPathStr))
	}()

	C.init_verifier(paramsPathStr, aggVkPathStr)

	return &Verifier{cfg: cfg}, nil
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Verifier disabled, VerifyProof skipped")
		return true, nil

	}
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	aggProofStr := C.CString(string(buf))
	defer func() {
		C.free(unsafe.Pointer(aggProofStr))
	}()

	log.Info("Start to verify proof ...")
	verified := C.verify_agg_proof(aggProofStr)
	return verified != 0, nil
}
