package ebpftracer

import (
	"testing"

	"github.com/kwaisu/sense-agent/pkg/proc"
	"k8s.io/klog"
)

func TestNewTracer(t *testing.T) {
	_, kernelVersion, err := proc.Uname()
	if err != nil {
		klog.Error(err)
	}
	tracer, er := NewTracer(kernelVersion)
	if er != nil {
		klog.Error(er)
	}
}
