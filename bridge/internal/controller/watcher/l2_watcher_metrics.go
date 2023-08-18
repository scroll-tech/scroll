package watcher

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l2WatcherMetrics struct {
	fetchRunningMissingBlocksTotal  prometheus.Counter
	fetchRunningMissingBlocksHeight prometheus.Gauge
	fetchContractEventTotal         prometheus.Counter
	bridgeL2BlocksFetchedGap        prometheus.Gauge
}

var (
	initL2WatcherMetricOnce sync.Once
	l2WatcherMetric         *l2WatcherMetrics
)

func initL2WatcherMetrics(reg prometheus.Registerer) *l2WatcherMetrics {
	initL2WatcherMetricOnce.Do(func() {
		l2WatcherMetric = &l2WatcherMetrics{
			fetchRunningMissingBlocksTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_l2_watcher_fetch_running_missing_blocks_total",
				Help: "The total number of l2 watcher fetch running missing blocks",
			}),
			fetchRunningMissingBlocksHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_l2_watcher_fetch_running_missing_blocks_height",
				Help: "The total number of l2 watcher fetch running missing blocks height",
			}),
			fetchContractEventTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_l2_watcher_fetch_contract_events_total",
				Help: "The total number of l2 watcher fetch contract events",
			}),
			bridgeL2BlocksFetchedGap: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_l2_watcher_blocks_fetched_gap",
				Help: "The gap of l2 fetch",
			}),
		}
	})
	return l2WatcherMetric
}
