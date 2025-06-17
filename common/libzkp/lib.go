package libzkp

/*
#cgo LDFLAGS: -lzkp -lm -ldl -L${SRCDIR}/lib -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: -lzkp -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#include <stdlib.h>
#include "interface/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"fmt"
	"os"
	"unsafe"
)

// Helper function to convert Go string to C string and handle cleanup
func goToCString(s string) *C.char {
	return C.CString(s)
}

// Helper function to free C string
func freeCString(s *C.char) {
	C.free(unsafe.Pointer(s))
}

// Initialize the verifier
func InitVerifier(configJSON string) {
	cConfig := goToCString(configJSON)
	defer freeCString(cConfig)

	C.init_verifier(cConfig)
}

// Initialize the verifier
func InitL2geth(configJSON string) {
	cConfig := goToCString(configJSON)
	defer freeCString(cConfig)

	C.init_l2geth(cConfig)
}

// Verify a chunk proof
func VerifyChunkProof(proofData, forkName string) bool {
	cProof := goToCString(proofData)
	cForkName := goToCString(forkName)
	defer freeCString(cProof)
	defer freeCString(cForkName)

	result := C.verify_chunk_proof(cProof, cForkName)
	return result != 0
}

// Verify a batch proof
func VerifyBatchProof(proofData, forkName string) bool {
	cProof := goToCString(proofData)
	cForkName := goToCString(forkName)
	defer freeCString(cProof)
	defer freeCString(cForkName)

	result := C.verify_batch_proof(cProof, cForkName)
	return result != 0
}

// Verify a bundle proof
func VerifyBundleProof(proofData, forkName string) bool {
	cProof := goToCString(proofData)
	cForkName := goToCString(forkName)
	defer freeCString(cProof)
	defer freeCString(cForkName)

	result := C.verify_bundle_proof(cProof, cForkName)
	return result != 0
}

// Generate a universal task
func GenerateUniversalTask(taskType int, taskJSON, forkName string) (bool, string, string, []byte) {
	cTask := goToCString(taskJSON)
	cForkName := goToCString(forkName)
	defer freeCString(cTask)
	defer freeCString(cForkName)

	result := C.gen_universal_task(C.int(taskType), cTask, cForkName)
	defer C.release_task_result(result)

	// Check if the operation was successful
	if result.ok == 0 {
		return false, "", "", nil
	}

	// Convert C strings to Go strings
	universalTask := C.GoString(result.universal_task)
	metadata := C.GoString(result.metadata)

	// Convert C array to Go slice
	piHash := make([]byte, 32)
	for i := 0; i < 32; i++ {
		piHash[i] = byte(result.expected_pi_hash[i])
	}

	return true, universalTask, metadata, piHash
}

// Generate wrapped proof
func GenerateWrappedProof(proofJSON, metadata string, vkData []byte) string {
	cProofJSON := goToCString(proofJSON)
	cMetadata := goToCString(metadata)
	defer freeCString(cProofJSON)
	defer freeCString(cMetadata)

	// Create a C array from Go slice
	cVkData := (*C.char)(unsafe.Pointer(&vkData[0]))

	resultPtr := C.gen_wrapped_proof(cProofJSON, cMetadata, cVkData, C.size_t(len(vkData)))
	if resultPtr == nil {
		return ""
	}

	// Convert result to Go string and free C memory
	result := C.GoString(resultPtr)
	C.release_string(resultPtr)

	return result
}

// Dumps a verification key to a file
func DumpVk(forkName, filePath string) error {
	cForkName := goToCString(forkName)
	cFilePath := goToCString(filePath)
	defer freeCString(cForkName)
	defer freeCString(cFilePath)

	// Call the C function to dump the verification key
	C.dump_vk(cForkName, cFilePath)

	// Check if the file was created successfully
	// Note: The C function doesn't return an error code, so we check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("failed to dump verification key: file %s was not created", filePath)
	}

	return nil
}
