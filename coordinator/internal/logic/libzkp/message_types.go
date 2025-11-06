package libzkp

import (
	"fmt"

	"scroll-tech/common/types/message"
)

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
