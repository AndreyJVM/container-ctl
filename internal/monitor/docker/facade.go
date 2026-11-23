package docker

import (
	"io"
	"os/exec"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/operations"
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/resources"
)

// --- Resources (Images, Volumes, Networks, Disk Usage) ---

// GetDockerImages returns Docker images without changing daemon state.
func GetDockerImages() ([]DockerImage, error) {
	return resources.GetDockerImages()
}

// RemoveDockerImage removes a Docker image by ID or name, with optional force.
func RemoveDockerImage(imageID string, force bool) error {
	return resources.RemoveDockerImage(imageID, force)
}

// GetDockerVolumes returns Docker volumes without changing daemon state.
func GetDockerVolumes() ([]DockerVolume, error) {
	return resources.GetDockerVolumes()
}

// GetVolumeInspectJSON returns formatted JSON inspection for a Docker volume.
func GetVolumeInspectJSON(name string) (string, error) {
	return resources.GetVolumeInspectJSON(name)
}

// RemoveDockerVolume removes a Docker volume by name.
func RemoveDockerVolume(name string, force bool) error {
	return resources.RemoveDockerVolume(name, force)
}

// MatchContainersForVolume finds container names that reference the given volume name in their mounts.
func MatchContainersForVolume(volName string, containers []ContainerItem) []string {
	return resources.MatchContainersForVolume(volName, containers)
}

// EnrichVolumesWithContainers populates Containers slice on volumes based on container mounts.
func EnrichVolumesWithContainers(volumes []DockerVolume, containers []ContainerItem) []DockerVolume {
	return resources.EnrichVolumesWithContainers(volumes, containers)
}

// GetDockerNetworks returns Docker networks without changing daemon state.
func GetDockerNetworks() ([]DockerNetwork, error) {
	return resources.GetDockerNetworks()
}

// GetNetworkInspectJSON returns formatted JSON inspection for a Docker network.
func GetNetworkInspectJSON(idOrName string) (string, error) {
	return resources.GetNetworkInspectJSON(idOrName)
}

// RemoveDockerNetwork removes a Docker network by ID or name.
func RemoveDockerNetwork(idOrName string) error {
	return resources.RemoveDockerNetwork(idOrName)
}

// GetDockerDiskUsage collects Docker disk usage statistics (system df).
func GetDockerDiskUsage() (DockerDiskUsage, error) {
	return resources.GetDockerDiskUsage()
}

// FormatBytes formats an int64 byte count into a human-readable string.
func FormatBytes(b int64) string {
	return resources.FormatBytes(b)
}

// FormatUintBytes formats a uint64 byte count into a human-readable string.
func FormatUintBytes(b uint64) string {
	return resources.FormatUintBytes(b)
}

// --- Operations (Lifecycle, Inspect, Logs, Stats, Exec) ---

// StartContainer starts a container by ID.
func StartContainer(containerID string) error {
	return operations.StartContainer(containerID)
}

// StopContainer stops a container by ID.
func StopContainer(containerID string) error {
	return operations.StopContainer(containerID)
}

// RestartContainer restarts a container by ID.
func RestartContainer(containerID string) error {
	return operations.RestartContainer(containerID)
}

// KillContainer terminates a container by ID with SIGKILL.
func KillContainer(containerID string) error {
	return operations.KillContainer(containerID)
}

// RemoveContainer force-removes a container by ID.
func RemoveContainer(containerID string) error {
	return operations.RemoveContainer(containerID)
}

// GetContainerDetail returns detailed inspection data for a container.
func GetContainerDetail(containerID string) (ContainerDetail, error) {
	return operations.GetContainerDetail(containerID)
}

// GetContainerLogs reads the latest log lines from a container.
func GetContainerLogs(containerID string, tailLines int) (string, error) {
	return operations.GetContainerLogs(containerID, tailLines)
}

// DemuxDockerLogs decodes the Docker Engine API multiplexed stream with an 8-byte frame header.
func DemuxDockerLogs(r io.Reader) string {
	return operations.DemuxDockerLogs(r)
}

// CleanLogSnippet trims whitespace and extracts the last maxLines non-empty lines from log content.
func CleanLogSnippet(raw string, maxLines int) string {
	return operations.CleanLogSnippet(raw, maxLines)
}

// GetContainerStats retrieves metrics for a single container.
func GetContainerStats(containerID string) (ContainerStats, error) {
	return operations.GetContainerStats(containerID)
}

// GetContainerExecCommandWithOptions creates an exec.Cmd based on specified ExecOptions.
func GetContainerExecCommandWithOptions(containerID string, opts ExecOptions) *exec.Cmd {
	return operations.GetContainerExecCommandWithOptions(containerID, opts)
}

// GetContainerExecCommand creates an exec.Cmd for an interactive container shell.
func GetContainerExecCommand(containerID string) *exec.Cmd {
	return operations.GetContainerExecCommand(containerID)
}
