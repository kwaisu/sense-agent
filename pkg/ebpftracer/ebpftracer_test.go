package ebpftracer

import (
	"testing"

	"k8s.io/klog"

	"github.com/kwaisu/sense-agent/pkg/system"
)

func TestNewTracer(t *testing.T) {
	_, kernelVersion, err := system.Uname()
	if err != nil {
		klog.Error(err)
	}
	tracer, er := NewTracer(kernelVersion)
	if er != nil {
		klog.Error(er)
	}
	ch := make(chan Event)
	tracer.SubscribeEvents(EventTypeL7Request, ch)
	tracer.Run()
	for {
		select {
		case event, more := <-ch:
			if !more {
				continue
			}
			klog.Infof("L7 Method: %s , Protocol: %s, Status: %s ,Payload : %s ", event.L7Request.Method, event.L7Request.Protocol.String(), event.L7Request.Status, string(event.L7Request.Payload))
		}
	}
}
