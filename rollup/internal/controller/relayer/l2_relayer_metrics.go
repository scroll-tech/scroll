package relayer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l2RelayerMetrics struct {
	rollupL2RelayerProcessBatchesPerTxCount                         prometheus.Gauge
	rollupL2RelayerProcessPendingBatchTotal                         prometheus.Counter
	rollupL2RelayerProcessPendingBatchSuccessTotal                  prometheus.Counter
	rollupL2RelayerProcessPendingBatchErrTooManyPendingBlobTxsTotal prometheus.Counter
	rollupL2BatchesCommittedConfirmedTotal                          prometheus.Counter
	rollupL2BatchesCommittedConfirmedFailedTotal                    prometheus.Counter
	rollupL2BatchesFinalizedConfirmedTotal                          prometheus.Counter
	rollupL2BatchesFinalizedConfirmedFailedTotal                    prometheus.Counter
	rollupL2ChainMonitorLatestFailedCall                            prometheus.Counter
	rollupL2ChainMonitorLatestFailedBatchStatus                     prometheus.Counter
	rollupL2RelayerProcessPendingBundlesTotal                       prometheus.Counter
	rollupL2RelayerProcessPendingBundlesFinalizedTotal              prometheus.Counter
	rollupL2RelayerProcessPendingBundlesFinalizedSuccessTotal       prometheus.Counter
	rollupL2BundlesFinalizedConfirmedTotal                          prometheus.Counter
	rollupL2BundlesFinalizedConfirmedFailedTotal                    prometheus.Counter

	rollupL2RelayerCommitBlockHeight prometheus.Gauge
	rollupL2RelayerCommitThroughput  prometheus.Counter
}

var (
	initL2RelayerMetricOnce sync.Once
	l2RelayerMetric         *l2RelayerMetrics
)

func initL2RelayerMetrics(reg prometheus.Registerer) *l2RelayerMetrics {
	initL2RelayerMetricOnce.Do(func() {
		l2RelayerMetric = &l2RelayerMetrics{
			rollupL2RelayerProcessBatchesPerTxCount: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_layer2_process_batches_per_tx_count",
				Help: "The number of batches processed per transaction",
			}),
			rollupL2RelayerProcessPendingBatchTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_pending_batch_total",
				Help: "The total number of layer2 process pending batch",
			}),
			rollupL2RelayerProcessPendingBatchSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_pending_batch_success_total",
				Help: "The total number of layer2 process pending success batch",
			}),
			rollupL2RelayerProcessPendingBatchErrTooManyPendingBlobTxsTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_pending_batch_err_too_many_pending_blob_txs_total",
				Help: "The total number of layer2 process pending batch failed on too many pending blob txs",
			}),
			rollupL2BatchesCommittedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_committed_batches_confirmed_total",
				Help: "The total number of layer2 process committed batches confirmed total",
			}),
			rollupL2BatchesCommittedConfirmedFailedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_committed_batches_confirmed_failed_total",
				Help: "The total number of layer2 process committed batches confirmed failed total",
			}),
			rollupL2BatchesFinalizedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_finalized_batches_confirmed_total",
				Help: "The total number of layer2 process finalized batches confirmed total",
			}),
			rollupL2BatchesFinalizedConfirmedFailedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_finalized_batches_confirmed_failed_total",
				Help: "The total number of layer2 process finalized batches confirmed failed total",
			}),
			rollupL2ChainMonitorLatestFailedCall: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_chain_monitor_latest_failed_batch_call",
				Help: "The total number of failed call chain_monitor api",
			}),
			rollupL2ChainMonitorLatestFailedBatchStatus: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_chain_monitor_latest_failed_batch_status",
				Help: "The total number of failed batch status get from chain_monitor",
			}),
			rollupL2RelayerProcessPendingBundlesTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_relayer_process_pending_bundles_total",
				Help: "Total number of times the layer2 relayer has processed pending bundles.",
			}),
			rollupL2RelayerProcessPendingBundlesFinalizedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_relayer_process_pending_bundles_finalized_total",
				Help: "Total number of times the layer2 relayer has finalized proven bundle processes.",
			}),
			rollupL2RelayerProcessPendingBundlesFinalizedSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_relayer_process_pending_bundles_finalized_success_total",
				Help: "Total number of times the layer2 relayer has successful finalized proven bundle processes.",
			}),
			rollupL2BundlesFinalizedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_bundles_finalized_confirmed_total",
				Help: "Total number of finalized bundles confirmed on layer2.",
			}),
			rollupL2BundlesFinalizedConfirmedFailedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_bundles_finalized_confirmed_failed_total",
				Help: "Total number of failed confirmations for finalized bundles on layer2.",
			}),
			rollupL2RelayerCommitBlockHeight: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_l2_relayer_commit_block_height",
				Help: "The latest block height committed by the L2 relayer",
			}),
			rollupL2RelayerCommitThroughput: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_l2_relayer_commit_throughput",
				Help: "The cumulative gas used in blocks committed by the L2 relayer",
			}),
		}
	})
	return l2RelayerMetric
}
