//go:build !mock_prover

package prover

/*
#cgo LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl -lzktrie -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl -lgmp -lstdc++ -lprocps -lzktrie -L/usr/local/cuda/lib64/ -L${SRCDIR}/lib/ -lcudart -Wl,-rpath=${SRCDIR}/lib
#include <stdlib.h>
#include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/types/message"

	"scroll-tech/roller/config"
)

// Prover sends block-traces to rust-prover through ffi and get back the zk-proof.
type Prover struct {
	cfg    *config.ProverConfig
	ethCli *ethclient.Client
}

// NewProver inits a Prover object.
func NewProver(cfg *config.ProverConfig) (*Prover, error) {
	ethCli, err := ethclient.Dial(cfg.EthEndpoint)
	if err != nil {
		return nil, err
	}

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

	return &Prover{cfg: cfg, ethCli: ethCli}, nil
}

// Prove call rust ffi to generate proof, if first failed, try again.
func (p *Prover) Prove(task *message.TaskMsg) (*message.AggProof, error) {
	var proofByt []byte
	if p.cfg.ProveType == message.BasicProve {
		traces, err := p.getTracesByHashes(task.BlockHashes)
		if err != nil {
			return nil, err
		}
		tracesByt, err := json.Marshal(traces)
		if err != nil {
			return nil, err
		}
		proofByt = p.prove(tracesByt)
	} else if p.cfg.ProveType == message.AggregatorProve {
		// TODO: aggregator prove
	}

	// dump proof
	err := p.dumpProof(task.ID, proofByt)
	if err != nil {
		log.Error("Dump proof failed", "task-id", task.ID, "error", err)
	}

	zkProof := &message.AggProof{}
	return zkProof, json.Unmarshal(proofByt, zkProof)
}

// Call cgo to generate proof.
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

func (p *Prover) getTracesByHashes(blockHashes []common.Hash) ([]*types.BlockTrace, error) {
	var traces []*types.BlockTrace
	for _, blockHash := range blockHashes {
		trace, err := p.ethCli.GetBlockTraceByHash(context.Background(), blockHash)
		if err != nil {
			return nil, err
		}
		traces = append(traces, trace)
	}
	return traces, nil
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
