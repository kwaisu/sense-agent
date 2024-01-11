package log

import (
	"context"
	"time"

	"github.com/kwaisu/sense-agent/pkg/log"
)

type LogExporterType int32

const (
	Otel LogExporterType = iota - 1
)

// export messsage call back function
type ExportMessageF func(msg []log.Message)
type Exporter interface {
	Export() ExportMessageF
}

type ExportProvider struct {
	exporter Exporter
	message  chan log.Message
	stop     context.CancelFunc
	config   ExportProverConfig
}

type ExportProverConfig struct {
	maxBytes int
	maxLines int
	timeout  time.Duration
}

func NewLoggerProvider(exporter Exporter, message chan log.Message, config ExportProverConfig) *ExportProvider {
	return &ExportProvider{
		exporter: exporter,
		message:  message,
		config:   config,
	}
}

func (e *ExportProvider) Start() {
	ctx, stop := context.WithCancel(context.Background())
	e.stop = stop
	msgBuffer := log.NewMessageBuffer(e.config.maxBytes, e.config.maxLines, e.config.timeout)
	// read  message
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-e.message:
				msgBuffer.Add(msg)
			}
		}
	}()
	//export message
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msgBuffer := <-msgBuffer.MessagesChan:
				e.exporter.Export()(msgBuffer)
			}
		}
	}()

}
func (e *ExportProvider) Stop() {
	e.stop()
}
