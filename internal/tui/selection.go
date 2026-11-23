package tui

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) isContainerSelected(id string) bool {
	if m.selectedContainers == nil {
		return false
	}
	_, ok := m.selectedContainers[id]
	return ok
}

func (m *Model) toggleContainerSelection() {
	if m.selectedContainers == nil {
		m.selectedContainers = make(map[string]struct{})
	}
	containers := m.getFilteredContainers()
	if len(containers) == 0 || m.cursorIdx >= len(containers) {
		return
	}
	c := containers[m.cursorIdx]
	if _, ok := m.selectedContainers[c.ID]; ok {
		delete(m.selectedContainers, c.ID)
	} else {
		m.selectedContainers[c.ID] = struct{}{}
	}
	if m.cursorIdx < len(containers)-1 {
		m.moveCursor(1)
	}
}

func (m *Model) toggleSelectAllContainers() {
	if m.selectedContainers == nil {
		m.selectedContainers = make(map[string]struct{})
	}
	containers := m.getFilteredContainers()
	if len(containers) == 0 {
		return
	}
	allSelected := true
	for _, c := range containers {
		if _, ok := m.selectedContainers[c.ID]; !ok {
			allSelected = false
			break
		}
	}

	if allSelected {
		for _, c := range containers {
			delete(m.selectedContainers, c.ID)
		}
		m.showToast("All visible containers deselected")
	} else {
		for _, c := range containers {
			m.selectedContainers[c.ID] = struct{}{}
		}
		m.showToast(fmt.Sprintf("Selected %d container(s)", len(containers)))
	}
}

func (m *Model) clearSelectedContainers() {
	m.selectedContainers = make(map[string]struct{})
}

func (m Model) getSelectedContainers() []monitor.ContainerItem {
	if len(m.selectedContainers) == 0 {
		return nil
	}
	var selected []monitor.ContainerItem
	for _, c := range m.dockerInfo.Containers {
		if _, ok := m.selectedContainers[c.ID]; ok {
			selected = append(selected, c)
		}
	}
	return selected
}

func (m Model) handleDockerBatchStartStop() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}

	allRunning := true
	for _, c := range selected {
		if c.State != "running" {
			allRunning = false
			break
		}
	}

	if allRunning {
		m.showToast(fmt.Sprintf("Stopping %d selected container(s)...", len(selected)))
		return m, batchStopContainersCmd(selected)
	}

	m.showToast(fmt.Sprintf("Starting %d selected container(s)...", len(selected)))
	return m, batchStartContainersCmd(selected)
}

func (m Model) handleDockerBatchRestart() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}
	m.showToast(fmt.Sprintf("Restarting %d selected container(s)...", len(selected)))
	return m, batchRestartContainersCmd(selected)
}

func (m Model) handleDockerBatchKill() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}
	m.showToast(fmt.Sprintf("Force killing %d selected container(s)...", len(selected)))
	return m, batchKillContainersCmd(selected)
}

func (m Model) handleDockerBatchRemoveModal() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}
	m.removeContainerBatch = selected
	var names []string
	for _, c := range selected {
		names = append(names, c.Name)
	}
	m.removeContainerTargetName = strings.Join(names, ", ")
	m.removeContainerTargetID = fmt.Sprintf("%d containers", len(selected))
	m.openModal(ModalRemoveContainer)
	return m, nil
}
