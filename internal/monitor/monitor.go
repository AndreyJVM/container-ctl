// Package monitor is a thin facade that re-exports all public symbols from the
// docker, compose, swarm, and prune sub-packages so that callers in
// internal/tui/ can keep importing this package unchanged.
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
// docker constants re-exported as vars (constants cannot be aliased across pkgs)
// ---------------------------------------------------------------------------

const (
	DockerStatusRunning      = docker.DockerStatusRunning
	DockerStatusStopped      = docker.DockerStatusStopped
	DockerStatusNotInstalled = docker.DockerStatusNotInstalled

	DockerStatusDisplayRunning      = docker.DockerStatusDisplayRunning
	DockerStatusDisplayStopped      = docker.DockerStatusDisplayStopped
	DockerStatusDisplayNotInstalled = docker.DockerStatusDisplayNotInstalled

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

func GetDockerSummary() DockerSummary { return docker.GetDockerSummary() }
func GetContainerDetail(id string) (ContainerDetail, error) {
	return docker.GetContainerDetail(id)
}
func GetContainerLogs(id string, tail int) (string, error) {
	return docker.GetContainerLogs(id, tail)
}
func GetDockerImages() ([]DockerImage, error)       { return docker.GetDockerImages() }
func RemoveDockerImage(id string, force bool) error { return docker.RemoveDockerImage(id, force) }
func GetDockerNetworks() ([]DockerNetwork, error)   { return docker.GetDockerNetworks() }
func GetDockerVolumes() ([]DockerVolume, error)     { return docker.GetDockerVolumes() }
func GetNetworkInspectJSON(idOrName string) (string, error) {
	return docker.GetNetworkInspectJSON(idOrName)
}
func GetVolumeInspectJSON(name string) (string, error) {
	return docker.GetVolumeInspectJSON(name)
}
func GetDockerDiskUsage() (DockerDiskUsage, error) { return docker.GetDockerDiskUsage() }
func ParseHealthStatus(val string) string          { return docker.ParseHealthStatus(val) }
func ParseExitCode(status string) (bool, int)      { return docker.ParseExitCode(status) }
func IsContainerFailed(state, status string) (bool, int) {
	return docker.IsContainerFailed(state, status)
}
func FormatBytes(b int64) string       { return docker.FormatBytes(b) }
func FormatUintBytes(b uint64) string  { return docker.FormatUintBytes(b) }
func FindDockerSocket() string         { return docker.FindDockerSocket() }
func IsDockerCLIAvailable() bool       { return docker.IsDockerCLIAvailable() }
func StartContainer(id string) error   { return docker.StartContainer(id) }
func StopContainer(id string) error    { return docker.StopContainer(id) }
func RestartContainer(id string) error { return docker.RestartContainer(id) }
func KillContainer(id string) error    { return docker.KillContainer(id) }
func RemoveContainer(id string) error  { return docker.RemoveContainer(id) }

func GetContainerExecCommand(id string) *exec.Cmd {
	return docker.GetContainerExecCommand(id)
}
func GetContainerExecCommandWithOptions(id string, opts ExecOptions) *exec.Cmd {
	return docker.GetContainerExecCommandWithOptions(id, opts)
}
func SetDockerCLIAvailableCheckerForTesting(checker func() bool) func() {
	return docker.SetDockerCLIAvailableCheckerForTesting(checker)
}

// ---------------------------------------------------------------------------
// compose functions
// ---------------------------------------------------------------------------

func GetComposeProjects(containers []ContainerItem) []ComposeProject {
	return compose.GetComposeProjects(containers)
}
func GetComposeProjectLogs(projectName string, containers []ContainerItem, tail int) (string, error) {
	return compose.GetComposeProjectLogs(projectName, containers, tail)
}
func RestartComposeProject(name string, containers []ContainerItem) error {
	return compose.RestartComposeProject(name, containers)
}
func StopComposeProject(name string, containers []ContainerItem) error {
	return compose.StopComposeProject(name, containers)
}
func StartComposeProject(name string, containers []ContainerItem) error {
	return compose.StartComposeProject(name, containers)
}
func DownComposeProject(name string, containers []ContainerItem) error {
	return compose.DownComposeProject(name, containers)
}

// ---------------------------------------------------------------------------
// swarm functions
// ---------------------------------------------------------------------------

func GetSwarmSummary() SwarmSummary { return swarm.GetSwarmSummary() }
func GetSwarmServiceLogs(serviceID string, tailLines int) (string, error) {
	return swarm.GetSwarmServiceLogs(serviceID, tailLines)
}

// ---------------------------------------------------------------------------
// prune functions
// ---------------------------------------------------------------------------

func PruneContainers() (string, error) { return prune.PruneContainers() }
func PruneImages() (string, error)     { return prune.PruneImages() }
func PruneVolumes() (string, error)    { return prune.PruneVolumes() }
func PruneAll() (string, error)        { return prune.PruneAll() }
