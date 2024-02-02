package container

import (
	"sync"
	"time"

	"inet.af/netaddr"
	"k8s.io/klog/v2"

	"github.com/kwaisu/sense-agent/pkg/cgroup"
	"github.com/kwaisu/sense-agent/pkg/ebpftracer/l7"
)

const TIMEOUT = 30 * time.Second

type Container struct {
	ContainerID        string
	Metadata           *ContainerMetadata
	Cgroup             *cgroup.Cgroup
	Pid                uint32
	lock               sync.RWMutex
	connectionsByPidFd map[PidFd]*ActiveConnection
}
type PidFd struct {
	Pid uint32
	Fd  uint64
}
type ActiveConnection struct {
	Dest       netaddr.IPPort
	ActualDest netaddr.IPPort
	Pid        uint32
	Fd         uint64
	Timestamp  uint64
	Closed     time.Time
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

func (c *ContainerClientProvider) NewContainer(containerID string, cg *cgroup.Cgroup, pid uint32) (*Container, error) {
	metadata, err := c.client.GetContainerMetadata(containerID)
	if err != nil {
		klog.Warning("")
	}
	return &Container{ContainerID: containerID, Cgroup: cg, Pid: pid, Metadata: metadata}, nil
}

func (c *Container) onL7Request(pid uint32, fd uint64, timestamp uint64, r *l7.RequestData) {
	c.lock.Lock()
	defer c.lock.Unlock()

	conn := c.connectionsByPidFd[PidFd{Pid: pid, Fd: fd}]
	if conn == nil {
		return
	}
	if timestamp != 0 && conn.Timestamp != timestamp {
		return
	}

}
