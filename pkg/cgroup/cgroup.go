package cgroup

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"

	"github.com/kwaisu/sense-agent/pkg/proc"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
	"k8s.io/klog/v2"
)

type Version int

const (
	V1 Version = iota
	V2
)

type ContainerType uint8

const (
	ContainerTypeUnknown ContainerType = iota
	ContainerTypeStandaloneProcess
	ContainerTypeDocker
	ContainerTypeSystemdService
	ContainerTypeContainerd
	ContainerTypeSandbox
)

func (t ContainerType) String() string {
	switch t {
	case ContainerTypeStandaloneProcess:
		return "standalone"
	case ContainerTypeDocker:
		return "docker"
	case ContainerTypeContainerd:
		return "cri-containerd"
	case ContainerTypeSystemdService:
		return "systemd"
	default:
		return "unknown"
	}
}

var (
	GlobalCgroup        string
	dockerIdRegexp      = regexp.MustCompile(`([a-z0-9]{64})`)
	containerdIdRegexp  = regexp.MustCompile(`cri-containerd[-:]([a-z0-9]{64})`)
	systemSliceIdRegexp = regexp.MustCompile(`(/(system|runtime)\.slice/([^/]+))`)
)

type Cgroup struct {
	Id            string
	Version       Version
	ContainerType ContainerType
	ContainerId   string
	subsystems    map[string]string
}

func InitCgroup() error {
	selfNs, err := netns.GetFromPath("/proc/self/ns/cgroup")
	if err != nil {
		return err
	}
	defer selfNs.Close()
	hostNs, err := netns.GetFromPath("/proc/1/ns/cgroup")
	if err != nil {
		return err
	}
	defer hostNs.Close()
	if selfNs.Equal(hostNs) {
		return nil
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := unix.Setns(int(hostNs), unix.CLONE_NEWCGROUP); err != nil {
		return err
	}

	cg, err := ReadCgroupFromFile("/proc/self/cgroup")
	if err != nil {
		return err
	}
	GlobalCgroup = cg.Id

	if err := unix.Setns(int(selfNs), unix.CLONE_NEWCGROUP); err != nil {
		return err
	}

	return nil

}

func ReadCgroupByPid(pid uint32) (*Cgroup, error) {
	return ReadCgroupFromFile(proc.Path(pid, "cgroup"))
}

func ReadCgroupFromFile(file string) (*Cgroup, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	cg := &Cgroup{
		subsystems: map[string]string{},
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		for _, cgType := range strings.Split(parts[1], ",") {
			cg.subsystems[cgType] = path.Join(GlobalCgroup, parts[2])
		}
	}
	if p := cg.subsystems["cpu"]; p != "" {
		cg.Id = p
		cg.Version = V1
	} else {
		cg.Id = cg.subsystems[""]
		cg.Version = V2
	}
	klog.Info(cg.Id)
	if cg.ContainerType, cg.ContainerId, err = containerByCgroup(cg.Id); err != nil {
		return nil, err
	}
	return cg, nil
}

func containerByCgroup(path string) (ContainerType, string, error) {
	parts := strings.Split(strings.TrimLeft(path, "/"), "/")
	if len(parts) < 2 {
		return ContainerTypeStandaloneProcess, "", nil
	}
	prefix := parts[0]
	if prefix == "user.slice" || prefix == "init.scope" {
		return ContainerTypeStandaloneProcess, "", nil
	}
	//docker 9:memory:/system.slice/docker-3b4c4778e0f6640cb1cddd1eddf22638b37ef6e44ce3a6a6264262ccc0353232.scope
	//8:pids:/docker/ffc408b364e6d265434bf40ec532d8d1380c4ec35d1e0b1494ef8cefa4334d90
	if prefix == "docker" || (prefix == "system.slice" && strings.HasPrefix(parts[1], "docker-")) {
		matches := dockerIdRegexp.FindStringSubmatch(path)
		if matches == nil {
			return ContainerTypeUnknown, "", fmt.Errorf("invalid docker cgroup %s", path)
		}
		return ContainerTypeDocker, matches[1], nil
	}
	//0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod*.slice/cri-containerd-*
	if strings.Contains(path, "kubepods") {
		containerdMatches := containerdIdRegexp.FindStringSubmatch(path)
		if containerdMatches != nil {
			return ContainerTypeContainerd, containerdMatches[1], nil
		}
		matches := dockerIdRegexp.FindStringSubmatch(path)
		if matches == nil {
			return ContainerTypeSandbox, "", nil
		}
		return ContainerTypeDocker, matches[1], nil
	}
	if prefix == "system.slice" || prefix == "runtime.slice" {
		matches := systemSliceIdRegexp.FindStringSubmatch(path)
		if matches == nil {
			return ContainerTypeUnknown, "", fmt.Errorf("invalid systemd cgroup %s", path)
		}
		return ContainerTypeSystemdService, matches[1], nil
	}
	return ContainerTypeUnknown, "", fmt.Errorf("unknown container: %s", path)
}
