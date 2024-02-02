package container

import (
	"fmt"
	"strings"

	"k8s.io/klog/v2"

	"github.com/kwaisu/sense-agent/pkg/cgroup"
	"github.com/kwaisu/sense-agent/pkg/ebpftracer"
	"github.com/kwaisu/sense-agent/pkg/kubernetes"
	"github.com/kwaisu/sense-agent/pkg/system"
)

type ContainerContext struct {
	*ContainerClientProvider
	containersById       map[string]*Container
	containersByCgroupId map[string]*Container
	containersByPid      map[uint32]*Container
	events               chan ebpftracer.Event
	ebpftracer           *ebpftracer.EBPFTracer
	conntrack            *system.Conntrack
}

func NewContainerContext(kernelVersion string) (*ContainerContext, error) {
	cgroup.InitCgroup()
	ctx := &ContainerContext{
		ContainerClientProvider: NewContainerClientProvider(),
		events:                  make(chan ebpftracer.Event, 10000),
		containersById:          map[string]*Container{},
		containersByCgroupId:    map[string]*Container{},
		containersByPid:         map[uint32]*Container{},
	}
	if ebpftracer, err := ebpftracer.NewTracer(kernelVersion); err != nil {
		klog.Warning(err)
	} else {
		ctx.ebpftracer = ebpftracer
	}
	if conntrack, err := system.NewHostNetConntrack(); err != nil {
		return nil, err
	} else {
		ctx.conntrack = conntrack
	}
	ctx.ebpfEventSubscribe()
	go ctx.handleEvents(ctx.events)
	ctx.initContainer(ctx.events)
	return ctx, nil
}

func (ctx *ContainerContext) ebpfEventSubscribe() {
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeProcessStart, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeProcessExit, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeConnectionOpen, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeConnectionClose, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeConnectionError, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeListenOpen, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeListenClose, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeFileOpen, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeTCPRetransmit, ctx.events)
	ctx.ebpftracer.SubscribeEvents(ebpftracer.EventTypeL7Request, ctx.events)
}

func (ctx *ContainerContext) initContainer(ch chan<- ebpftracer.Event) {
	pids, err := system.GetAllPids()
	if err != nil {
		klog.Warning(fmt.Errorf("failed to list pids: %w", err))
	}
	for _, pid := range pids {
		ch <- ebpftracer.Event{Type: ebpftracer.EventTypeProcessStart, Pid: pid}
	}
}

func (ctx *ContainerContext) handleEvents(ch <-chan ebpftracer.Event) {
	for {
		select {
		case event, more := <-ch:
			if !more {
				return
			}
			switch event.Type {
			case ebpftracer.EventTypeProcessStart:
				ctx.createContainer(event.Pid)
			case ebpftracer.EventTypeProcessExit:
				if c, exists := ctx.containersByPid[event.Pid]; exists {
					delete(ctx.containersByCgroupId, c.Cgroup.Id)
					delete(ctx.containersById, c.ContainerID)
					delete(ctx.containersByPid, event.Pid)
				}
			case ebpftracer.EventTypeConnectionOpen:
				// if c, _ := ctx.containersByPid[event.Pid]; c != nil {
				// 	klog.Infof("contianer: %s, pid = %d,Fd = %d, srcadd = %s, destaddr =  %s,Timestamp =  %d", c.Metadata.Name, event.Pid, event.Fd, event.SrcAddr.IP().String(), event.DstAddr.IP().String(), event.Timestamp)
				// 	// 	actualDst := ctx.conntrack.GetActualDestination(event.SrcAddr, event.DstAddr)
				// 	// 	if actualDst != nil {
				// 	// 		klog.Warningf("cannot open NetNs for pid %d ", event.Pid)
				// 	// 		return
				// 	// 	}
				// } else {
				// 	klog.Infoln("TCP connection from unknown container", event)
				// }

			case ebpftracer.EventTypeL7Request:

			}
		}
	}
}

func (ctx *ContainerContext) createContainer(pid uint32) *Container {
	if container, ok := ctx.containersByPid[pid]; ok {
		return container
	}
	cg, err := cgroup.ReadCgroupByPid(pid)
	if err != nil {
		klog.Warningln("failed to read proc cgroup:", err)
		return nil
	}
	if c, ok := ctx.containersByCgroupId[cg.Id]; ok {
		ctx.containersByPid[pid] = c
		return c
	}
	if cg.ContainerType == cgroup.ContainerTypeDocker || cg.ContainerType == cgroup.ContainerTypeContainerd {
		if metadata, err := ctx.client.GetContainerMetadata(cg.ContainerId); err != nil {
			klog.Warningf("failed to get container metadata for pid %d -> %s: %s", pid, cg.Id, err)
			return nil
		} else {
			id := getContainerID(cg, metadata)
			if id == "" {
				if cg.Id == "/init.scope" && pid != 1 {
					klog.InfoS("ignoring without persisting", "cg", cg.Id, "pid", pid)
				} else {
					klog.InfoS("ignoring", "cg", cg.Id, "pid", pid)
					ctx.containersByPid[pid] = nil
				}
				return nil
			}
			if c, err := ctx.NewContainer(id, cg, pid); err != nil {
				if err != nil {
					klog.Warningf("failed to create container pid=%d cg=%s id=%s: %s", pid, cg.Id, id, err)
					return nil
				}
			} else {
				klog.InfoS("container:", "pid", pid, "cg", cg.Id, "id", id)
				ctx.containersByPid[pid] = c
				ctx.containersByCgroupId[cg.Id] = c
				ctx.containersById[id] = c
			}
		}
	}
	return nil
}

func getContainerID(cg *cgroup.Cgroup, meta *ContainerMetadata) string {
	if cg.ContainerType == cgroup.ContainerTypeSystemdService {
		if strings.HasPrefix(cg.ContainerId, "/system.slice/crio-conmon-") {
			return ""
		}
		return cg.ContainerId
	}
	if cg.ContainerId == "" {
		return ""
	}
	if cg.ContainerType != cgroup.ContainerTypeDocker && cg.ContainerType != cgroup.ContainerTypeContainerd && cg.ContainerType != cgroup.ContainerTypeSandbox {
		return ""
	}
	if meta.Labels[kubernetes.KUBERNETES_LABEL_PODNAME] != "" {
		pod := meta.Labels[kubernetes.KUBERNETES_LABEL_PODNAME]
		namespace := meta.Labels[kubernetes.KUBERNETES_LABEL_NAMESPACE]
		name := meta.Labels[kubernetes.KUBERNETES_LABEL_CONTAINER_NAME]
		if cg.ContainerType == cgroup.ContainerTypeSandbox {
			name = "sandbox"
		}
		if name == "" || name == "POD" { // skip pause containers
			return ""
		}
		return fmt.Sprintf("/k8s/%s/%s/%s", namespace, pod, name)
	}
	return ""
}
