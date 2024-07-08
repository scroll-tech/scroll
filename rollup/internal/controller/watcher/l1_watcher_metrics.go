package watcher

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l1WatcherMetrics struct {
	l1WatcherFetchBlockHeaderTotal                prometheus.Counter
	l1WatcherFetchBlockHeaderProcessedBlockHeight prometheus.Gauge
}

var (
	initL1WatcherMetricOnce sync.Once
	l1WatcherMetric         *l1WatcherMetrics
)

func initL1WatcherMetrics(reg prometheus.Registerer) *l1WatcherMetrics {
	initL1WatcherMetricOnce.Do(func() {
		l1WatcherMetric = &l1WatcherMetrics{
			l1WatcherFetchBlockHeaderTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_l1_watcher_fetch_block_header_total",
				Help: "The total number of l1 watcher fetch block header total",
			}),
			l1WatcherFetchBlockHeaderProcessedBlockHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_l1_watcher_fetch_block_header_processed_block_height",
				Help: "The current processed block height of l1 watcher fetch block header",
			}),
		}
	})
	return l1WatcherMetric
}
