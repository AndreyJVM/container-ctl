// Package monitor is a thin facade that re-exports public types, constants,
// and functions from the docker, compose, swarm, and prune sub-packages.
// It allows callers (such as internal/tui/) to import a unified package
// or migrate gradually to individual sub-packages.
package monitor

import (
	"os/exec"

	"github.com/AndreyJVM/container-ctl/internal/monitor/compose"
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker"
	"github.com/AndreyJVM/container-ctl/internal/monitor/prune"
	"github.com/AndreyJVM/container-ctl/internal/monitor/swarm"
)

// ---------------------------------------------------------------------------
// docker types (type aliases)
// ---------------------------------------------------------------------------

type ContainerItem = docker.ContainerItem
type ContainerDetail = docker.ContainerDetail
type DockerSummary = docker.DockerSummary
type ContainerStatsRaw = docker.ContainerStatsRaw
type ContainerStats = docker.ContainerStats
type DockerDiskItem = docker.DockerDiskItem
type DockerDiskUsage = docker.DockerDiskUsage
type DockerImage = docker.DockerImage
type DockerNetwork = docker.DockerNetwork
type DockerVolume = docker.DockerVolume
type ExecOptions = docker.ExecOptions

// ---------------------------------------------------------------------------
// compose types (type aliases)
// ---------------------------------------------------------------------------

type ComposeService = compose.ComposeService
type ComposeProject = compose.ComposeProject

// ---------------------------------------------------------------------------
// swarm types (type aliases)
// ---------------------------------------------------------------------------

type SwarmNode = swarm.SwarmNode
type SwarmService = swarm.SwarmService
type SwarmSummary = swarm.SwarmSummary

// ---------------------------------------------------------------------------
// constants re-exported from sub-packages
// ---------------------------------------------------------------------------

const (
	// docker socket timeouts
	SocketDialTimeout   = docker.SocketDialTimeout
	SocketClientTimeout = docker.SocketClientTimeout

	// docker status constants
	DockerStatusRunning      = docker.DockerStatusRunning
	DockerStatusStopped      = docker.DockerStatusStopped
	DockerStatusNotInstalled = docker.DockerStatusNotInstalled

	DockerStatusDisplayRunning      = docker.DockerStatusDisplayRunning
	DockerStatusDisplayStopped      = docker.DockerStatusDisplayStopped
	DockerStatusDisplayNotInstalled = docker.DockerStatusDisplayNotInstalled

	// health status constants
	HealthStatusHealthy   = docker.HealthStatusHealthy
	HealthStatusUnhealthy = docker.HealthStatusUnhealthy
	HealthStatusStarting  = docker.HealthStatusStarting

	// compose statuses
	StatusRunning = compose.StatusRunning
	StatusPartial = compose.StatusPartial
	StatusStopped = compose.StatusStopped

	StatusDisplayRunning = compose.StatusDisplayRunning
	StatusDisplayPartial = compose.StatusDisplayPartial
	StatusDisplayStopped = compose.StatusDisplayStopped

	ActionRestart = compose.ActionRestart
	ActionStop    = compose.ActionStop
	ActionStart   = compose.ActionStart
	ActionDown    = compose.ActionDown
)

// ---------------------------------------------------------------------------
// docker functions
// ---------------------------------------------------------------------------

// GetDockerSummary collects Docker daemon status and container list with live metrics.
func GetDockerSummary() DockerSummary { return docker.GetDockerSummary() }

// GetContainerDetail returns detailed inspection data for a container.
func GetContainerDetail(id string) (ContainerDetail, error) {
	return docker.GetContainerDetail(id)
}

// GetContainerLogs returns the recent log output of a container.
func GetContainerLogs(id string, tail int) (string, error) {
	return docker.GetContainerLogs(id, tail)
}

// GetDockerImages returns all Docker images without changing daemon state.
func GetDockerImages() ([]DockerImage, error) { return docker.GetDockerImages() }

// RemoveDockerImage removes a Docker image by ID or name, with optional force.
func RemoveDockerImage(id string, force bool) error { return docker.RemoveDockerImage(id, force) }

// GetDockerNetworks returns Docker networks without changing daemon state.
func GetDockerNetworks() ([]DockerNetwork, error) { return docker.GetDockerNetworks() }

// GetDockerVolumes returns Docker volumes without changing daemon state.
func GetDockerVolumes() ([]DockerVolume, error) { return docker.GetDockerVolumes() }

// GetNetworkInspectJSON returns formatted JSON inspection for a Docker network.
func GetNetworkInspectJSON(idOrName string) (string, error) {
	return docker.GetNetworkInspectJSON(idOrName)
}

// GetVolumeInspectJSON returns formatted JSON inspection for a Docker volume.
func GetVolumeInspectJSON(name string) (string, error) {
	return docker.GetVolumeInspectJSON(name)
}

// GetDockerDiskUsage collects Docker disk usage statistics (system df) via socket or CLI fallback.
func GetDockerDiskUsage() (DockerDiskUsage, error) { return docker.GetDockerDiskUsage() }

// ParseHealthStatus extracts a standardized healthcheck status from container status strings.
func ParseHealthStatus(val string) string { return docker.ParseHealthStatus(val) }

// ParseExitCode extracts whether an exit code is present and its integer value from Docker status strings.
func ParseExitCode(status string) (bool, int) { return docker.ParseExitCode(status) }

// IsContainerFailed returns true and the exit code if the container is in an error state.
func IsContainerFailed(state, status string) (bool, int) {
	return docker.IsContainerFailed(state, status)
}

// FormatBytes formats bytes as a human-readable string (e.g. "1.5 MB", "2.0 GB").
func FormatBytes(b int64) string { return docker.FormatBytes(b) }

// FormatUintBytes formats uint64 bytes as a human-readable string.
func FormatUintBytes(b uint64) string { return docker.FormatUintBytes(b) }

// FindDockerSocket finds an available Docker or Podman Unix socket.
func FindDockerSocket() string { return docker.FindDockerSocket() }

// SetCustomSocketPath sets a custom Docker socket path override.
func SetCustomSocketPath(path string) { docker.SetCustomSocketPath(path) }

// ResetSocketClientsPool closes idle connections and clears the client cache.
func ResetSocketClientsPool() { docker.ResetSocketClientsPool() }

// IsDockerCLIAvailable checks availability of the docker CLI in PATH.
func IsDockerCLIAvailable() bool { return docker.IsDockerCLIAvailable() }

// GetContainerCLI returns the active CLI binary ("docker" or "podman").
func GetContainerCLI() string { return docker.GetContainerCLI() }

// IsPodmanSocket returns true if the socket is associated with Podman.
func IsPodmanSocket(sockPath string) bool { return docker.IsPodmanSocket(sockPath) }

// StartContainer starts a container by ID.
func StartContainer(id string) error { return docker.StartContainer(id) }

// StopContainer stops a container by ID.
func StopContainer(id string) error { return docker.StopContainer(id) }

// RestartContainer restarts a container by ID.
func RestartContainer(id string) error { return docker.RestartContainer(id) }

// KillContainer terminates a container by ID with SIGKILL.
func KillContainer(id string) error { return docker.KillContainer(id) }

// RemoveContainer force-removes a container by ID.
func RemoveContainer(id string) error { return docker.RemoveContainer(id) }

// GetContainerExecCommand creates an interactive exec command for a container.
func GetContainerExecCommand(id string) *exec.Cmd {
	return docker.GetContainerExecCommand(id)
}

// GetContainerExecCommandWithOptions creates an interactive exec command with options.
func GetContainerExecCommandWithOptions(id string, opts ExecOptions) *exec.Cmd {
	return docker.GetContainerExecCommandWithOptions(id, opts)
}

// SetDockerCLIAvailableCheckerForTesting overrides the docker CLI check for unit tests and returns a restore function.
func SetDockerCLIAvailableCheckerForTesting(checker func() bool) func() {
	return docker.SetDockerCLIAvailableCheckerForTesting(checker)
}

// ---------------------------------------------------------------------------
// compose functions
// ---------------------------------------------------------------------------

// GetComposeProjects groups containers into Docker Compose projects and computes aggregate statuses.
func GetComposeProjects(containers []ContainerItem) []ComposeProject {
	return compose.GetComposeProjects(containers)
}

// GetComposeProjectLogs retrieves aggregated logs for all services in a Compose project.
func GetComposeProjectLogs(projectName string, containers []ContainerItem, tail int) (string, error) {
	return compose.GetComposeProjectLogs(projectName, containers, tail)
}

// RestartComposeProject restarts all containers in a Compose project.
func RestartComposeProject(name string, containers []ContainerItem) error {
	return compose.RestartComposeProject(name, containers)
}

// StopComposeProject stops all containers in a Compose project.
func StopComposeProject(name string, containers []ContainerItem) error {
	return compose.StopComposeProject(name, containers)
}

// StartComposeProject starts all containers in a Compose project.
func StartComposeProject(name string, containers []ContainerItem) error {
	return compose.StartComposeProject(name, containers)
}

// DownComposeProject tears down a Compose project.
func DownComposeProject(name string, containers []ContainerItem) error {
	return compose.DownComposeProject(name, containers)
}

// ---------------------------------------------------------------------------
// swarm functions
// ---------------------------------------------------------------------------

// GetSwarmSummary collects information about Docker Swarm cluster state and topology.
func GetSwarmSummary() SwarmSummary { return swarm.GetSwarmSummary() }

// GetSwarmServiceLogs retrieves logs for a Swarm service by its ID or name.
func GetSwarmServiceLogs(serviceID string, tailLines int) (string, error) {
	return swarm.GetSwarmServiceLogs(serviceID, tailLines)
}

// ScaleSwarmService updates the desired replica count for a Swarm service.
func ScaleSwarmService(serviceID string, replicas int) error {
	return swarm.ScaleSwarmService(serviceID, replicas)
}

// ---------------------------------------------------------------------------
// prune functions
// ---------------------------------------------------------------------------

// PruneContainers removes all stopped containers.
func PruneContainers() (string, error) { return prune.PruneContainers() }

// PruneImages removes dangling Docker images.
func PruneImages() (string, error) { return prune.PruneImages() }

// PruneVolumes removes unused Docker volumes.
func PruneVolumes() (string, error) { return prune.PruneVolumes() }

// PruneAll performs cleanup of unused containers, images, volumes, and build cache.
func PruneAll() (string, error) { return prune.PruneAll() }
