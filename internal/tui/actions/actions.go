package actions

import (
	"fmt"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

// TimeFormat is the standard format used for dashboard timestamps.
const TimeFormat = "15:04:05"

// Prune options constants.
const (
	PruneOptionContainers = 1
	PruneOptionImages     = 2
	PruneOptionVolumes    = 3
	PruneOptionAll        = 4
)

// FinishedMsg is returned upon completion of an async container or compose operation.
type FinishedMsg struct {
	Message string
	Err     error
	Refresh bool
}

// DataFetchedMsg delivers freshly gathered metrics and container lists across all domains.
type DataFetchedMsg struct {
	DockerInfo monitor.DockerSummary
	Images     []monitor.DockerImage
	Compose    []monitor.ComposeProject
	Networks   []monitor.DockerNetwork
	Volumes    []monitor.DockerVolume
	Swarm      monitor.SwarmSummary
	ImageErr   string
	NetworkErr string
	VolumeErr  string
	Timestamp  string
}

func formatFetchError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// FetchAllData queries all Docker resources synchronously.
func FetchAllData() DataFetchedMsg {
	dockerInfo := monitor.GetDockerSummary()
	images, imageErr := monitor.GetDockerImages()
	networks, networkErr := monitor.GetDockerNetworks()
	volumes, volumeErr := monitor.GetDockerVolumes()
	return DataFetchedMsg{
		DockerInfo: dockerInfo,
		Images:     images,
		Compose:    monitor.GetComposeProjects(dockerInfo.Containers),
		Networks:   networks,
		Volumes:    volumes,
		Swarm:      monitor.GetSwarmSummary(),
		ImageErr:   formatFetchError(imageErr),
		NetworkErr: formatFetchError(networkErr),
		VolumeErr:  formatFetchError(volumeErr),
		Timestamp:  time.Now().Format(TimeFormat),
	}
}

// FetchData creates a tea.Cmd to fetch all Docker data asynchronously.
func FetchData() tea.Cmd {
	return func() tea.Msg {
		return FetchAllData()
	}
}

func MakeActionCmd(action func() error, errPrefix, successMsg string) tea.Cmd {
	return func() tea.Msg {
		if err := action(); err != nil {
			return FinishedMsg{Err: fmt.Errorf("%s%w", errPrefix, err)}
		}
		return FinishedMsg{Message: successMsg, Refresh: true}
	}
}

// StartContainer returns a command to start a container.
func StartContainer(id, name string) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.StartContainer(id) },
		"Error starting container: ",
		fmt.Sprintf("Container %s started", name),
	)
}

// StopContainer returns a command to stop a container.
func StopContainer(id, name string) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.StopContainer(id) },
		"Error stopping container: ",
		fmt.Sprintf("Container %s stopped", name),
	)
}

// RestartContainer returns a command to restart a container.
func RestartContainer(id, name string) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.RestartContainer(id) },
		"Error restarting container: ",
		fmt.Sprintf("Container %s restarted successfully", name),
	)
}

// KillContainer returns a command to force-kill a container with SIGKILL.
func KillContainer(id, name string) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.KillContainer(id) },
		"Error killing container: ",
		fmt.Sprintf("Container %s force-killed successfully", name),
	)
}

// RemoveContainer returns a command to remove a container.
func RemoveContainer(id, name string) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.RemoveContainer(id) },
		"Error removing container: ",
		fmt.Sprintf("Container %s removed successfully", name),
	)
}

// RemoveImage returns a command to remove a Docker image.
func RemoveImage(id, name string, force bool) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.RemoveDockerImage(id, force) },
		"Error removing image: ",
		fmt.Sprintf("Image %s removed successfully", name),
	)
}

// StartCompose returns a command to start a Compose project.
func StartCompose(name string, containers []monitor.ContainerItem) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.StartComposeProject(name, containers) },
		"Error starting project: ",
		fmt.Sprintf("Project %s started successfully", name),
	)
}

// StopCompose returns a command to stop a Compose project.
func StopCompose(name string, containers []monitor.ContainerItem) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.StopComposeProject(name, containers) },
		"Error stopping project: ",
		fmt.Sprintf("Project %s stopped successfully", name),
	)
}

// RestartCompose returns a command to restart a Compose project.
func RestartCompose(name string, containers []monitor.ContainerItem) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.RestartComposeProject(name, containers) },
		"Error restarting project: ",
		fmt.Sprintf("Project %s restarted successfully", name),
	)
}

// DownCompose returns a command to tear down a Compose project.
func DownCompose(name string, containers []monitor.ContainerItem) tea.Cmd {
	return MakeActionCmd(
		func() error { return monitor.DownComposeProject(name, containers) },
		"Error downing project: ",
		fmt.Sprintf("Project %s downed successfully", name),
	)
}

// ExecutePrune runs the specified prune operation.
func ExecutePrune(option int) (string, error) {
	switch option {
	case PruneOptionContainers:
		return monitor.PruneContainers()
	case PruneOptionImages:
		return monitor.PruneImages()
	case PruneOptionVolumes:
		return monitor.PruneVolumes()
	case PruneOptionAll:
		return monitor.PruneAll()
	default:
		return "", fmt.Errorf("unknown prune option: %d", option)
	}
}

// Prune returns a command to prune unused Docker resources.
func Prune(option int) tea.Cmd {
	return func() tea.Msg {
		res, err := ExecutePrune(option)
		if err != nil {
			return FinishedMsg{Err: fmt.Errorf("Prune error: %w", err)}
		}
		return FinishedMsg{Message: res, Refresh: true}
	}
}
