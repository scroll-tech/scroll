package ginmetrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

// Metric defines a metric object. Users can use it to save
// metric data. Every metric should be globally unique by name.
type Metric struct {
	Type        MetricType
	Name        string
	Description string
	Labels      []string
	Buckets     []float64
	Objectives  map[float64]float64

	vec prometheus.Collector
}

// SetGaugeValue set data for Gauge type Metric.
func (m *Metric) SetGaugeValue(labelValues []string, value float64) error {
	if m.Type == None {
		return fmt.Errorf("metric %s not existed", m.Name)
	}

	if m.Type != Gauge {
		return fmt.Errorf("metric %s not Gauge type", m.Name)
	}
	m.vec.(*prometheus.GaugeVec).WithLabelValues(labelValues...).Set(value)
	return nil
}

// Inc increases value for Counter/Gauge type metric, increments
// the counter by 1
func (m *Metric) Inc(labelValues []string) error {
	if m.Type == None {
		return fmt.Errorf("metric %s not existed", m.Name)
	}

	if m.Type != Gauge && m.Type != Counter {
		return fmt.Errorf("metric %s not Gauge or Counter type", m.Name)
	}
	switch m.Type {
	case Counter:
		m.vec.(*prometheus.CounterVec).WithLabelValues(labelValues...).Inc()
	case Gauge:
		m.vec.(*prometheus.GaugeVec).WithLabelValues(labelValues...).Inc()
	}
	return nil
}

// Add adds the given value to the Metric object. Only
// for Counter/Gauge type metric.
func (m *Metric) Add(labelValues []string, value float64) error {
	if m.Type == None {
		return fmt.Errorf("metric %s not existed", m.Name)
	}

	if m.Type != Gauge && m.Type != Counter {
		return fmt.Errorf("metric %s not Gauge or Counter type", m.Name)
	}
	switch m.Type {
	case Counter:
		m.vec.(*prometheus.CounterVec).WithLabelValues(labelValues...).Add(value)
	case Gauge:
		m.vec.(*prometheus.GaugeVec).WithLabelValues(labelValues...).Add(value)
	}
	return nil
}

// Observe is used by Histogram and Summary type metric to
// add observations.
func (m *Metric) Observe(labelValues []string, value float64) error {
	if m.Type == 0 {
		return fmt.Errorf("metric %s not existed", m.Name)
	}
	if m.Type != Histogram && m.Type != Summary {
		return fmt.Errorf("metric %s not Histogram or Summary type", m.Name)
	}
	switch m.Type {
	case Histogram:
		m.vec.(*prometheus.HistogramVec).WithLabelValues(labelValues...).Observe(value)
	case Summary:
		m.vec.(*prometheus.SummaryVec).WithLabelValues(labelValues...).Observe(value)
	}
	return nil
}
