package verifier

/*
#cgo LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl
#cgo gpu LDFLAGS: ${SRCDIR}/lib/libzkp.a -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart
#include <stdlib.h>
#include "./lib/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/spf13/viper"

	"scroll-tech/common/message"
)

// Verifier represents a rust ffi to a halo2 verifier.
type Verifier struct {
	v *viper.Viper
}

// NewVerifier Sets up a rust ffi to call verify.
func NewVerifier(v *viper.Viper) (*Verifier, error) {
	mockMode := v.GetBool("mock_mode")
	if mockMode {
		return &Verifier{v: v}, nil
	}
	paramsPathStr := C.CString(v.GetString("params_path"))
	aggVkPathStr := C.CString(v.GetString("agg_vk_path"))
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(aggVkPathStr))
	}()

	C.init_verifier(paramsPathStr, aggVkPathStr)

	return &Verifier{v: v}, nil
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	mockMode := v.v.GetBool("mock_mode")
	if mockMode {
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

	log.Info("Start to verify proof ...")
	verified := C.verify_agg_proof(aggProofStr)
	return verified != 0, nil
}
