package verifier

/*
#cgo LDFLAGS: ${SRCDIR}/interface/libzkp.a -lm -ldl
#cgo gpu LDFLAGS: ${SRCDIR}/interface/libzkp.a -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart
#include <stdlib.h>
#include "./interface/zkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/coordinator/config"

	"scroll-tech/common/message"
)

// Verifier represents a rust ffi to a halo2 verifier.
type Verifier struct {
	cfg *config.VerifierConfig
}

// NewVerifier Sets up a rust ffi to call verify.
func NewVerifier(cfg *config.VerifierConfig) (*Verifier, error) {
	if cfg.MockMode {
		return &Verifier{cfg: cfg}, nil
	}
	paramsPathStr := C.CString(cfg.ParamsPath)
	aggVkPathStr := C.CString(cfg.AggVkPath)
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(aggVkPathStr))
	}()

	C.init_verifier(paramsPathStr, aggVkPathStr)

	return &Verifier{cfg: cfg}, nil
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	if v.cfg.MockMode {
		log.Info("Verifier disabled, VerifyProof skipped")
		return true, nil

	}
	buf, err := json.Marshal(proof)
	if err != nil {
		return false, err
	}

	aggProofStr := C.CString(string(buf))
	defer func() {
		C.free(unsafe.Pointer(aggProofStr))
	}()
	verified := C.verify_agg_proof(aggProofStr)
	return verified != 0, nil
}
