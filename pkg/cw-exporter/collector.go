package cwexporter

import "github.com/prometheus/client_golang/prometheus"

type PrometheusCollector struct {
	metrics []prometheus.Metric
}

func NewPrometheusCollector() *PrometheusCollector {
	return &PrometheusCollector{}
}

func (p *PrometheusCollector) Describe(descs chan<- *prometheus.Desc) {
	for _, metric := range p.metrics {
		descs <- metric.Desc()
	}
}

func (p *PrometheusCollector) Collect(metrics chan<- prometheus.Metric) {
	for _, metric := range p.metrics {
		metrics <- metric
	}
}

func (p *PrometheusCollector) addMetric(metric prometheus.Metric) {
	p.metrics = append(p.metrics, metric)
}

func (p *PrometheusCollector) reset() {
	if len(p.metrics) > 0 {
		prometheus.Unregister(p)
	}
	p.metrics = []prometheus.Metric{}
}
