package container

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/kwaisu/sense-agent/pkg/proc"
	"inet.af/netaddr"
	"k8s.io/klog/v2"
)

type DockerdClient struct {
	client *client.Client
}

func NewDockerd() (ContainerClient, error) {
	klog.Info(proc.ProcRootSubpath("/run/docker.sock"))
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix://" + proc.ProcRootSubpath("/run/docker.sock")),
	)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()
	if _, err := cli.Ping(ctx); err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(ctx)
	return &DockerdClient{client: cli}, nil
}

func (c *DockerdClient) ListContainerID() ([]string, error) {
	if c.client == nil {
		return nil, fmt.Errorf("dockerd client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()
	if contianers, err := c.client.ContainerList(ctx, types.ContainerListOptions{}); err != nil {
		return nil, err
	} else {
		contianerIDs := make([]string, 0, len(contianers))
		for _, c := range contianers {
			contianerIDs = append(contianerIDs, c.ID)
		}
		return contianerIDs, nil
	}
}
func (c *DockerdClient) GetContainerMetadata(containerID string) (*ContainerMetadata, error) {
	if c.client == nil {
		return nil, fmt.Errorf("dockerd client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT)
	defer cancel()
	if inspect, err := c.client.ContainerInspect(ctx, containerID); err != nil {
		return nil, err
	} else {
		metadata := &ContainerMetadata{
			Name:        inspect.Name,
			Labels:      inspect.Config.Labels,
			Image:       inspect.Config.Image,
			ID:          inspect.ID,
			Created:     inspect.Created,
			Volumes:     map[string]string{},
			HostListens: map[string][]netaddr.IPPort{},
			Networks:    map[string]ContainerNetwork{},
		}
		for _, mountPoint := range inspect.Mounts {
			metadata.Volumes[mountPoint.Destination] = mountPoint.Source
		}
		if inspect.LogPath != "" && inspect.HostConfig.LogConfig.Type == "json-file" {
			metadata.LogPath = inspect.LogPath
		}
		if inspect.NetworkSettings != nil {
			listens := []netaddr.IPPort{}
			containerPorts := make([]ContainerPort, 0)
			for port, bindings := range inspect.NetworkSettings.Ports {
				for _, binding := range bindings {
					if port.Proto() == "tcp" {
						if ipp, err := netaddr.ParseIPPort(binding.HostIP + ":" + binding.HostPort); err == nil {
							listens = append(listens, ipp)
						}
					}
					containerPort, portErr := strconv.ParseUint(port.Port(), 10, 32)
					hostPort, hostPortErr := strconv.ParseUint(binding.HostPort, 10, 32)
					if portErr != nil && hostPortErr != nil {
						klog.Warning(err)
					}
					containerPorts = append(containerPorts, ContainerPort{ContainerPort: uint32(containerPort), Protocol: port.Proto(), HostPort: uint32(hostPort)})
				}
			}
			metadata.ContainerPorts = containerPorts
			metadata.HostListens["docker"] = listens
		}
		for name, setting := range inspect.NetworkSettings.Networks {
			metadata.Networks[name] = ContainerNetwork{
				NetworkID: setting.NetworkID,
				Aliases:   setting.Aliases,
				IPAddress: setting.IPAddress,
			}
		}
		return metadata, nil
	}

}
