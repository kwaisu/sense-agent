package container

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/pkg/cri/constants"
	"github.com/kwaisu/sense-agent/pkg/kubernetes"
	"github.com/kwaisu/sense-agent/pkg/proc"
	"k8s.io/klog"
)

var MetadataLabel = "io.cri-containerd.container.metadata"

type ContainerdClient struct {
	client *containerd.Client
}
type Config struct {
	Annotations map[string]string
}
type Metadata struct {
	LogPath string
	Name    string
	Config  Config
}
type ContainerdMetadata struct {
	Metadata Metadata
}

func NewContainerd() (ContainerClient, error) {
	sockets := "/run/containerd/containerd.sock"
	var addr string
	klog.Info(addr)
	cli, err := containerd.New(proc.ProcRootSubpath(sockets),
		containerd.WithDefaultNamespace(constants.K8sContainerdNamespace),
		containerd.WithTimeout(time.Second))
	if err != nil {
		return nil, err
	}
	if cli == nil {
		return nil, fmt.Errorf(
			"couldn't connect to containerd through the following UNIX sockets [%s]: %s",
			sockets, err,
		)
	}
	return &ContainerdClient{client: cli}, nil
}
func (c *ContainerdClient) ListContainerID() ([]string, error) {
	if c.client == nil {
		return nil, fmt.Errorf("containerd client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()
	if contianers, err := c.client.ContainerService().List(ctx); err != nil {
		return nil, err
	} else {
		contianerIDs := make([]string, 0, len(contianers))
		for _, c := range contianers {
			contianerIDs = append(contianerIDs, c.ID)
		}
		return contianerIDs, nil
	}
}

func (c *ContainerdClient) GetContainerMetadata(containerID string) (*ContainerMetadata, error) {
	if c.client == nil {
		return nil, fmt.Errorf("containerd client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()
	if contianer, err := c.client.ContainerService().Get(ctx, containerID); err != nil {
		return nil, err
	} else {
		metadata := &ContainerMetadata{
			ID:      contianer.ID,
			Labels:  contianer.Labels,
			Image:   contianer.Image,
			Created: contianer.CreatedAt.String(),
			Volumes: map[string]string{},
		}
		var spec oci.Spec
		if err := json.Unmarshal(contianer.Spec.GetValue(), &spec); err != nil {
			klog.Warningln(err)
		}
		for _, mount := range spec.Mounts {
			metadata.Volumes[mount.Destination] = mount.Source
		}
		if data, ok := contianer.Extensions[MetadataLabel]; ok {
			containerdMetadata := ContainerdMetadata{}
			if err := json.Unmarshal(data.GetValue(), &containerdMetadata); err != nil {
				klog.Warningln(err)
			} else {
				if containerdMetadata.Metadata.Config.Annotations != nil {
					if portData, ok := containerdMetadata.Metadata.Config.Annotations[kubernetes.KUBERNETES_ANNOTATION_CONTAINER_PORTS]; ok {
						containerPorts := make([]ContainerPort, 0)
						if err := json.Unmarshal([]byte(portData), &containerPorts); err != nil {
							klog.Warning(err)
						} else {
							metadata.ContainerPorts = containerPorts
						}
					}
				}
				metadata.LogPath = containerdMetadata.Metadata.LogPath
				metadata.Name = containerdMetadata.Metadata.Name
			}
		}
		return metadata, nil
	}
}
