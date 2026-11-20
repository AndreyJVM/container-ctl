package testutil

import (
	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

// ContainerOption defines a functional option to customize a test ContainerItem.
type ContainerOption func(*monitor.ContainerItem)

// AContainer creates a monitor.ContainerItem populated with sensible test defaults.
// Any aspect can be customized by passing ContainerOption functions.
func AContainer(opts ...ContainerOption) monitor.ContainerItem {
	c := monitor.ContainerItem{
		ID:            "c123456789ab",
		Name:          "test-container",
		Image:         "nginx:alpine",
		State:         "running",
		Status:        "Up 2 hours",
		Health:        "healthy",
		Ports:         "0.0.0.0:80->80/tcp",
		IPAddress:     "172.17.0.2",
		NetworkName:   "bridge",
		RestartPolicy: "always",
		Created:       "2026-11-17 10:00:00",
		CPUPerc:       5.0,
		MemUsageMB:    64.0,
		MemLimitMB:    512.0,
		MemPerc:       12.5,
		NetRxBytes:    1024 * 1024,
		NetTxBytes:    2048 * 1024,
	}

	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// WithID overrides the container ID.
func WithID(id string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.ID = id
	}
}

// WithName overrides the container name.
func WithName(name string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.Name = name
	}
}

// WithImage overrides the container image.
func WithImage(image string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.Image = image
	}
}

// WithState overrides the container lifecycle state and status text.
func WithState(state, status string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.State = state
		c.Status = status
	}
}

// WithHealth overrides the container health status.
func WithHealth(health string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.Health = health
	}
}

// WithMetrics overrides CPU and Memory usage values.
func WithMetrics(cpuPerc, memUsageMB, memLimitMB float64) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.CPUPerc = cpuPerc
		c.MemUsageMB = memUsageMB
		c.MemLimitMB = memLimitMB
		if memLimitMB > 0 {
			c.MemPerc = (memUsageMB / memLimitMB) * 100.0
		} else {
			c.MemPerc = 0
		}
	}
}

// WithNetworkIO overrides network RX and TX bytes.
func WithNetworkIO(rx, tx uint64) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.NetRxBytes = rx
		c.NetTxBytes = tx
	}
}

// WithBlockIO overrides block read and write bytes.
func WithBlockIO(read, write uint64) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.BlockReadBytes = read
		c.BlockWriteBytes = write
	}
}

// WithCompose overrides compose project and service names.
func WithCompose(project, service string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.ComposeProject = project
		c.ComposeService = service
	}
}

// WithPorts overrides container exposed ports.
func WithPorts(ports string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.Ports = ports
	}
}

// WithMounts overrides container filesystem mounts.
func WithMounts(mounts ...string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.Mounts = mounts
	}
}

// WithNetwork overrides container IP address and network name.
func WithNetwork(ipAddress, networkName string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.IPAddress = ipAddress
		c.NetworkName = networkName
	}
}

// WithRestartPolicy overrides container restart policy.
func WithRestartPolicy(policy string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.RestartPolicy = policy
	}
}

// WithCreated overrides container created timestamp.
func WithCreated(created string) ContainerOption {
	return func(c *monitor.ContainerItem) {
		c.Created = created
	}
}

// ImageOption defines a functional option to customize a test DockerImage.
type ImageOption func(*monitor.DockerImage)

// AnImage creates a monitor.DockerImage populated with sensible test defaults.
func AnImage(opts ...ImageOption) monitor.DockerImage {
	img := monitor.DockerImage{
		ID:         "img123456789",
		FullID:     "sha256:img123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Repository: "test-repo",
		Tag:        "latest",
		AllTags:    []string{"test-repo:latest"},
		Size:       128 * 1024 * 1024,
		Created:    "2026-11-17 09:00",
		Containers: 1,
		Digest:     "sha256:feedbeef",
		InUse:      true,
		Dangling:   false,
	}

	for _, opt := range opts {
		opt(&img)
	}
	return img
}

// WithImageID overrides the image ID and FullID.
func WithImageID(id, fullID string) ImageOption {
	return func(img *monitor.DockerImage) {
		img.ID = id
		img.FullID = fullID
	}
}

// WithRepoTag overrides repository and tag.
func WithRepoTag(repo, tag string) ImageOption {
	return func(img *monitor.DockerImage) {
		img.Repository = repo
		img.Tag = tag
		if repo != "<none>" && tag != "<none>" {
			img.AllTags = []string{repo + ":" + tag}
		}
	}
}

// WithAllTags overrides the list of all tags for an image.
func WithAllTags(tags ...string) ImageOption {
	return func(img *monitor.DockerImage) {
		img.AllTags = tags
	}
}

// WithImageStatus overrides in-use, dangling flags and container reference count.
func WithImageStatus(inUse, dangling bool, containers int) ImageOption {
	return func(img *monitor.DockerImage) {
		img.InUse = inUse
		img.Dangling = dangling
		img.Containers = containers
	}
}

// NetworkOption defines a functional option to customize a test DockerNetwork.
type NetworkOption func(*monitor.DockerNetwork)

// ANetwork creates a monitor.DockerNetwork populated with sensible test defaults.
func ANetwork(opts ...NetworkOption) monitor.DockerNetwork {
	net := monitor.DockerNetwork{
		ID:         "net123456789",
		Name:       "test-net",
		Driver:     "bridge",
		Scope:      "local",
		Containers: 1,
		Subnet:     "172.28.0.0/16",
		Gateway:    "172.28.0.1",
		Attachable: true,
	}

	for _, opt := range opts {
		opt(&net)
	}
	return net
}

// VolumeOption defines a functional option to customize a test DockerVolume.
type VolumeOption func(*monitor.DockerVolume)

// AVolume creates a monitor.DockerVolume populated with sensible test defaults.
func AVolume(opts ...VolumeOption) monitor.DockerVolume {
	vol := monitor.DockerVolume{
		Name:       "test-volume",
		Driver:     "local",
		Scope:      "local",
		Mountpoint: "/var/lib/docker/volumes/test-volume/_data",
		Size:       10 * 1024 * 1024,
		RefCount:   1,
	}

	for _, opt := range opts {
		opt(&vol)
	}
	return vol
}
