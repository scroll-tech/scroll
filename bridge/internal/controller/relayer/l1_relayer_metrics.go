package relayer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l1RelayerMetrics struct {
	bridgeL1RelayedMsgsTotal               prometheus.Counter
	bridgeL1RelayedMsgsFailureTotal        prometheus.Counter
	bridgeL1RelayerGasPriceOraclerRunTotal prometheus.Counter
	bridgeL1RelayerLastGasPrice            prometheus.Gauge
	bridgeL1MsgsRelayedConfirmedTotal      prometheus.Counter
	bridgeL1GasOraclerConfirmedTotal       prometheus.Counter
}

var (
	initL1RelayerMetricOnce sync.Once
	l1RelayerMetric         *l1RelayerMetrics
)

func initL1RelayerMetrics(reg prometheus.Registerer) *l1RelayerMetrics {
	initL1RelayerMetricOnce.Do(func() {
		l1RelayerMetric = &l1RelayerMetrics{
			bridgeL1RelayedMsgsTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer1_msg_relayed_total",
				Help: "The total number of the l1 relayed message.",
			}),
			bridgeL1RelayedMsgsFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer1_msg_relayed_failure_total",
				Help: "The total number of the l1 relayed failure message.",
			}),
			bridgeL1MsgsRelayedConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer1_relayed_confirmed_total",
				Help: "The total number of layer1 relayed confirmed",
			}),
			bridgeL1RelayerGasPriceOraclerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer1_gas_price_oracler_total",
				Help: "The total number of layer1 gas price oracler run total",
			}),
			bridgeL1RelayerLastGasPrice: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "bridge_layer1_gas_price_latest_gas_price",
				Help: "The latest gas price of bridge relayer l1",
			}),
			bridgeL1GasOraclerConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "bridge_layer1_gas_oracler_confirmed_total",
				Help: "The total number of layer1 relayed confirmed",
			}),
		}
	})
	return l1RelayerMetric
}
