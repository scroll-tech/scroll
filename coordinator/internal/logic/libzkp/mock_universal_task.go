//go:build mock_verifier

package libzkp

import (
	"encoding/json"
	"fmt"

	"scroll-tech/common/types/message"

	"github.com/scroll-tech/go-ethereum/common"
)

func InitL2geth(configJSON string) {
}

func generateUniversalTask(taskType int, taskJSON, forkName string, expectedVk []byte) (bool, string, string, []byte) {

	fmt.Printf("call mocked generate universal task %d, taskJson %s\n", taskType, taskJSON)
	var metadata interface{}
	switch taskType {
	case TaskTypeChunk:
		metadata = struct {
			ChunkInfo *message.ChunkInfo `json:"chunk_info"`
		}{ChunkInfo: &message.ChunkInfo{}}
	case TaskTypeBatch:
		metadata = struct {
			BatchInfo *message.OpenVMBatchInfo `json:"batch_info"`
			BatchHash common.Hash              `json:"batch_hash"`
		}{BatchInfo: &message.OpenVMBatchInfo{}}
	case TaskTypeBundle:
		metadata = struct {
			BundleInfo   *message.OpenVMBundleInfo `json:"bundle_info"`
			BundlePIHash common.Hash               `json:"bundle_pi_hash"`
		}{BundleInfo: &message.OpenVMBundleInfo{}}
	}

	encodeData, err := json.Marshal(metadata)
	if err != nil {
		fmt.Println("mock encoding json fail:", err)
		return false, "", "", nil
	}

	return true, "UniversalTask data is not parsed", string(encodeData), []byte{0}
}
