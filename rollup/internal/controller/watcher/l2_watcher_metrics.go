package watcher

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l2WatcherMetrics struct {
	fetchRunningMissingBlocksTotal    prometheus.Counter
	fetchRunningMissingBlocksHeight   prometheus.Gauge
	rollupL2BlocksFetchedGap          prometheus.Gauge
	rollupL2BlockL1CommitCalldataSize prometheus.Gauge
	fetchNilRowConsumptionBlockTotal  prometheus.Counter

	rollupL2WatcherSyncThroughput prometheus.Counter
}

var (
	initL2WatcherMetricOnce sync.Once
	l2WatcherMetric         *l2WatcherMetrics
)

func initL2WatcherMetrics(reg prometheus.Registerer) *l2WatcherMetrics {
	initL2WatcherMetricOnce.Do(func() {
		l2WatcherMetric = &l2WatcherMetrics{
			fetchRunningMissingBlocksTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_l2_watcher_fetch_running_missing_blocks_total",
				Help: "The total number of l2 watcher fetch running missing blocks",
			}),
			fetchRunningMissingBlocksHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_l2_watcher_fetch_running_missing_blocks_height",
				Help: "The total number of l2 watcher fetch running missing blocks height",
			}),
			rollupL2BlocksFetchedGap: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_l2_watcher_blocks_fetched_gap",
				Help: "The gap of l2 fetch",
			}),
			rollupL2BlockL1CommitCalldataSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_l2_block_l1_commit_calldata_size",
				Help: "The l1 commitBatch calldata size of the l2 block",
			}),
			fetchNilRowConsumptionBlockTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_l2_watcher_fetch_nil_row_consumption_block_total",
				Help: "The total number of occurrences where a fetched block has nil RowConsumption",
			}),
			rollupL2WatcherSyncThroughput: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_l2_watcher_sync_throughput",
				Help: "The cumulative gas used in blocks that L2 watcher sync",
			}),
		}
	})
	return l2WatcherMetric
}
