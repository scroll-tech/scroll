//go:build !mock_prover

package core

/*
#cgo LDFLAGS: -lpthread -lzkp -lm -ldl -lzktrie -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: -lpthread -lzkp -lm -ldl -lgmp -lstdc++ -lprocps -lzktrie -L/usr/local/cuda/lib64/ -lcudart -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
# include <stdio.h>
# include <stdlib.h>
# include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"unsafe"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"

	jsoniter "github.com/json-iterator/go"

	"scroll-tech/common/types/message"

	"scroll-tech/prover/config"
)

type ProverCore struct {
	cfg *config.ProverCoreConfig
	VK  string
}

func NewProverCore(cfg *config.ProverCoreConfig) (*ProverCore, error) {
	paramsPathStr := C.CString(cfg.ParamsPath)
	assetsPathStr := C.CString(cfg.AssetsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(assetsPathStr))
	}()

	var vk string
	C.init_chunk_prover(paramsPathStr, assetsPathStr)

	return &ProverCore{cfg: cfg, VK: vk}, nil
}

func (p *ProverCore) ProveChunk(taskID string, traces []*types.BlockTrace) (*message.ChunkProof, error) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	tracesByt, _ := json.Marshal(traces)

	p.proveChunk(tracesByt)

	return nil, nil
}

type MyString struct {
       Str *C.char
       Len int
}

func (p *ProverCore) proveChunk(tracesByt []byte) ([]byte, error) {
	log.Info("Start to create chunk proof ...")

	len := len(tracesByt)
	ptr := C.CBytes(tracesByt)

	cProof := C.gen_chunk_proof((*C.char)(ptr), C.uint(len))
	C.free(ptr)
	defer C.free_c_chars(cProof)
	log.Info("Finish creating chunk proof!")

	return make([]byte, 1), nil
}
