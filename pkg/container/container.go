package container

import (
	"fmt"

	"github.com/kwaisu/sense-agent/pkg/cgroup"
	"github.com/kwaisu/sense-agent/pkg/ebpftracer"
	"github.com/kwaisu/sense-agent/pkg/proc"
	"k8s.io/klog/v2"
)

type ContainerContext struct {
	*ContainerClientProvider
	containersById       map[ContainerID]*Container
	containersByCgroupId map[string]*Container
	containersByPid      map[uint32]*Container
	events               chan ebpftracer.Event
}

func NewContainerContext() (*ContainerContext, error) {
	cgroup.InitCgroup()
	ctx := &ContainerContext{
		ContainerClientProvider: NewContainerClientProvider(),
		events:                  make(chan ebpftracer.Event, 10000),
		containersById:          map[ContainerID]*Container{},
		containersByCgroupId:    map[string]*Container{},
		containersByPid:         map[uint32]*Container{},
	}
	go ctx.handleEvents(ctx.events)
	//init container
	ctx.initContainer(ctx.events)
	return ctx, nil
}

func (ctx *ContainerContext) initContainer(ch chan<- ebpftracer.Event) {
	pids, err := proc.GetAllPids()
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
			klog.Info(event.Type)
			switch event.Type {
			case ebpftracer.EventTypeProcessStart:
				ctx.getContainer(event.Pid)
			}
		}
	}
}

func (ctx *ContainerContext) getContainer(pid uint32) *Container {
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
			var id ContainerID
			if metadata.Labels["io.kubernetes.pod.name"] != "" {
				pod := metadata.Labels["io.kubernetes.pod.name"]
				namespace := metadata.Labels["io.kubernetes.pod.namespace"]
				name := metadata.Labels["io.kubernetes.container.name"]
				if cg.ContainerType == cgroup.ContainerTypeSandbox {
					name = "sandbox"
				}
				if name == "" || name == "POD" { // skip pause containers
					id = ""
				}
				id = ContainerID(fmt.Sprintf("/k8s/%s/%s/%s", namespace, pod, name))
			}

			klog.Infof("calculated container id %d -> %s -> %s", pid, cg.Id, id)
			if id == "" {
				if cg.Id == "/init.scope" && pid != 1 {
					klog.InfoS("ignoring without persisting", "cg", cg.Id, "pid", pid)
				} else {
					klog.InfoS("ignoring", "cg", cg.Id, "pid", pid)
					ctx.containersByPid[pid] = nil
				}
				return nil
			}
			// if c := ctx.containersById[id]; c != nil {
			// 	klog.Warningln("id conflict:", id)
			// 	if cg.CreatedAt().After(c.cgroup.CreatedAt()) {
			// 		c.cgroup = cg
			// 		c.metadata = md
			// 		c.runLogParser("")
			// 		if c.nsConntrack != nil {
			// 			_ = c.nsConntrack.Close()
			// 			c.nsConntrack = nil
			// 		}
			// 	}
			// 	r.containersByPid[pid] = c
			// 	r.containersByCgroupId[cg.Id] = c
			// 	return c
			// }

		}

	}

	return nil
}
