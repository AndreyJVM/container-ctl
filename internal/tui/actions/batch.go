package actions

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	startContainerFunc   = monitor.StartContainer
	stopContainerFunc    = monitor.StopContainer
	restartContainerFunc = monitor.RestartContainer
	killContainerFunc    = monitor.KillContainer
	removeContainerFunc  = monitor.RemoveContainer
)

// BatchStartContainers starts all specified containers that are not currently running.
func BatchStartContainers(items []monitor.ContainerItem) tea.Cmd {
	return func() tea.Msg {
		if len(items) == 0 {
			return FinishedMsg{Message: "No containers selected"}
		}
		var errs []string
		started := 0
		for _, c := range items {
			if c.State != "running" {
				if err := startContainerFunc(c.ID); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", c.Name, err))
				} else {
					started++
				}
			}
		}
		if len(errs) > 0 {
			return FinishedMsg{
				Message: fmt.Sprintf("Started %d container(s), %d failed", started, len(errs)),
				Err:     fmt.Errorf("start errors: %s", strings.Join(errs, "; ")),
				Refresh: true,
			}
		}
		return FinishedMsg{
			Message: fmt.Sprintf("Started %d container(s) successfully", started),
			Refresh: true,
		}
	}
}

// BatchStopContainers stops all specified containers that are currently running.
func BatchStopContainers(items []monitor.ContainerItem) tea.Cmd {
	return func() tea.Msg {
		if len(items) == 0 {
			return FinishedMsg{Message: "No containers selected"}
		}
		var errs []string
		stopped := 0
		for _, c := range items {
			if c.State == "running" {
				if err := stopContainerFunc(c.ID); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", c.Name, err))
				} else {
					stopped++
				}
			}
		}
		if len(errs) > 0 {
			return FinishedMsg{
				Message: fmt.Sprintf("Stopped %d container(s), %d failed", stopped, len(errs)),
				Err:     fmt.Errorf("stop errors: %s", strings.Join(errs, "; ")),
				Refresh: true,
			}
		}
		return FinishedMsg{
			Message: fmt.Sprintf("Stopped %d container(s) successfully", stopped),
			Refresh: true,
		}
	}
}

// BatchRestartContainers restarts all specified containers.
func BatchRestartContainers(items []monitor.ContainerItem) tea.Cmd {
	return func() tea.Msg {
		if len(items) == 0 {
			return FinishedMsg{Message: "No containers selected"}
		}
		var errs []string
		restarted := 0
		for _, c := range items {
			if err := restartContainerFunc(c.ID); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", c.Name, err))
			} else {
				restarted++
			}
		}
		if len(errs) > 0 {
			return FinishedMsg{
				Message: fmt.Sprintf("Restarted %d container(s), %d failed", restarted, len(errs)),
				Err:     fmt.Errorf("restart errors: %s", strings.Join(errs, "; ")),
				Refresh: true,
			}
		}
		return FinishedMsg{
			Message: fmt.Sprintf("Restarted %d container(s) successfully", restarted),
			Refresh: true,
		}
	}
}

// BatchKillContainers force-kills all specified running containers with SIGKILL.
func BatchKillContainers(items []monitor.ContainerItem) tea.Cmd {
	return func() tea.Msg {
		if len(items) == 0 {
			return FinishedMsg{Message: "No containers selected"}
		}
		var errs []string
		killed := 0
		for _, c := range items {
			if c.State == "running" {
				if err := killContainerFunc(c.ID); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", c.Name, err))
				} else {
					killed++
				}
			}
		}
		if len(errs) > 0 {
			return FinishedMsg{
				Message: fmt.Sprintf("Force-killed %d container(s), %d failed", killed, len(errs)),
				Err:     fmt.Errorf("kill errors: %s", strings.Join(errs, "; ")),
				Refresh: true,
			}
		}
		return FinishedMsg{
			Message: fmt.Sprintf("Force-killed %d container(s) successfully", killed),
			Refresh: true,
		}
	}
}

// BatchRemoveContainers removes all specified containers.
func BatchRemoveContainers(items []monitor.ContainerItem) tea.Cmd {
	return func() tea.Msg {
		if len(items) == 0 {
			return FinishedMsg{Message: "No containers selected"}
		}
		var errs []string
		removed := 0
		for _, c := range items {
			if err := removeContainerFunc(c.ID); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", c.Name, err))
			} else {
				removed++
			}
		}
		if len(errs) > 0 {
			return FinishedMsg{
				Message: fmt.Sprintf("Removed %d container(s), %d failed", removed, len(errs)),
				Err:     fmt.Errorf("remove errors: %s", strings.Join(errs, "; ")),
				Refresh: true,
			}
		}
		return FinishedMsg{
			Message: fmt.Sprintf("Removed %d container(s) successfully", removed),
			Refresh: true,
		}
	}
}
