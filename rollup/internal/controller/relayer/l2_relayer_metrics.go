package relayer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l2RelayerMetrics struct {
	rollupL2RelayerProcessPendingBatchTotal                     prometheus.Counter
	rollupL2RelayerProcessPendingBatchSuccessTotal              prometheus.Counter
	rollupL2RelayerGasPriceOraclerRunTotal                      prometheus.Counter
	rollupL2RelayerLastGasPrice                                 prometheus.Gauge
	rollupL2RelayerProcessCommittedBatchesTotal                 prometheus.Counter
	rollupL2RelayerProcessCommittedBatchesFinalizedTotal        prometheus.Counter
	rollupL2RelayerProcessCommittedBatchesFinalizedSuccessTotal prometheus.Counter
	rollupL2BatchesCommittedConfirmedTotal                      prometheus.Counter
	rollupL2BatchesFinalizedConfirmedTotal                      prometheus.Counter
	rollupL2BatchesGasOraclerConfirmedTotal                     prometheus.Counter
	rollupL2ChainMonitorLatestFailedCall                        prometheus.Counter
	rollupL2ChainMonitorLatestFailedBatchStatus                 prometheus.Counter
}

var (
	initL2RelayerMetricOnce sync.Once
	l2RelayerMetric         *l2RelayerMetrics
)

func initL2RelayerMetrics(reg prometheus.Registerer) *l2RelayerMetrics {
	initL2RelayerMetricOnce.Do(func() {
		l2RelayerMetric = &l2RelayerMetrics{
			rollupL2RelayerProcessPendingBatchTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_pending_batch_total",
				Help: "The total number of layer2 process pending batch",
			}),
			rollupL2RelayerProcessPendingBatchSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_pending_batch_success_total",
				Help: "The total number of layer2 process pending success batch",
			}),
			rollupL2RelayerGasPriceOraclerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_gas_price_oracler_total",
				Help: "The total number of layer2 gas price oracler run total",
			}),
			rollupL2RelayerLastGasPrice: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_layer2_gas_price_latest_gas_price",
				Help: "The latest gas price of rollup relayer l2",
			}),
			rollupL2RelayerProcessCommittedBatchesTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_committed_batches_total",
				Help: "The total number of layer2 process committed batches run total",
			}),
			rollupL2RelayerProcessCommittedBatchesFinalizedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_committed_batches_finalized_total",
				Help: "The total number of layer2 process committed batches finalized total",
			}),
			rollupL2RelayerProcessCommittedBatchesFinalizedSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_committed_batches_finalized_success_total",
				Help: "The total number of layer2 process committed batches finalized success total",
			}),
			rollupL2BatchesCommittedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_committed_batches_confirmed_total",
				Help: "The total number of layer2 process committed batches confirmed total",
			}),
			rollupL2BatchesFinalizedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_finalized_batches_confirmed_total",
				Help: "The total number of layer2 process finalized batches confirmed total",
			}),
			rollupL2BatchesGasOraclerConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_process_gras_oracler_confirmed_total",
				Help: "The total number of layer2 process finalized batches confirmed total",
			}),
			rollupL2ChainMonitorLatestFailedCall: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_chain_monitor_latest_failed_batch_call",
				Help: "The total number of failed call chain_monitor api",
			}),
			rollupL2ChainMonitorLatestFailedBatchStatus: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer2_chain_monitor_latest_failed_batch_status",
				Help: "The total number of failed batch status get from chain_monitor",
			}),
		}
	})
	return l2RelayerMetric
}
