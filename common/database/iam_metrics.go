package database

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type rdsIAMMetrics struct {
	tokenTotal        prometheus.Counter
	tokenFailureTotal prometheus.Counter
	tokenDuration     prometheus.Histogram
}

var (
	initRDSIAMMetricsOnce sync.Once
	rdsIAMMetric          *rdsIAMMetrics
)

func initRDSIAMMetrics() *rdsIAMMetrics {
	initRDSIAMMetricsOnce.Do(func() {
		reg := prometheus.DefaultRegisterer
		rdsIAMMetric = &rdsIAMMetrics{
			tokenTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "database_rds_iam_token_total",
				Help: "Total number of AWS RDS IAM auth tokens generated (one per new connection).",
			}),
			tokenFailureTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "database_rds_iam_token_failure_total",
				Help: "Total number of AWS RDS IAM auth token generation failures.",
			}),
			tokenDuration: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
				Name:    "database_rds_iam_token_duration_seconds",
				Help:    "Latency of AWS RDS IAM auth token generation; spikes indicate a credential refresh.",
				Buckets: []float64{.0001, .001, .005, .01, .05, .1, .5, 1, 5},
			}),
		}
	})
	return rdsIAMMetric
}
