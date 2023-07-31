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

// BatchProver sends block-traces to rust-prover through ffi and get back the zk-proof.
type BatchProver struct {
	cfg *config.BatchProverConfig
}

// NewBatchProver inits a BatchProver object.
func NewBatchProver(cfg *config.BatchProverConfig) (*BatchProver, error) {
	paramsPathStr := C.CString(cfg.ParamsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
	}()
	C.init_batch_prover(paramsPathStr)

	if cfg.DumpDir != "" {
		err := os.MkdirAll(cfg.DumpDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
		log.Info("Enabled dump_batch_proof", "dir", cfg.DumpDir)
	}

	return &BatchProver{cfg: cfg}, nil
}

// Prove call rust ffi to generate batch proof, if first failed, try again.
func (p *BatchProver) Prove(taskID string, chunkHashes []*types.ChunkHash, chunkProofs []*types.ChunkProof) (*message.BatchProof, error) {
	if p.cfg.ProofType != message.ProofTypeBatch {
		return nil, errors.New("Wrong proof type in batch-prover: %d", p.cfg.ProofType)
	}

	chunkHashesByt, err := json.Marshal(chunkHashes)
	if err != nil {
		return nil, err
	}
	chunkProofsByt, err := json.Marshal(chunkProofs)
	if err != nil {
		return nil, err
	}
	proofByt := p.prove(chunkHashesByt, chunkProofsByt)

	// dump proof
	err := p.dumpProof(taskID, proofByt)
	if err != nil {
		log.Error("Dump batch proof failed", "task-id", taskID, "error", err)
	}

	zkProof := &message.BatchProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

// Call cgo to generate batch proof.
func (p *BatchProver) prove(chunkHashesByt []byte, chunkProofsByt []byte) []byte {
	chunkHashesStr := C.CString(string(chunkHashesByt))
	chunkProofsStr := C.CString(string(chunkProofsByt))

	defer func() {
		C.free(unsafe.Pointer(chunkHashesStr))
		C.free(unsafe.Pointer(chunkProofsStr))
	}()

	log.Info("Start to create batch proof ...")
	cProof := C.gen_batch_proof(chunkHashesStr, chunkProofsStr)
	log.Info("Finish creating batch proof!")

	proof := C.GoString(cProof)
	return []byte(proof)
}

func (p *BatchProver) dumpProof(id string, proofByt []byte) error {
	if p.cfg.DumpDir == "" {
		return nil
	}
	path := filepath.Join(p.cfg.DumpDir, id)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	log.Info("Saving batch proof", "task-id", id)
	_, err = f.Write(proofByt)
	return err
}
