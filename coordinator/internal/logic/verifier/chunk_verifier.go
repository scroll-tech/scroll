//go:build !mock_verifier

package verifier

/*
#cgo LDFLAGS: -lzkp -lm -ldl -lzktrie -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: -lzkp -lm -ldl -lgmp -lstdc++ -lprocps -lzktrie -L/usr/local/cuda/lib64/ -lcudart -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#include <stdlib.h>
#include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"

	"scroll-tech/common/types/message"
)

// InvalidTestChunkProof invalid chunk proof used in tests
const InvalidTestChunkProof = "this is an invalid chunk proof"

// ChunkVerifier represents a rust ffi to a halo2 verifier.
type ChunkVerifier struct {
	cfg *config.ChunkVerifierConfig
}

// NewChunkVerifier Sets up a rust ffi to call verify.
func NewChunkVerifier(cfg *config.ChunkVerifierConfig) (*ChunkVerifier, error) {
	if cfg.MockMode {
		return &ChunkVerifier{cfg: cfg}, nil
	}
	paramsPathStr := C.CString(cfg.ParamsPath)
	assetsPathStr := C.CString(cfg.assetsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(assetsPathStr))
	}()

	C.init_chunk_verifier(paramsPathStr, assetsPathStr)

	return &ChunkVerifier{cfg: cfg}, nil
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *ChunkVerifier) VerifyProof(proof *message.ChunkProof) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Mock mode, verifier disabled")
		if string(proof.Proof) == InvalidTestChunkProof {
			return false, nil
		}
		return true, nil

	}
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	proofStr := C.CString(string(buf))
	defer func() {
		C.free(unsafe.Pointer(proofStr))
	}()

	log.Info("Start to verify chunk proof ...")
	verified := C.verify_chunk_proof(proofStr)
	return verified != 0, nil
}
