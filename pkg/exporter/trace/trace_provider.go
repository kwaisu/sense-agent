package trace

import (
	"time"

	"github.com/kwaisu/sense-agent/pkg/ebpftracer/l7"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"inet.af/netaddr"
)

type Trace struct {
	containerId string
	destination netaddr.IPPort
	commonAttrs []attribute.KeyValue
}

// New Trace
func (t *TraceProvider) NewTrace(containerId string, destination netaddr.IPPort) *Trace {
	if t.traceExporter == nil {
		return nil
	}
	return &Trace{containerId: containerId, destination: destination, commonAttrs: []attribute.KeyValue{
		semconv.ContainerID(containerId),
		semconv.NetPeerName(destination.IP().String()),
		semconv.NetPeerPort(int(destination.Port())),
	}}
}

type Exporter interface {
	createSpan(name string, duration time.Duration, error bool, attrs ...attribute.KeyValue)
	HttpRequest(method, path string, status l7.Status, duration time.Duration)
}

type TraceProvider struct {
	traceExporter Exporter
}
