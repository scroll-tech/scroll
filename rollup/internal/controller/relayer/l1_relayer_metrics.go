package relayer

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type l1RelayerMetrics struct {
	rollupL1RelayerGasPriceOraclerRunTotal      prometheus.Counter
	rollupL1RelayerLastGasPrice                 prometheus.Gauge
	rollupL1UpdateGasOracleConfirmedTotal       prometheus.Counter
	rollupL1UpdateGasOracleConfirmedFailedTotal prometheus.Counter
}

var (
	initL1RelayerMetricOnce sync.Once
	l1RelayerMetric         *l1RelayerMetrics
)

func initL1RelayerMetrics(reg prometheus.Registerer) *l1RelayerMetrics {
	initL1RelayerMetricOnce.Do(func() {
		l1RelayerMetric = &l1RelayerMetrics{
			rollupL1RelayerGasPriceOraclerRunTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_gas_price_oracler_total",
				Help: "The total number of layer1 gas price oracler run total",
			}),
			rollupL1RelayerLastGasPrice: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
				Name: "rollup_layer1_gas_price_latest_gas_price",
				Help: "The latest gas price of rollup relayer l1",
			}),
			rollupL1UpdateGasOracleConfirmedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_update_gas_oracle_confirmed_total",
				Help: "The total number of updating layer1 gas oracle confirmed",
			}),
			rollupL1UpdateGasOracleConfirmedFailedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_layer1_update_gas_oracle_confirmed_failed_total",
				Help: "The total number of updating layer1 gas oracle confirmed failed",
			}),
		}
	})
	return l1RelayerMetric
}
