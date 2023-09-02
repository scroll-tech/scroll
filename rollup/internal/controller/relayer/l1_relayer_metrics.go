package relayer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l1RelayerMetrics struct {
	rollupL1RelayedMsgsTotal               prometheus.Counter
	rollupL1RelayedMsgsFailureTotal        prometheus.Counter
	rollupL1RelayerGasPriceOraclerRunTotal prometheus.Counter
	rollupL1RelayerLastGasPrice            prometheus.Gauge
	rollupL1MsgsRelayedConfirmedTotal      prometheus.Counter
	rollupL1GasOraclerConfirmedTotal       prometheus.Counter
}

var (
	initL1RelayerMetricOnce sync.Once
	l1RelayerMetric         *l1RelayerMetrics
)

func initL1RelayerMetrics(reg prometheus.Registerer) *l1RelayerMetrics {
	initL1RelayerMetricOnce.Do(func() {
		l1RelayerMetric = &l1RelayerMetrics{
			rollupL1RelayedMsgsTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_msg_relayed_total",
				Help: "The total number of the l1 relayed message.",
			}),
			rollupL1RelayedMsgsFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_msg_relayed_failure_total",
				Help: "The total number of the l1 relayed failure message.",
			}),
			rollupL1MsgsRelayedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_relayed_confirmed_total",
				Help: "The total number of layer1 relayed confirmed",
			}),
			rollupL1RelayerGasPriceOraclerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_gas_price_oracler_total",
				Help: "The total number of layer1 gas price oracler run total",
			}),
			rollupL1RelayerLastGasPrice: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_layer1_gas_price_latest_gas_price",
				Help: "The latest gas price of rollup relayer l1",
			}),
			rollupL1GasOraclerConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_gas_oracler_confirmed_total",
				Help: "The total number of layer1 relayed confirmed",
			}),
		}
	})
	return l1RelayerMetric
}
