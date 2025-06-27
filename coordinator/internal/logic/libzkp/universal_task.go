//go:build !mock_verifier

package libzkp

/*
#include <stdlib.h>
#include "libzkp.h"
*/
import "C" //nolint:typecheck

// Initialize the handler for universal task
func InitL2geth(configJSON string) {
	cConfig := goToCString(configJSON)
	defer freeCString(cConfig)

	C.init_l2geth(cConfig)
}

func generateUniversalTask(taskType int, taskJSON, forkName string) (bool, string, string, []byte) {
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
