package container

import (
	"testing"

	"k8s.io/klog/v2"
)

func TestDocker(t *testing.T) {
	containerdClient, err := NewDockerd()
	if err == nil && containerdClient != nil {
		if ids, err := containerdClient.ListContainerID(); err == nil {
			for _, id := range ids {
				if metadata, err := containerdClient.GetContainerMetadata(id); err == nil {
					klog.Info(metadata)
				}
			}
		}
	}
}
func TestContainerd(t *testing.T) {
	containerdClient, err := NewContainerd()
	if err == nil && containerdClient != nil {
		if ids, err := containerdClient.ListContainerID(); err == nil {
			for _, id := range ids {
				if metadata, err := containerdClient.GetContainerMetadata(id); err == nil {
					klog.Info(metadata)
				}
			}
		}
	}

}

func TestNewContainerContext(t *testing.T) {
	NewContainerContext()
	for {

	}
}
