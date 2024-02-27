package watcher

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// l2WatcherMetrics contient les métriques de surveillance pour le watcher de la couche 2 (L2)
type l2WatcherMetrics struct {
    fetchRunningMissingBlocksTotal    prometheus.Counter
    fetchRunningMissingBlocksHeight   prometheus.Gauge
    fetchContractEventTotal           prometheus.Counter
    fetchContractEventHeight          prometheus.Gauge
    rollupL2MsgsRelayedEventsTotal    prometheus.Counter
    rollupL2BlocksFetchedGap          prometheus.Gauge
    rollupL2BlockL1CommitCalldataSize prometheus.Gauge
}

// initL2WatcherMetrics initialise les métriques de surveillance pour le watcher de la couche 2 (L2)
func initL2WatcherMetrics(reg prometheus.Registerer) *l2WatcherMetrics {
    metrics := &l2WatcherMetrics{
        fetchRunningMissingBlocksTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
            Name: "rollup_l2_watcher_fetch_running_missing_blocks_total",
            Help: "The total number of L2 watcher fetch running missing blocks",
        }),
        fetchRunningMissingBlocksHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
            Name: "rollup_l2_watcher_fetch_running_missing_blocks_height",
            Help: "The height of L2 watcher fetch running missing blocks",
        }),
        fetchContractEventTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
            Name: "rollup_l2_watcher_fetch_contract_events_total",
            Help: "The total number of L2 watcher fetch contract events",
        }),
        fetchContractEventHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
            Name: "rollup_l2_watcher_fetch_contract_height",
            Help: "The height of L2 watcher fetch contract",
        }),
        rollupL2MsgsRelayedEventsTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
            Name: "rollup_l2_watcher_msg_relayed_events_total",
            Help: "The total number of L2 watcher relayed events",
        }),
        rollupL2BlocksFetchedGap: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
            Name: "rollup_l2_watcher_blocks_fetched_gap",
            Help: "The gap of L2 fetch",
        }),
        rollupL2BlockL1CommitCalldataSize: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
            Name: "rollup_l2_block_l1_commit_calldata_size",
            Help: "The size of L1 commitBatch calldata in the L2 block",
        }),
    }
    return metrics
}
