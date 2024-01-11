package log

import (
	"context"
	"fmt"
	"time"

	otel "github.com/agoda-com/opentelemetry-logs-go"
	"github.com/agoda-com/opentelemetry-logs-go/exporters/otlp/otlplogs"
	"github.com/agoda-com/opentelemetry-logs-go/exporters/otlp/otlplogs/otlplogshttp"
	"github.com/kwaisu/sense-agent/pkg/log"
	"k8s.io/klog/v2"

	"github.com/agoda-com/opentelemetry-logs-go/logs"
	sdk "github.com/agoda-com/opentelemetry-logs-go/sdk/logs"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

type otelExporter struct {
	log logs.Logger
}

var _ Exporter = (*otelExporter)(nil)

func NewExporter(machineId, hostname, version, endpoint, serviceName string) (Exporter, error) {
	klog.Info(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("OpenTelemetry logs collector endpoint is nil")
	}
	exporter, err := otlplogs.NewExporter(context.Background(), otlplogs.WithClient(otlplogshttp.NewClient(otlplogshttp.WithEndpoint(endpoint), otlplogshttp.WithProtobufProtocol(), otlplogshttp.WithInsecure())))

	if err != nil {
		return nil, err
	}
	loggerProvider := sdk.NewLoggerProvider(
		sdk.WithBatcher(exporter),
		sdk.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName(serviceName),
				semconv.ServiceVersion(version),
				semconv.HostName(hostname),
				semconv.HostID(machineId),
			),
		),
	)
	otel.SetLoggerProvider(loggerProvider)
	logger := loggerProvider.Logger("sense-agent", logs.WithInstrumentationVersion(version))
	return &otelExporter{log: logger}, nil
}
func (e *otelExporter) Export() ExportMessageF {
	return func(messages []log.Message) {
		for _, msg := range messages {
			start := time.Now()
			severityText := msg.Level
			severityNumber := Level2OtelLogsLevel(severityText)
			attributes := map2Attribute(msg.Fields)
			meta := map2Attribute(msg.Meta)
			klog.Info("otel exporter send record start")
			e.log.Emit(
				logs.NewLogRecord(
					logs.LogRecordConfig{
						ObservedTimestamp: msg.Timestamp,
						SeverityText:      &severityText,
						SeverityNumber:    &severityNumber,
						Body:              &msg.Content,
						Resource: resource.NewSchemaless(
							meta...,
						),
						Attributes: &attributes,
					},
				),
			)
			klog.Info("otel exporter send record end,time: ", time.Since(start))
		}
	}
}

func map2Attribute(resource map[string]string) []attribute.KeyValue {
	attr := make([]attribute.KeyValue, len(resource))
	for k, v := range resource {
		attr = append(attr, attribute.String(k, v))
	}
	return attr
}

func Level2OtelLogsLevel(levelText string) logs.SeverityNumber {
	level := log.String2Level(levelText)
	switch level {
	case log.LevelDebug:
		return logs.DEBUG
	case log.LevelInfo:
		return logs.INFO
	case log.LevelWarning:
		return logs.WARN
	case log.LevelError:
		return logs.ERROR
	case log.LevelCritical:
		return logs.FATAL
	}
	return logs.UNSPECIFIED
}
