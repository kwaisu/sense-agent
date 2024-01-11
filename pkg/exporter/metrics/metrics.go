package metrics

import "github.com/prometheus/client_golang/prometheus"

type ContianerMetrics struct {
	ContainerInfo *prometheus.Desc
}

var metrics = &ContianerMetrics{
	ContainerInfo: metricDesc("container_info", "Meta information about the container", "image", "name", "labels", "annotations"),
}

func metricDesc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc(name, help, labels, nil)
}

func NewMetrics(desc *prometheus.Desc, value float64, labelValues ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, value, labelValues...)
}
