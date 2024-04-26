package container

import (
	"sync"
	"time"

	"inet.af/netaddr"
	"k8s.io/klog/v2"

	"github.com/kwaisu/sense-agent/pkg/cgroup"
	"github.com/kwaisu/sense-agent/pkg/ebpftracer/l7"
	"github.com/kwaisu/sense-agent/pkg/system"
)

const TIMEOUT = 30 * time.Second

type Container struct {
	ContainerID        string
	Metadata           *ContainerMetadata
	Cgroup             *cgroup.Cgroup
	Pid                uint32
	lock               sync.RWMutex
	connectionsByPidFd map[PidFd]*ActiveConnection
	connectsSuccessful map[AddrPair]int64       // dst:actual_dst -> count
	connectsFailed     map[netaddr.IPPort]int64 // dst -> count
	connectionsActive  map[AddrPair]*ActiveConnection
	connectLastAttempt map[netaddr.IPPort]time.Time // dst -> time
	hostConntrack      *system.Conntrack
}
type PidFd struct {
	Pid uint32
	Fd  uint64
}
type AddrPair struct {
	src netaddr.IPPort
	dst netaddr.IPPort
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

func (c *ContainerClientProvider) NewContainer(containerID string, metadata *ContainerMetadata, cg *cgroup.Cgroup, pid uint32, hostConntrack *system.Conntrack) (*Container, error) {
	contianer := &Container{
		ContainerID:        containerID,
		Cgroup:             cg,
		Pid:                pid,
		Metadata:           metadata,
		hostConntrack:      hostConntrack,
		connectionsByPidFd: make(map[PidFd]*ActiveConnection),
		connectsSuccessful: make(map[AddrPair]int64),
		connectsFailed:     make(map[netaddr.IPPort]int64),
		connectionsActive:  make(map[AddrPair]*ActiveConnection),
		connectLastAttempt: make(map[netaddr.IPPort]time.Time),
	}
	return contianer, nil
}

func (c *Container) OnL7Request(pid uint32, fd uint64, timestamp uint64, r *l7.RequestData) {
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

func (c *Container) OnConnectionOpen(srcAddr, dstAddr netaddr.IPPort, pid uint32, fd uint64, timestamp uint64, isConnectError bool) {
	if dstAddr.IP().IsLoopback() {
		return
	}
	actualDst, _ := c.getActualDestination(srcAddr, dstAddr)
	if actualDst == nil {
		actualDst = &dstAddr
	}
	activeConnection := &ActiveConnection{
		ActualDest: *actualDst,
		Pid:        pid,
		Fd:         fd,
		Timestamp:  timestamp,
		Dest:       dstAddr,
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if isConnectError {
		c.connectsFailed[dstAddr]++
		klog.Infof("OnConnectionError contianer: %s, pid = %d,Fd = %d, srcadd = %s:%d, destaddr =  %s:%d,Timestamp =  %d", c.Metadata.Name, pid, fd, srcAddr.IP().String(), srcAddr.Port(), dstAddr.IP().String(), dstAddr.Port(), timestamp)
	} else {
		c.connectionsActive[AddrPair{src: srcAddr, dst: *actualDst}] = activeConnection
		c.connectionsByPidFd[PidFd{Pid: pid, Fd: fd}] = activeConnection
		c.connectsSuccessful[AddrPair{src: srcAddr, dst: *actualDst}]++
		klog.Infof("OnConnectionOpen contianer: %s, pid = %d,Fd = %d, srcadd = %s:%d, destaddr =  %s:%d,Timestamp =  %d", c.Metadata.Name, pid, fd, srcAddr.IP().String(), srcAddr.Port(), dstAddr.IP().String(), dstAddr.Port(), timestamp)
	}
	c.connectLastAttempt[dstAddr] = time.Now()

}

func (c *Container) getActualDestination(src, dst netaddr.IPPort) (*netaddr.IPPort, error) {
	actualDst := c.hostConntrack.GetActualDestination(src, dst)
	if actualDst != nil {
		return actualDst, nil
	}
	return nil, nil
}
