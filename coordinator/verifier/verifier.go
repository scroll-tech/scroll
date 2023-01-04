//go:build !mock_verifier

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

	"scroll-tech/common/message"
	"scroll-tech/common/viper"
)

// Verifier represents a rust ffi to a halo2 verifier.
type Verifier struct {
	vp *viper.Viper
}

// NewVerifier Sets up a rust ffi to call verify.
func NewVerifier(vp *viper.Viper) (*Verifier, error) {
	if vp.GetBool("mock_mode") {
		return &Verifier{vp: vp}, nil
	}
	paramsPathStr := C.CString(vp.GetString("params_path"))
	aggVkPathStr := C.CString(vp.GetString("agg_vk_path"))
	defer func() {
		C.free(unsafe.Pointer(paramsPathStr))
		C.free(unsafe.Pointer(aggVkPathStr))
	}()

	C.init_verifier(paramsPathStr, aggVkPathStr)

	return &Verifier{vp: vp}, nil
}

// VerifyProof Verify a ZkProof by marshaling it and sending it to the Halo2 Verifier.
func (v *Verifier) VerifyProof(proof *message.AggProof) (bool, error) {
	if v.vp.GetBool("mock_mode") {
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
