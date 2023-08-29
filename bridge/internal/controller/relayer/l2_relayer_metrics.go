package relayer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l2RelayerMetrics struct {
	bridgeL2RelayerProcessPendingBatchTotal                     prometheus.Counter
	bridgeL2RelayerProcessPendingBatchSuccessTotal              prometheus.Counter
	bridgeL2RelayerGasPriceOraclerRunTotal                      prometheus.Counter
	bridgeL2RelayerLastGasPrice                                 prometheus.Gauge
	bridgeL2RelayerProcessCommittedBatchesTotal                 prometheus.Counter
	bridgeL2RelayerProcessCommittedBatchesFinalizedTotal        prometheus.Counter
	bridgeL2RelayerProcessCommittedBatchesFinalizedSuccessTotal prometheus.Counter
	bridgeL2BatchesCommittedConfirmedTotal                      prometheus.Counter
	bridgeL2BatchesFinalizedConfirmedTotal                      prometheus.Counter
	bridgeL2BatchesGasOraclerConfirmedTotal                     prometheus.Counter
	bridgeL2ChainMonitorLatestFailedCall                        prometheus.Gauge
	bridgeL2ChainMonitorLatestFailedBatchStatus                 prometheus.Gauge
}

var (
	initL2RelayerMetricOnce sync.Once
	l2RelayerMetric         *l2RelayerMetrics
)

func initL2RelayerMetrics(reg prometheus.Registerer) *l2RelayerMetrics {
	initL2RelayerMetricOnce.Do(func() {
		l2RelayerMetric = &l2RelayerMetrics{
			bridgeL2RelayerProcessPendingBatchTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_pending_batch_total",
				Help: "The total number of layer2 process pending batch",
			}),
			bridgeL2RelayerProcessPendingBatchSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_pending_batch_success_total",
				Help: "The total number of layer2 process pending success batch",
			}),
			bridgeL2RelayerGasPriceOraclerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_gas_price_oracler_total",
				Help: "The total number of layer2 gas price oracler run total",
			}),
			bridgeL2RelayerLastGasPrice: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_layer2_gas_price_latest_gas_price",
				Help: "The latest gas price of bridge relayer l2",
			}),
			bridgeL2RelayerProcessCommittedBatchesTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_committed_batches_total",
				Help: "The total number of layer2 process committed batches run total",
			}),
			bridgeL2RelayerProcessCommittedBatchesFinalizedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_committed_batches_finalized_total",
				Help: "The total number of layer2 process committed batches finalized total",
			}),
			bridgeL2RelayerProcessCommittedBatchesFinalizedSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_committed_batches_finalized_success_total",
				Help: "The total number of layer2 process committed batches finalized success total",
			}),
			bridgeL2BatchesCommittedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_committed_batches_confirmed_total",
				Help: "The total number of layer2 process committed batches confirmed total",
			}),
			bridgeL2BatchesFinalizedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_finalized_batches_confirmed_total",
				Help: "The total number of layer2 process finalized batches confirmed total",
			}),
			bridgeL2BatchesGasOraclerConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer2_process_gras_oracler_confirmed_total",
				Help: "The total number of layer2 process finalized batches confirmed total",
			}),
			bridgeL2ChainMonitorLatestFailedCall: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_layer2_chain_monitor_latest_failed_batch_call",
				Help: "",
			}),
			bridgeL2ChainMonitorLatestFailedBatchStatus: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_layer2_chain_monitor_latest_failed_batch_status",
				Help: "The latest failed batch index before sending finalize batch tx",
			}),
		}
	})
	return l2RelayerMetric
}
