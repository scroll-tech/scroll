//go:build !mock_zkp

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
	"os"
	"path/filepath"
	"unsafe"

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

	if cfg.DumpDir != "" {
		err := os.MkdirAll(cfg.DumpDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		log.Info("Enabled dump_proof", "dir", cfg.DumpDir)
	}

	return &Prover{cfg: cfg}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *Prover) Prove(task *message.TaskMsg) (*message.AggProof, error) {
	tracesByt, err := json.Marshal(task.Traces)
	if err != nil {
		return nil, err
	}

	proofByt := p.prove(tracesByt)

	// dump proof
	err = p.dumpProof(task.ID, proofByt)
	if err != nil {
		log.Error("Dump proof failed", "task-id", task.ID, "error", err)
	}

	zkProof := &message.AggProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

func (p *Prover) prove(tracesByt []byte) []byte {
	tracesStr := C.CString(string(tracesByt))

	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Info("Start to create agg proof ...")
	cProof := C.create_agg_proof_multi(tracesStr)
	log.Info("Finish creating agg proof!")

	proof := C.GoString(cProof)
	return []byte(proof)
}

func (p *Prover) dumpProof(id string, proofByt []byte) error {
	if p.cfg.DumpDir == "" {
		return nil
	}
	path := filepath.Join(p.cfg.DumpDir, id)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	log.Info("Saving proof", "task-id", id)
	_, err = f.Write(proofByt)
	return err
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
