//go:build !mock_verifier

package old_verifier

/*
#cgo LDFLAGS: -loldzkp -lm -ldl -loldzktrie -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: -loldzkp -lm -ldl -lgmp -lstdc++ -lprocps -loldzktrie -L/usr/local/cuda/lib64/ -lcudart -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#include <stdlib.h>
#include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"scroll-tech/common/types/message"
	"scroll-tech/coordinator/internal/logic/verifier"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/internal/config"
)

// OldVerifier represents a rust ffi to a halo2 verifier.
type OldVerifier struct {
	cfg *config.VerifierConfig
}

// NewOldVerifier Sets up a rust ffi to call verify.
func NewOldVerifier(cfg *config.VerifierConfig) (*OldVerifier, error) {
	if cfg.MockMode {
		return &OldVerifier{cfg: cfg}, nil
	}
	paramsPathStr := C.CString(cfg.ParamsPath)
	assetsPathStr := C.CString(cfg.AssetsPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(assetsPathStr))
	}()

	log.Info("Init old verifier!")

	C.init_batch_verifier(paramsPathStr, assetsPathStr)
	C.init_chunk_verifier(paramsPathStr, assetsPathStr)

	return &OldVerifier{cfg: cfg}, nil
}

// VerifyBatchProof Verify a ZkProof by marshaling it and sending it to the Halo2 OldVerifier.
func (v *OldVerifier) VerifyBatchProof(proof *message.BatchProof) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Mock mode, batch verifier disabled")
		if string(proof.Proof) == verifier.InvalidTestProof {
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

	log.Info("Start to verify batch proof ...")
	verified := C.verify_batch_proof(proofStr)
	return verified != 0, nil
}

// VerifyChunkProof Verify a ZkProof by marshaling it and sending it to the Halo2 OldVerifier.
func (v *OldVerifier) VerifyChunkProof(proof *message.ChunkProof) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Mock mode, verifier disabled")
		if string(proof.Proof) == verifier.InvalidTestProof {
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
