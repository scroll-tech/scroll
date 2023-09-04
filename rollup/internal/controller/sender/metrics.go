package sender

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type senderMetrics struct {
	senderCheckBalancerTotal                *prometheus.CounterVec
	senderCheckPendingTransactionTotal      *prometheus.CounterVec
	sendTransactionTotal                    *prometheus.CounterVec
	sendTransactionFailureFullTx            *prometheus.GaugeVec
	sendTransactionFailureRepeatTransaction *prometheus.CounterVec
	sendTransactionFailureGetFee            *prometheus.CounterVec
	sendTransactionFailureSendTx            *prometheus.CounterVec
	resubmitTransactionTotal                *prometheus.CounterVec
	currentPendingTxsNum                    *prometheus.GaugeVec
	currentGasFeeCap                        *prometheus.GaugeVec
	currentGasTipCap                        *prometheus.GaugeVec
	currentGasPrice                         *prometheus.GaugeVec
	currentGasLimit                         *prometheus.GaugeVec
	currentNonce                            *prometheus.GaugeVec
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
				Help: "The total number of sending transaction.",
			}, []string{"service", "name"}),
			sendTransactionFailureFullTx: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_send_transaction_full_tx_failure_total",
				Help: "The total number of sending transaction failure for full size tx.",
			}, []string{"service", "name"}),
			sendTransactionFailureRepeatTransaction: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_repeat_transaction_failure_total",
				Help: "The total number of sending transaction failure for repeat transaction.",
			}, []string{"service", "name"}),
			sendTransactionFailureGetFee: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_get_fee_failure_total",
				Help: "The total number of sending transaction failure for getting fee.",
			}, []string{"service", "name"}),
			sendTransactionFailureSendTx: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_send_tx_failure_total",
				Help: "The total number of sending transaction failure for sending tx.",
			}, []string{"service", "name"}),
			resubmitTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_send_transaction_resubmit_send_transaction_total",
				Help: "The total number of resubmit transaction.",
			}, []string{"service", "name"}),
			currentPendingTxsNum: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_pending_tx_count",
				Help: "The pending tx count in the sender.",
			}, []string{"service", "name"}),
			currentGasFeeCap: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_fee_cap",
				Help: "The gas fee of current transaction.",
			}, []string{"service", "name"}),
			currentGasTipCap: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_tip_cap",
				Help: "The gas tip of current transaction.",
			}, []string{"service", "name"}),
			currentGasPrice: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_price_cap",
				Help: "The gas price of current transaction.",
			}, []string{"service", "name"}),
			currentGasLimit: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_gas_limit",
				Help: "The gas limit of current transaction.",
			}, []string{"service", "name"}),
			currentNonce: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
				Name: "rollup_sender_nonce",
				Help: "The nonce of current transaction.",
			}, []string{"service", "name"}),
			senderCheckPendingTransactionTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_check_pending_transaction_total",
				Help: "The total number of check pending transaction.",
			}, []string{"service", "name"}),
			senderCheckBalancerTotal: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
				Name: "rollup_sender_check_balancer_total",
				Help: "The total number of check balancer.",
			}, []string{"service", "name"}),
		}
	})

	return sm
}
