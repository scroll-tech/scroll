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
	vk  string
}

// NewProverCore inits a ProverCore object.
func NewProverCore(cfg *config.ProverCoreConfig) (*ProverCore, error) {
	paramsPathStr := C.CString(cfg.ParamsPath)
	assetsPathStr := C.CString(cfg.AssetsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(assetsPathStr))
	}()

	if cfg.ProofType == message.ProofTypeBatch {
		C.init_batch_prover(paramsPathStr, assetsPathStr)
	} else if cfg.ProofType == message.ProofTypeChunk {
		C.init_chunk_prover(paramsPathStr, assetsPathStr)
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

// GetVk get Base64 format of vk.
func (p *ProverCore) GetVk() string {
	if p.vk != "" { // cached
		return p.vk
	}

	var raw *C.char
	if p.cfg.ProofType == message.ProofTypeBatch {
		raw = C.get_batch_vk()
	} else if p.cfg.ProofType == message.ProofTypeChunk {
		raw = C.get_chunk_vk()
	}

	if raw != nil {
		p.vk = C.GoString(raw) // cache it
	}

	return p.vk
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

	isValid, err := p.checkChunkProofs(chunkProofsByt)
	if err != nil {
		return nil, err
	}

	if !isValid {
		return nil, fmt.Errorf("non-match chunk protocol, task-id: %s", taskID)
	}

	proofByt, err := p.proveBatch(chunkInfosByt, chunkProofsByt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate batch proof: %v", err)
	}

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
	proofByt, err := p.proveChunk(tracesByt)
	if err != nil {
		return nil, err
	}

	err = p.mayDumpProof(taskID, proofByt)
	if err != nil {
		log.Error("Dump chunk proof failed", "task-id", taskID, "error", err)
	}

	zkProof := &message.ChunkProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

// TracesToChunkInfo convert traces to chunk info
func (p *ProverCore) TracesToChunkInfo(traces []*types.BlockTrace) (*message.ChunkInfo, error) {
	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return nil, err
	}
	chunkInfoByt := p.tracesToChunkInfo(tracesByt)

	chunkInfo := &message.ChunkInfo{}
	return chunkInfo, json.Unmarshal(chunkInfoByt, chunkInfo)
}

// CheckChunkProofsResponse represents the result of a chunk proof checking operation.
// Ok indicates whether the proof checking was successful.
// Error provides additional details in case the check failed.
type CheckChunkProofsResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// ProofResult encapsulates the result from generating a proof.
// Message holds the generated proof in byte slice format.
// Error provides additional details in case the proof generation failed.
type ProofResult struct {
	Message []byte `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (p *ProverCore) checkChunkProofs(chunkProofsByt []byte) (bool, error) {
	chunkProofsStr := C.CString(string(chunkProofsByt))
	defer C.free(unsafe.Pointer(chunkProofsStr))

	log.Info("Start to check chunk proofs ...")
	cResult := C.check_chunk_proofs(chunkProofsStr)
	defer C.free(unsafe.Pointer(cResult))
	log.Info("Finish checking chunk proofs!")

	var result CheckChunkProofsResponse
	err := json.Unmarshal([]byte(C.GoString(cResult)), &result)
	if err != nil {
		return false, fmt.Errorf("failed to parse check chunk proofs result: %v", err)
	}

	if result.Error != "" {
		return false, fmt.Errorf("failed to check chunk proofs: %s", result.Error)
	}

	return result.Ok, nil
}

func (p *ProverCore) proveBatch(chunkInfosByt []byte, chunkProofsByt []byte) ([]byte, error) {
	chunkInfosStr := C.CString(string(chunkInfosByt))
	chunkProofsStr := C.CString(string(chunkProofsByt))

	defer func() {
		C.free(unsafe.Pointer(chunkInfosStr))
		C.free(unsafe.Pointer(chunkProofsStr))
	}()

	log.Info("Start to create batch proof ...")
	bResult := C.gen_batch_proof(chunkInfosStr, chunkProofsStr)
	defer C.free(unsafe.Pointer(bResult))
	log.Info("Finish creating batch proof!")

	var result ProofResult
	err := json.Unmarshal([]byte(C.GoString(bResult)), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse batch proof result: %v", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("failed to generate batch proof: %s", result.Error)
	}

	return result.Message, nil
}

func (p *ProverCore) proveChunk(tracesByt []byte) ([]byte, error) {
	tracesStr := C.CString(string(tracesByt))
	defer C.free(unsafe.Pointer(tracesStr))

	log.Info("Start to create chunk proof ...")
	cProof := C.gen_chunk_proof(tracesStr)
	defer C.free(unsafe.Pointer(cProof))
	log.Info("Finish creating chunk proof!")

	var result ProofResult
	err := json.Unmarshal([]byte(C.GoString(cProof)), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chunk proof result: %v", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("failed to generate chunk proof: %s", result.Error)
	}

	return result.Message, nil
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

func (p *ProverCore) tracesToChunkInfo(tracesByt []byte) []byte {
	tracesStr := C.CString(string(tracesByt))

	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	cChunkInfo := C.block_traces_to_chunk_info(tracesStr)

	chunkInfo := C.GoString(cChunkInfo)
	return []byte(chunkInfo)
}
