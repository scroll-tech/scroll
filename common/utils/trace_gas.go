package utils

import "github.com/scroll-tech/go-ethereum/core/types"

// ComputeTraceGasCost computes gascost based on ExecutionResults.StructLogs.GasCost
func ComputeTraceGasCost(trace *types.BlockTrace) uint64 {
	var gas_cost uint64 = 0
	finishCh := make(chan uint64)
	for _, v := range trace.ExecutionResults {
		go func(v *types.ExecutionResult) {
			var sum uint64 = 0
			for _, structV := range v.StructLogs {
				sum += structV.GasCost
			}
			finishCh <- sum
		}(v)
	}
	for range trace.ExecutionResults {
		res := <-finishCh
		gas_cost += res
	}

	return gas_cost
}
