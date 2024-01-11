package cgroup

import (
	"testing"

	"k8s.io/klog/v2"
)

func TestInit(t *testing.T) {
	InitCgroup()
}

func TestNewCgroup(t *testing.T) {
	cgroup, err := ReadCgroupByPid(23545)
	if err != nil {
		klog.Info(err)
	}
	klog.Info(cgroup.ContainerId)
}
