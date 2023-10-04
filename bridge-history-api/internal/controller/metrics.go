package controller

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type cacheMetrics struct {
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec
}

var (
	initMetricsOnce sync.Once
	cm              *cacheMetrics
)

func initCacheMetrics() *cacheMetrics {
	initMetricsOnce.Do(func() {
		cm = &cacheMetrics{
			cacheHits: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "bridge_history_api_cache_hits_total",
					Help: "The total number of cache hits",
				},
				[]string{"api"},
			),
			cacheMisses: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "bridge_history_api_cache_misses_total",
					Help: "The total number of cache misses",
				},
				[]string{"api"},
			),
		}
	})
	return cm
}
