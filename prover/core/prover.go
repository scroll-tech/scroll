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
	"fmt"
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

// NewProverCore inits a ProverCore object.
func NewProverCore(cfg *config.ProverCoreConfig) (*ProverCore, error) {
	paramsPathStr := C.CString(cfg.ParamsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
	}()

	if cfg.ProofType == message.ProofTypeBatch {
		C.init_batch_prover(paramsPathStr)
	} else if cfg.ProofType == message.ProofTypeChunk {
		C.init_chunk_prover(paramsPathStr)
	}

	if cfg.DumpDir != "" {
		err := os.MkdirAll(cfg.DumpDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		log.Info("Enabled dump_proof", "dir", cfg.DumpDir)
	}

	return &ProverCore{cfg: cfg}, nil
}

// ProveBatch call rust ffi to generate batch proof.
func (p *ProverCore) ProveBatch(taskID string, chunkInfos []*message.ChunkInfo, chunkProofs []*message.ChunkProof) (*message.BatchProof, error) {
	if p.cfg.ProofType != message.ProofTypeBatch {
		return nil, fmt.Errorf("prover is not a batch-prover (type: %v), but is trying to prove a batch", p.cfg.ProofType)
	}

	chunkInfosByt, err := json.Marshal(chunkInfos)
	if err != nil {
		return nil, err
	}
	chunkProofsByt, err := json.Marshal(chunkProofs)
	if err != nil {
		return nil, err
	}
	proofByt := p.proveBatch(chunkInfosByt, chunkProofsByt)

	err = p.mayDumpProof(taskID, proofByt)
	if err != nil {
		log.Error("Dump batch proof failed", "task-id", taskID, "error", err)
	}

	zkProof := &message.BatchProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

// ProveChunk call rust ffi to generate chunk proof.
func (p *ProverCore) ProveChunk(taskID string, traces []*types.BlockTrace) (*message.ChunkProof, error) {
	if p.cfg.ProofType != message.ProofTypeChunk {
		return nil, fmt.Errorf("prover is not a chunk-prover (type: %v), but is trying to prove a chunk", p.cfg.ProofType)
	}

	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return nil, err
	}
	proofByt := p.proveChunk(tracesByt)

	err = p.mayDumpProof(taskID, proofByt)
	if err != nil {
		log.Error("Dump chunk proof failed", "task-id", taskID, "error", err)
	}

	zkProof := &message.ChunkProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

// TracesToChunkHash convert traces to chunk hash
func (p *ProverCore) TracesToChunkHash(traces []*types.BlockTrace) (*message.ChunkHash, error) {
	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return nil, err
	}
	chunkHashByt := p.tracesToChunkHash(tracesByt)

	chunkHash := &message.ChunkHash{}
	return chunkHash, json.Unmarshal(chunkHashByt, chunkHash)
}

func (p *ProverCore) proveBatch(chunkInfosByt []byte, chunkProofsByt []byte) []byte {
	chunkInfosStr := C.CString(string(chunkInfosByt))
	chunkProofsStr := C.CString(string(chunkProofsByt))

	defer func() {
		C.free(unsafe.Pointer(chunkInfosStr))
		C.free(unsafe.Pointer(chunkProofsStr))
	}()

	log.Info("Start to create batch proof ...")
	cProof := C.gen_batch_proof(chunkInfosStr, chunkProofsStr)
	log.Info("Finish creating batch proof!")

	proof := C.GoString(cProof)
	return []byte(proof)
}

func (p *ProverCore) proveChunk(tracesByt []byte) []byte {
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

func (p *ProverCore) mayDumpProof(id string, proofByt []byte) error {
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

func (p *ProverCore) tracesToChunkHash(tracesByt []byte) []byte {
	tracesStr := C.CString(string(tracesByt))

	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	cChunkHash := C.block_traces_to_chunk_hash(tracesStr)

	chunkHash := C.GoString(cChunkHash)
	return []byte(chunkHash)
}
