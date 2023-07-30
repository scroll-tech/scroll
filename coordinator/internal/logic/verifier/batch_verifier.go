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

// InvalidTestBatchProof invalid batch proof used in tests
const InvalidTestBatchProof = "this is an invalid batch proof"

// BatchVerifier represents a rust ffi to a halo2 verifier.
type BatchVerifier struct {
	cfg *config.BatchVerifierConfig
}

// NewBatchVerifier Sets up a rust ffi to call verify.
func NewBatchVerifier(cfg *config.BatchVerifierConfig) (*BatchVerifier, error) {
	if cfg.MockMode {
		return &BatchVerifier{cfg: cfg}, nil
	}
	paramsPathStr := C.CString(cfg.ParamsPath)
	vkPathStr := C.CString(cfg.vkPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(vkPathStr))
	}()

	C.init_batch_verifier(paramsPathStr, vkPathStr)

	return &BatchVerifier{cfg: cfg}, nil
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *BatchVerifier) VerifyProof(proof *message.BatchProof) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Mock mode, batch verifier disabled")
		if string(proof.Proof) == InvalidTestBatchProof {
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
