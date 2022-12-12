package utils

import (
	"sync/atomic"

	"github.com/scroll-tech/go-ethereum/core/types"
	"golang.org/x/sync/errgroup"
)

// ComputeTraceGasCost computes gascost based on ExecutionResults.StructLogs.GasCost
func ComputeTraceGasCost(trace *types.BlockTrace) uint64 {
	var (
		gasCost uint64
		eg      errgroup.Group
	)
	for idx := range trace.ExecutionResults {
		i := idx
		eg.Go(func() error {
			var sum uint64
			for _, log := range trace.ExecutionResults[i].StructLogs {
				sum += log.GasCost
			}
			atomic.AddUint64(&gasCost, sum)
			return nil
		})
	}
	_ = eg.Wait()

	return gasCost
}
