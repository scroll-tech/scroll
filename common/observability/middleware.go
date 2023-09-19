package observability

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"scroll-tech/common/observability/ginmetrics"
)

// Use register the gin metric
func Use(router *gin.Engine, metricsPrefix string, reg prometheus.Registerer) {
	m := ginmetrics.GetMonitor(reg)
	m.SetMetricPath("/metrics")
	m.SetMetricPrefix(metricsPrefix + "_")
	m.SetSlowTime(1)
	m.SetDuration([]float64{0.025, .05, .1, .5, 1, 5, 10})
	m.UseWithoutExposingEndpoint(router)
}
