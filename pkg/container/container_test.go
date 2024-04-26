package container

import (
	"testing"

	"github.com/kwaisu/sense-agent/pkg/system"
)

// func TestDocker(t *testing.T) {
// 	containerdClient, err := NewDockerd()
// 	if err == nil && containerdClient != nil {
// 		if ids, err := containerdClient.ListContainerID(); err == nil {
// 			for _, id := range ids {
// 				if metadata, err := containerdClient.GetContainerMetadata(id); err == nil {
// 					klog.Info(metadata)
// 				}
// 			}
// 		}
// 	}
// }
// func TestContainerd(t *testing.T) {
// 	containerdClient, err := NewContainerd()
// 	if err == nil && containerdClient != nil {
// 		if ids, err := containerdClient.ListContainerID(); err == nil {
// 			for _, id := range ids {
// 				if metadata, err := containerdClient.GetContainerMetadata(id); err == nil {
// 					klog.Info(metadata)
// 				}
// 			}
// 		}
// 	}

// }

func TestNewContainerContext(t *testing.T) {
	_, kernelVersion, _ := system.Uname()
	if ctx, err := NewContainerContext(kernelVersion); err == nil {
		ctx.ebpftracer.Run()
		for {

		}
	}
}
