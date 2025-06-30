package libzkp

/*
#cgo LDFLAGS: -lzkp -lm -ldl -L${SRCDIR}/lib -Wl,-rpath=${SRCDIR}/lib
#cgo gpu LDFLAGS: -lzkp -lm -ldl -lgmp -lstdc++ -lprocps -L/usr/local/cuda/lib64/ -lcudart -L${SRCDIR}/lib/ -Wl,-rpath=${SRCDIR}/lib
#include <stdlib.h>
#include "libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"scroll-tech/common/types/message"
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

// Verify a chunk proof
func VerifyChunkProof(proofData, forkName string) bool {
	cProof := goToCString(proofData)
	cForkName := goToCString(strings.ToLower(forkName))
	defer freeCString(cProof)
	defer freeCString(cForkName)

	result := C.verify_chunk_proof(cProof, cForkName)
	return result != 0
}

// Verify a batch proof
func VerifyBatchProof(proofData, forkName string) bool {
	cProof := goToCString(proofData)
	cForkName := goToCString(strings.ToLower(forkName))
	defer freeCString(cProof)
	defer freeCString(cForkName)

	result := C.verify_batch_proof(cProof, cForkName)
	return result != 0
}

// Verify a bundle proof
func VerifyBundleProof(proofData, forkName string) bool {
	cProof := goToCString(proofData)
	cForkName := goToCString(strings.ToLower(forkName))
	defer freeCString(cProof)
	defer freeCString(cForkName)

	result := C.verify_bundle_proof(cProof, cForkName)
	return result != 0
}

// TaskType enum values matching the Rust enum
const (
	TaskTypeChunk  = 0
	TaskTypeBatch  = 1
	TaskTypeBundle = 2
)

func fromMessageTaskType(taskType int) int {
	switch message.ProofType(taskType) {
	case message.ProofTypeChunk:
		return TaskTypeChunk
	case message.ProofTypeBatch:
		return TaskTypeBatch
	case message.ProofTypeBundle:
		return TaskTypeBundle
	default:
		panic(fmt.Sprintf("unsupported proof type: %d", taskType))
	}
}

// Generate a universal task
func GenerateUniversalTask(taskType int, taskJSON, forkName string, expectedVk []byte) (bool, string, string, []byte) {
	return generateUniversalTask(fromMessageTaskType(taskType), taskJSON, strings.ToLower(forkName), expectedVk)
}

// Generate wrapped proof
func GenerateWrappedProof(proofJSON, metadata string, vkData []byte) string {
	cProofJSON := goToCString(proofJSON)
	cMetadata := goToCString(metadata)
	defer freeCString(cProofJSON)
	defer freeCString(cMetadata)

	// Create a C array from Go slice
	var cVkData *C.char
	if len(vkData) > 0 {
		cVkData = (*C.char)(unsafe.Pointer(&vkData[0]))
	}

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
	cForkName := goToCString(strings.ToLower(forkName))
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
