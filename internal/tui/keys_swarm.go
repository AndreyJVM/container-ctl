package tui

import (
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleSwarmLogs() (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
		svc := m.swarm.Services[m.swarmCursorIdx]
		m.logsTargetName = "Swarm: " + svc.Name
		m.logsTargetID = svc.ID
		logs, err := monitor.GetSwarmServiceLogs(svc.ID, 150)
		if err != nil {
			m.showToast("Logs error: " + err.Error())
		} else {
			m.logsLines = strings.Split(strings.TrimSpace(logs), "\n")
			m.logsScroll = max(0, len(m.logsLines)-18)
			m.openModal(ModalLogs)
		}
	}
	return m, nil
}

func (m Model) handleSwarmCopy() (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
		svc := m.swarm.Services[m.swarmCursorIdx]
		m.copyToClipboard(svc.Name)
	}
	return m, nil
}

func (m Model) handleSwarmScaleModal() (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
		svc := m.swarm.Services[m.swarmCursorIdx]
		if strings.EqualFold(svc.Mode, "global") {
			m.showToast("Cannot scale global Swarm service")
			return m, nil
		}
		m.swarmScaleTargetID = svc.ID
		m.swarmScaleTargetName = svc.Name
		m.swarmScaleCurrent = svc.TargetReplicas
		m.swarmScaleDesired = svc.TargetReplicas
		m.openModal(ModalSwarmScale)
	}
	return m, nil
}

func (m Model) handleSwarmQuickScale(delta int) (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
		svc := m.swarm.Services[m.swarmCursorIdx]
		if strings.EqualFold(svc.Mode, "global") {
			m.showToast("Cannot scale global Swarm service")
			return m, nil
		}
		desired := svc.TargetReplicas + delta
		if desired < 0 {
			desired = 0
		}
		return m, scaleSwarmServiceCmd(svc.ID, svc.Name, desired)
	}
	return m, nil
}
