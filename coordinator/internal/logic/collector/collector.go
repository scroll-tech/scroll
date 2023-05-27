package collector

import "context"

const (
	AggTaskCollectorName    = "agg_task_collector"
	BlockBatchCollectorName = "block_batch_collector"
)

// Collector the interface of a collector who send data to prover
type Collector interface {
	Name() string
	Collect(ctx context.Context) error
	Recover()
}
