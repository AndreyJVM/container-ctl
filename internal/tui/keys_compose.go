package tui

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleComposeLogs() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.logsTargetName = "Compose: " + proj.Name
		m.logsTargetID = proj.Name
		logs, err := monitor.GetComposeProjectLogs(proj.Name, m.dockerInfo.Containers, 150)
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

func (m Model) handleComposeStart() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.showToast(fmt.Sprintf("Starting Compose project %s...", proj.Name))
		return m, startComposeCmd(proj.Name, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleComposeStop() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.showToast(fmt.Sprintf("Stopping Compose project %s...", proj.Name))
		return m, stopComposeCmd(proj.Name, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleComposeRestart() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.showToast(fmt.Sprintf("Restarting Compose project %s...", proj.Name))
		return m, restartComposeCmd(proj.Name, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleComposeDownModal() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.composeDownTarget = proj.Name
		m.openModal(ModalComposeDown)
	}
	return m, nil
}

func (m Model) handleComposeCopy() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.copyToClipboard(proj.Name)
	}
	return m, nil
}
