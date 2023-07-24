//go:build !mock_prover

package core

/*
#cgo LDFLAGS: -lzkp -lm -ldl -lzktrie -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: -lzkp -lm -ldl -lgmp -lstdc++ -lprocps -lzktrie -L/usr/local/cuda/lib64/ -lcudart -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#include <stdlib.h>
#include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
)

// ProverCore sends block-traces to rust-prover through ffi and get back the zk-proof.
type ProverCore struct {
	cfg *config.ProverCoreConfig
}

// NewProverCore inits a Prover object.
func NewProverCore(cfg *config.ProverCoreConfig) (*ProverCore, error) {
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

	return &ProverCore{cfg: cfg}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *ProverCore) Prove(taskID string, traces []*types.BlockTrace) (*message.AggProof, error) {
	var proofByt []byte
	if p.cfg.ProofType == message.ProofTypeChunk {
		tracesByt, err := json.Marshal(traces)
		if err != nil {
			return nil, err
		}
		proofByt = p.prove(tracesByt)
	} else if p.cfg.ProofType == message.ProofTypeBatch {
		// TODO: aggregator prove
	}

	// dump proof
	err := p.dumpProof(taskID, proofByt)
	if err != nil {
		log.Error("Dump proof failed", "task-id", taskID, "error", err)
	}

	zkProof := &message.AggProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

// Call cgo to generate proof.
func (p *ProverCore) prove(tracesByt []byte) []byte {
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

func (p *ProverCore) dumpProof(id string, proofByt []byte) error {
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
