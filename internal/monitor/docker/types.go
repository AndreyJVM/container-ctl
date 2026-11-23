package docker

import (
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/types"
)

const (
	// Docker daemon statuses
	DockerStatusRunning      = types.DockerStatusRunning
	DockerStatusStopped      = types.DockerStatusStopped
	DockerStatusNotInstalled = types.DockerStatusNotInstalled

	// Localized status displays
	DockerStatusDisplayRunning      = types.DockerStatusDisplayRunning
	DockerStatusDisplayStopped      = types.DockerStatusDisplayStopped
	DockerStatusDisplayNotInstalled = types.DockerStatusDisplayNotInstalled

	// Healthcheck statuses
	HealthStatusHealthy   = types.HealthStatusHealthy
	HealthStatusUnhealthy = types.HealthStatusUnhealthy
	HealthStatusStarting  = types.HealthStatusStarting

	// Docker compose label names
	composeProjectLabel = types.ComposeProjectLabel
	composeServiceLabel = types.ComposeServiceLabel

	// Defaults, limits, and timeouts
	maxConcurrentStatsFetch = types.MaxConcurrentStatsFetch
	defaultDockerTimeout    = types.DefaultDockerTimeout
	containerTimeFormat     = types.ContainerTimeFormat
)

// Re-export type aliases for backward compatibility.
type (
	ContainerItem        = types.ContainerItem
	ContainerDetail      = types.ContainerDetail
	DockerSummary        = types.DockerSummary
	DockerDiskItem       = types.DockerDiskItem
	DockerDiskUsage      = types.DockerDiskUsage
	DockerImage          = types.DockerImage
	DockerVolume         = types.DockerVolume
	DockerNetwork        = types.DockerNetwork
	NetworkContainerInfo = types.NetworkContainerInfo
	ExecOptions          = types.ExecOptions
	ContainerStats       = types.ContainerStats
	ContainerStatsRaw    = types.ContainerStatsRaw
)
