package ginmetrics

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricType define metric type
type MetricType int

const (
	// None unknown metric type
	None MetricType = iota
	// Counter MetricType
	Counter
	// Gauge MetricType
	Gauge
	// Histogram MetricType
	Histogram
	// Summary MetricType
	Summary

	defaultMetricPath = "/debug/metrics"
	defaultSlowTime   = int32(5)
)

var (
	defaultDuration = []float64{0.1, 0.3, 1.2, 5, 10}
	monitor         *Monitor

	promTypeHandler = map[MetricType]func(metric *Metric, reg prometheus.Registerer){
		Counter:   counterHandler,
		Gauge:     gaugeHandler,
		Histogram: histogramHandler,
		Summary:   summaryHandler,
	}
)

// Monitor is an object that uses to set gin server monitor.
type Monitor struct {
	slowTime    int32
	metricPath  string
	reqDuration []float64
	metrics     map[string]*Metric
	register    prometheus.Registerer
}

// GetMonitor used to get global Monitor object,
// this function returns a singleton object.
func GetMonitor(reg prometheus.Registerer) *Monitor {
	if monitor == nil {
		monitor = &Monitor{
			metricPath:  defaultMetricPath,
			slowTime:    defaultSlowTime,
			reqDuration: defaultDuration,
			metrics:     make(map[string]*Metric),
			register:    reg,
		}
	}
	return monitor
}

// GetMetric used to get metric object by metric_name.
func (m *Monitor) GetMetric(name string) *Metric {
	if metric, ok := m.metrics[name]; ok {
		return metric
	}
	return &Metric{}
}

// SetMetricPath set metricPath property. metricPath is used for Prometheus
// to get gin server monitoring data.
func (m *Monitor) SetMetricPath(path string) {
	m.metricPath = path
}

// SetSlowTime set slowTime property. slowTime is used to determine whether
// the request is slow. For "gin_slow_request_total" metric.
func (m *Monitor) SetSlowTime(slowTime int32) {
	m.slowTime = slowTime
}

// SetDuration set reqDuration property. reqDuration is used to ginRequestDuration
// metric buckets.
func (m *Monitor) SetDuration(duration []float64) {
	m.reqDuration = duration
}

// SetMetricPrefix set the metric prefix
func (m *Monitor) SetMetricPrefix(prefix string) {
	metricRequestTotal = prefix + metricRequestTotal
	metricRequestUVTotal = prefix + metricRequestUVTotal
	metricURIRequestTotal = prefix + metricURIRequestTotal
	metricRequestBody = prefix + metricRequestBody
	metricResponseBody = prefix + metricResponseBody
	metricRequestDuration = prefix + metricRequestDuration
	metricSlowRequest = prefix + metricSlowRequest
}

// SetMetricSuffix set the metric suffix
func (m *Monitor) SetMetricSuffix(suffix string) {
	metricRequestTotal += suffix
	metricRequestUVTotal += suffix
	metricURIRequestTotal += suffix
	metricRequestBody += suffix
	metricResponseBody += suffix
	metricRequestDuration += suffix
	metricSlowRequest += suffix
}

// AddMetric add custom monitor metric.
func (m *Monitor) AddMetric(metric *Metric) error {
	if _, ok := m.metrics[metric.Name]; ok {
		return fmt.Errorf("metric %s is existed", metric.Name)
	}

	if metric.Name == "" {
		return errors.New("metric name cannot be empty")
	}

	if f, ok := promTypeHandler[metric.Type]; ok {
		f(metric, m.register)
		m.metrics[metric.Name] = metric
	}

	return nil
}

func counterHandler(metric *Metric, register prometheus.Registerer) {
	metric.vec = promauto.With(register).NewCounterVec(
		prometheus.CounterOpts{Name: metric.Name, Help: metric.Description},
		metric.Labels,
	)
}

func gaugeHandler(metric *Metric, register prometheus.Registerer) {
	metric.vec = promauto.With(register).NewGaugeVec(
		prometheus.GaugeOpts{Name: metric.Name, Help: metric.Description},
		metric.Labels,
	)
}

func histogramHandler(metric *Metric, register prometheus.Registerer) {
	metric.vec = promauto.With(register).NewHistogramVec(
		prometheus.HistogramOpts{Name: metric.Name, Help: metric.Description, Buckets: metric.Buckets},
		metric.Labels,
	)
}

func summaryHandler(metric *Metric, register prometheus.Registerer) {
	promauto.With(register).NewSummaryVec(
		prometheus.SummaryOpts{Name: metric.Name, Help: metric.Description, Objectives: metric.Objectives},
		metric.Labels,
	)
}
