package container

import (
	"time"

	"inet.af/netaddr"
	"k8s.io/klog/v2"
)

const TIMEOUT = 30 * time.Second

type ContainerID string

type Container struct {
	ContainerID string
	Metadata    *ContainerMetadata
	Pid         uint32
}
type ContainerPort struct {
	Name          string
	ContainerPort uint32
	Protocol      string
	HostPort      uint32
}
type ContainerMetadata struct {
	Name           string
	ID             string
	Labels         map[string]string
	Annotations    map[string]string
	Volumes        map[string]string
	Created        string
	LogPath        string
	Image          string
	HostListens    map[string][]netaddr.IPPort
	Networks       map[string]ContainerNetwork
	ContainerPorts []ContainerPort
}

type ContainerNetwork struct {
	NetworkID string
	Aliases   []string
	IPAddress string
}

type ContainerClient interface {
	GetContainerMetadata(containerID string) (*ContainerMetadata, error)
	ListContainerID() ([]string, error)
}

type ContainerClientProvider struct {
	client ContainerClient
}

func NewContainerClientProvider() *ContainerClientProvider {
	containerClientContext := &ContainerClientProvider{}
	containerdClient, err := NewContainerd()
	if err == nil && containerdClient != nil {
		containerClientContext.client = containerdClient
		klog.Info("Detected containers: containerd ")
		return containerClientContext
	}
	dockerclient, err := NewDockerd()
	if err == nil && dockerclient != nil {
		containerClientContext.client = dockerclient
		klog.Info("Detected containers: dockerd ")
		return containerClientContext
	}
	klog.Warning("No available docker or containerd detected")
	return nil
}

func (c *ContainerClientProvider) NewContainer(containerID string, pid uint32) *Container {
	metadata, err := c.client.GetContainerMetadata(containerID)
	if err != nil {
		klog.Warning("")
	}
	return &Container{ContainerID: containerID, Pid: pid, Metadata: metadata}
}
