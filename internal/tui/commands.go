package tui

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

func fetchAllData() dataFetchedMsg {
	dockerInfo := monitor.GetDockerSummary()
	images, imageErr := monitor.GetDockerImages()
	networks, networkErr := monitor.GetDockerNetworks()
	volumes, volumeErr := monitor.GetDockerVolumes()
	return dataFetchedMsg{
		dockerInfo: dockerInfo,
		images:     images,
		compose:    monitor.GetComposeProjects(dockerInfo.Containers),
		networks:   networks,
		volumes:    volumes,
		swarm:      monitor.GetSwarmSummary(),
		imageErr:   formatFetchError(imageErr),
		networkErr: formatFetchError(networkErr),
		volumeErr:  formatFetchError(volumeErr),
		timestamp:  time.Now().Format(TimeFormat),
	}
}

func fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		return fetchAllData()
	}
}

func makeActionCmd(action func() error, errPrefix, successMsg string) tea.Cmd {
	return func() tea.Msg {
		if err := action(); err != nil {
			return actionFinishedMsg{err: fmt.Errorf("%s%w", errPrefix, err)}
		}
		return actionFinishedMsg{message: successMsg, refresh: true}
	}
}

func startContainerCmd(id, name string) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.StartContainer(id) },
		"Error starting container: ",
		fmt.Sprintf("Container %s started", name),
	)
}

func stopContainerCmd(id, name string) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.StopContainer(id) },
		"Error stopping container: ",
		fmt.Sprintf("Container %s stopped", name),
	)
}

func restartContainerCmd(id, name string) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.RestartContainer(id) },
		"Error restarting container: ",
		fmt.Sprintf("Container %s restarted successfully", name),
	)
}

func killContainerCmd(id, name string) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.KillContainer(id) },
		"Error killing container: ",
		fmt.Sprintf("Container %s force-killed successfully", name),
	)
}

func removeContainerCmd(id, name string) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.RemoveContainer(id) },
		"Error removing container: ",
		fmt.Sprintf("Container %s removed successfully", name),
	)
}

func removeImageCmd(id, name string, force bool) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.RemoveDockerImage(id, force) },
		"Error removing image: ",
		fmt.Sprintf("Image %s removed successfully", name),
	)
}

func startComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.StartComposeProject(name, containers) },
		"Error starting project: ",
		fmt.Sprintf("Project %s started successfully", name),
	)
}

func stopComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.StopComposeProject(name, containers) },
		"Error stopping project: ",
		fmt.Sprintf("Project %s stopped successfully", name),
	)
}

func restartComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.RestartComposeProject(name, containers) },
		"Error restarting project: ",
		fmt.Sprintf("Project %s restarted successfully", name),
	)
}

func downComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return makeActionCmd(
		func() error { return monitor.DownComposeProject(name, containers) },
		"Error downing project: ",
		fmt.Sprintf("Project %s downed successfully", name),
	)
}

func executePrune(option int) (string, error) {
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

func pruneCmd(option int) tea.Cmd {
	return func() tea.Msg {
		res, err := executePrune(option)
		if err != nil {
			return actionFinishedMsg{err: fmt.Errorf("Prune error: %w", err)}
		}
		return actionFinishedMsg{message: res, refresh: true}
	}
}
