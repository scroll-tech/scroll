package watcher

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l1WatcherMetrics struct {
	l1WatcherFetchBlockHeaderTotal                  prometheus.Counter
	l1WatcherFetchBlockHeaderProcessedBlockHeight   prometheus.Gauge
	l1WatcherFetchContractEventTotal                prometheus.Counter
	l1WatcherFetchContractEventSuccessTotal         prometheus.Counter
	l1WatcherFetchContractEventProcessedBlockHeight prometheus.Gauge
	l1WatcherFetchContractEventRollupEventsTotal    prometheus.Counter
}

var (
	initL1WatcherMetricOnce sync.Once
	l1WatcherMetric         *l1WatcherMetrics
)

func initL1WatcherMetrics(reg prometheus.Registerer) *l1WatcherMetrics {
	initL1WatcherMetricOnce.Do(func() {
		l1WatcherMetric = &l1WatcherMetrics{
			l1WatcherFetchBlockHeaderTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_l1_watcher_fetch_block_header_total",
				Help: "The total number of l1 watcher fetch block header total",
			}),
			l1WatcherFetchBlockHeaderProcessedBlockHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_l1_watcher_fetch_block_header_processed_block_height",
				Help: "The current processed block height of l1 watcher fetch block header",
			}),
			l1WatcherFetchContractEventTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_l1_watcher_fetch_block_contract_event_total",
				Help: "The total number of l1 watcher fetch contract event total",
			}),
			l1WatcherFetchContractEventSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_l1_watcher_fetch_block_contract_event_success_total",
				Help: "The total number of l1 watcher fetch contract event success total",
			}),
			l1WatcherFetchContractEventProcessedBlockHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_l1_watcher_fetch_block_contract_event_processed_block_height",
				Help: "The current processed block height of l1 watcher fetch contract event",
			}),
			l1WatcherFetchContractEventRollupEventsTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_l1_watcher_fetch_block_contract_event_rollup_event_total",
				Help: "The current processed block height of l1 watcher fetch contract rollup event",
			}),
		}
	})
	return l1WatcherMetric
}
