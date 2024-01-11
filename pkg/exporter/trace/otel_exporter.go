package trace

import (
	"context"
	"fmt"
	"time"

	"github.com/kwaisu/sense-agent/pkg/ebpftracer/l7"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog/v2"
)

type otelExporter struct {
	tracer oteltrace.Tracer
}

func NewExporter(machineId, hostname, version, endpoint, serviceName string) (Exporter, error) {
	klog.Info(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("OpenTelemetry logs collector endpoint is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "endpoint",
		// Note the use of insecure transport here. TLS is recommended in production.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("sense-agent"),
			semconv.ServiceVersion(version),
		)),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tracer := otel.Tracer("sense-agent")
	return &otelExporter{tracer: tracer}, nil
}

func (t *otelExporter) createSpan(name string, duration time.Duration, error bool, attrs ...attribute.KeyValue) {

}

func (t *otelExporter) HttpRequest(method, path string, status l7.Status, duration time.Duration) {

}
