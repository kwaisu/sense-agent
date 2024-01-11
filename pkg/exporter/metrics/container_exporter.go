package metrics

import (
	"encoding/json"

	"github.com/kwaisu/sense-agent/pkg/container"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

type ContainerExporter struct {
	container container.Container
}

func (c *ContainerExporter) Collect(ch chan<- prometheus.Metric) {
	if c.container.Metadata != nil {
		var labels, annotations string
		if c.container.Metadata.Labels != nil {
			jsonStr, err := json.Marshal(c.container.Metadata.Labels)
			if err != nil {
				klog.Warning(err)
			} else {
				labels = string(jsonStr)
			}
		}
		if c.container.Metadata.Annotations != nil {
			jsonStr, err := json.Marshal(c.container.Metadata.Annotations)
			if err != nil {
				klog.Warning(err)
			} else {
				annotations = string(jsonStr)
			}
		}

		dls := []string{c.container.Metadata.Image, c.container.Metadata.Name, labels, annotations}
		ch <- NewMetrics(metrics.ContainerInfo, 1, dls...)
	}

}
