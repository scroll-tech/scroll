package utils

import (
	"github.com/scroll-tech/go-ethereum/core/types"
)

// ComputeTraceGasCost computes gascost based on ExecutionResults.StructLogs.GasCost
func ComputeTraceGasCost(trace *types.BlockTrace) uint64 {
	return trace.Header.GasUsed
}
