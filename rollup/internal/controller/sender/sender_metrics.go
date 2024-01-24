package sender

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type senderMetrics struct {
	senderCheckPendingTransactionTotal *prometheus.CounterVec
	sendTransactionTotal               *prometheus.CounterVec
	sendTransactionFailureGetFee       *prometheus.CounterVec
	sendTransactionFailureSendTx       *prometheus.CounterVec
	resubmitTransactionTotal           *prometheus.CounterVec
	resubmitTransactionFailedTotal     *prometheus.CounterVec
	currentGasFeeCap                   *prometheus.GaugeVec
	currentGasTipCap                   *prometheus.GaugeVec
	currentGasPrice                    *prometheus.GaugeVec
	currentGasLimit                    *prometheus.GaugeVec
}

var (
	initSenderMetricOnce sync.Once
	sm                   *senderMetrics
)

func initSenderMetrics(reg prometheus.Registerer) *senderMetrics {
	initSenderMetricOnce.Do(func() {
		sm = &senderMetrics{
			sendTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_total",
				Help: "The total number of sending transactions.",
			}, []string{"service", "name"}),
			sendTransactionFailureGetFee: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_get_fee_failure_total",
				Help: "The total number of sending transactions failure for getting fee.",
			}, []string{"service", "name"}),
			sendTransactionFailureSendTx: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_send_tx_failure_total",
				Help: "The total number of sending transactions failure for sending tx.",
			}, []string{"service", "name"}),
			resubmitTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_resubmit_send_transaction_total",
				Help: "The total number of resubmit transactions.",
			}, []string{"service", "name"}),
			resubmitTransactionFailedTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_resubmit_send_transaction_failed_total",
				Help: "The total number of failed resubmit transactions.",
			}, []string{"service", "name"}),
			currentGasFeeCap: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_fee_cap",
				Help: "The gas fee cap of current transaction.",
			}, []string{"service", "name"}),
			currentGasTipCap: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_tip_cap",
				Help: "The gas tip cap of current transaction.",
			}, []string{"service", "name"}),
			currentGasPrice: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_price_cap",
				Help: "The gas price of current transaction.",
			}, []string{"service", "name"}),
			currentGasLimit: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_limit",
				Help: "The gas limit of current transaction.",
			}, []string{"service", "name"}),
			senderCheckPendingTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_check_pending_transaction_total",
				Help: "The total number of check pending transaction.",
			}, []string{"service", "name"}),
		}
	})

	return sm
}
