package tui

import (
	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func (m Model) formatDockerStatus(status string) string {
	switch status {
	case monitor.DockerStatusRunning:
		return "Running"
	case monitor.DockerStatusStopped:
		return "Stopped"
	case monitor.DockerStatusNotInstalled:
		return "Not Installed"
	default:
		switch m.dockerInfo.StatusDisplay {
		case monitor.DockerStatusDisplayRunning, "running":
			return "Running"
		case monitor.DockerStatusDisplayStopped, "stopped":
			return "Stopped"
		case monitor.DockerStatusDisplayNotInstalled, "not_installed":
			return "Not Installed"
		default:
			if m.dockerInfo.StatusDisplay != "" {
				return m.dockerInfo.StatusDisplay
			}
			if status == "" {
				return "Stopped"
			}
			return status
		}
	}
}

func (m Model) formatComposeStatus(status, statusDisplay string) string {
	switch status {
	case monitor.StatusRunning:
		return "Running"
	case monitor.StatusPartial:
		return "Partial"
	case monitor.StatusStopped:
		return "Stopped"
	default:
		switch statusDisplay {
		case monitor.StatusDisplayRunning, "running":
			return "Running"
		case monitor.StatusDisplayPartial, "partial":
			return "Partial"
		case monitor.StatusDisplayStopped, "stopped":
			return "Stopped"
		default:
			if statusDisplay != "" {
				return statusDisplay
			}
			if status == "" {
				return "Stopped"
			}
			return status
		}
	}
}

func (m Model) formatSwarmStatus(isActive bool) string {
	if isActive {
		return "Active"
	}
	return "Inactive"
}
