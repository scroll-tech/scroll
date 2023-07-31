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

// ChunkProver sends block-traces to rust-prover through ffi and get back the zk-proof.
type ChunkProver struct {
	cfg *config.ChunkProverConfig
}

// NewChunkProver inits a ChunkProver object.
func NewChunkProver(cfg *config.ChunkProverConfig) (*ChunkProver, error) {
	paramsPathStr := C.CString(cfg.ParamsPath)
	assetsPathStr := C.CString(cfg.AssetsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(assetsPathStr))
	}()
	C.init_chunk_prover(paramsPathStr, assetsPathStr)

	if cfg.DumpDir != "" {
		err := os.MkdirAll(cfg.DumpDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		log.Info("Enabled dump_chunk_proof", "dir", cfg.DumpDir)
	}

	return &ChunkProver{cfg: cfg}, nil
}

// Prove call rust ffi to generate chunk proof, if first failed, try again.
func (p *ChunkProver) Prove(taskID string, traces []*types.BlockTrace) (*message.ChunkProof, error) {
	if p.cfg.ProofType != message.ProofTypeChunk {
		return nil, errors.New("Wrong proof type in chunk-prover: %d", p.cfg.ProofType)
	}

	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return nil, err
	}
	proofByt := p.prove(tracesByt)

	// dump proof
	err := p.dumpProof(taskID, proofByt)
	if err != nil {
		log.Error("Dump chunk proof failed", "task-id", taskID, "error", err)
	}

	zkProof := &message.ChunkProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

// Call cgo to generate chunk proof.
func (p *ChunkProver) prove(tracesByt []byte) []byte {
	tracesStr := C.CString(string(tracesByt))

	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Info("Start to create chunk proof ...")
	cProof := C.gen_chunk_proof(tracesStr)
	log.Info("Finish creating chunk proof!")

	proof := C.GoString(cProof)
	return []byte(proof)
}

func (p *ChunkProver) dumpProof(id string, proofByt []byte) error {
	if p.cfg.DumpDir == "" {
		return nil
	}
	path := filepath.Join(p.cfg.DumpDir, id)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	log.Info("Saving chunk proof", "task-id", id)
	_, err = f.Write(proofByt)
	return err
}
