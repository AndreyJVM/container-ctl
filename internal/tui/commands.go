package tui

import (
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

// TimeFormat is the standard format used for dashboard timestamps.
const TimeFormat = actions.TimeFormat

// Prune options constants.
const (
	PruneOptionContainers = actions.PruneOptionContainers
	PruneOptionImages     = actions.PruneOptionImages
	PruneOptionVolumes    = actions.PruneOptionVolumes
	PruneOptionAll        = actions.PruneOptionAll
)

func fetchAllData() dataFetchedMsg {
	return actions.FetchAllData()
}

func fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		return fetchAllData()
	}
}

func wrapAction(cmd tea.Cmd) tea.Cmd {
	return cmd
}

func makeActionCmd(action func() error, errPrefix, successMsg string) tea.Cmd {
	return actions.MakeActionCmd(action, errPrefix, successMsg)
}

func startContainerCmd(id, name string) tea.Cmd {
	return wrapAction(actions.StartContainer(id, name))
}

func stopContainerCmd(id, name string) tea.Cmd {
	return wrapAction(actions.StopContainer(id, name))
}

func restartContainerCmd(id, name string) tea.Cmd {
	return wrapAction(actions.RestartContainer(id, name))
}

func killContainerCmd(id, name string) tea.Cmd {
	return wrapAction(actions.KillContainer(id, name))
}

func removeContainerCmd(id, name string) tea.Cmd {
	return wrapAction(actions.RemoveContainer(id, name))
}

func batchStartContainersCmd(items []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.BatchStartContainers(items))
}

func batchStopContainersCmd(items []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.BatchStopContainers(items))
}

func batchRestartContainersCmd(items []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.BatchRestartContainers(items))
}

func batchKillContainersCmd(items []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.BatchKillContainers(items))
}

func batchRemoveContainersCmd(items []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.BatchRemoveContainers(items))
}

func removeImageCmd(id, name string, force bool) tea.Cmd {
	return wrapAction(actions.RemoveImage(id, name, force))
}

func removeNetworkCmd(id, name string) tea.Cmd {
	return wrapAction(actions.RemoveNetwork(id, name))
}

func removeVolumeCmd(name string, force bool) tea.Cmd {
	return wrapAction(actions.RemoveVolume(name, force))
}

func startComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.StartCompose(name, containers))
}

func stopComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.StopCompose(name, containers))
}

func restartComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.RestartCompose(name, containers))
}

func downComposeCmd(name string, containers []monitor.ContainerItem) tea.Cmd {
	return wrapAction(actions.DownCompose(name, containers))
}

func scaleSwarmServiceCmd(id, name string, replicas int) tea.Cmd {
	return wrapAction(actions.ScaleSwarmService(id, name, replicas))
}

func executePrune(option int) (string, error) {
	return actions.ExecutePrune(option)
}

func pruneCmd(option int) tea.Cmd {
	return wrapAction(actions.Prune(option))
}
