package tui

import (
	"fmt"

	"github.com/AndreyJVM/container-ctl/internal/tui/modals"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.activeModal == ModalComposeDown {
		switch msg.String() {
		case "esc", "q", "n", "N":
			m.closeModal()
			m.composeDownTarget = ""
			return m, nil
		case "y", "Y":
			target := m.composeDownTarget
			m.closeModal()
			m.composeDownTarget = ""
			m.showToast(fmt.Sprintf("Downing Compose project %s (down)...", target))
			return m, downComposeCmd(target, m.dockerInfo.Containers)
		}
		return m, nil
	}

	if m.activeModal == ModalRemoveContainer {
		switch msg.String() {
		case "esc", "q", "n", "N":
			m.closeModal()
			m.removeContainerTargetID = ""
			m.removeContainerTargetName = ""
			return m, nil
		case "y", "Y":
			targetID := m.removeContainerTargetID
			targetName := m.removeContainerTargetName
			m.closeModal()
			m.removeContainerTargetID = ""
			m.removeContainerTargetName = ""
			m.showToast(fmt.Sprintf("Removing container %s...", targetName))
			return m, removeContainerCmd(targetID, targetName)
		}
		return m, nil
	}

	if m.activeModal == ModalRemoveImage {
		switch msg.String() {
		case "esc", "q", "n", "N":
			m.closeModal()
			m.removeImageTargetID = ""
			m.removeImageTargetName = ""
			m.removeImageTargetInUse = false
			return m, nil
		case "y", "Y":
			targetID := m.removeImageTargetID
			targetName := m.removeImageTargetName
			inUse := m.removeImageTargetInUse
			m.closeModal()
			m.removeImageTargetID = ""
			m.removeImageTargetName = ""
			m.removeImageTargetInUse = false
			m.showToast(fmt.Sprintf("Removing image %s...", targetName))
			return m, removeImageCmd(targetID, targetName, inUse)
		}
		return m, nil
	}

	if m.activeModal == ModalExec {
		return m.handleExecModalInput(msg)
	}

	if m.activeModal == ModalSwarmScale {
		return m.handleSwarmScaleModalInput(msg)
	}

	if m.activeModal == ModalPrune {
		switch msg.String() {
		case "q":
			m.closeModal()
			m.pruneConfirm = 0
			return m, nil
		case "esc":
			if m.pruneConfirm > 0 {
				m.pruneConfirm = 0
			} else {
				m.closeModal()
			}
			return m, nil
		case "1", "2", "3", "4":
			if m.pruneConfirm == 0 {
				switch msg.String() {
				case "1":
					m.pruneConfirm = 1
				case "2":
					m.pruneConfirm = 2
				case "3":
					m.pruneConfirm = 3
				case "4":
					m.pruneConfirm = 4
				}
			}
			return m, nil
		case "y", "Y":
			if m.pruneConfirm > 0 {
				opt := m.pruneConfirm
				m.closeModal()
				m.pruneConfirm = 0
				m.showToast("Pruning Docker resources...")
				return m, pruneCmd(opt)
			}
			return m, nil
		case "n", "N":
			if m.pruneConfirm > 0 {
				m.pruneConfirm = 0
			}
			return m, nil
		}
		return m, nil
	}

	if m.activeModal == ModalLogs && m.logsFiltering {
		switch msg.Type {
		case tea.KeyEsc:
			m.logsFiltering = false
			m.logsFilterQuery = ""
			m.logsScroll = 0
			return m, nil
		case tea.KeyEnter:
			m.logsFiltering = false
			m.logsScroll = 0
			return m, nil
		case tea.KeyBackspace:
			if len(m.logsFilterQuery) > 0 {
				runes := []rune(m.logsFilterQuery)
				m.logsFilterQuery = string(runes[:len(runes)-1])
				m.logsScroll = 0
			}
			return m, nil
		case tea.KeyRunes, tea.KeySpace:
			m.logsFilterQuery += msg.String()
			m.logsScroll = 0
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "/":
		if m.activeModal == ModalLogs {
			m.logsFiltering = true
			return m, nil
		}

	case "esc":
		if m.activeModal == ModalLogs && m.logsFilterQuery != "" {
			m.logsFilterQuery = ""
			m.logsScroll = 0
			return m, nil
		}
		m.closeModal()
		return m, nil

	case "q":
		m.closeModal()
		return m, nil

	case "up", "k":
		if m.activeModal == ModalLogs && m.logsScroll > 0 {
			m.logsScroll--
		} else if m.activeModal == ModalInspect && m.inspectScroll > 0 {
			m.inspectScroll--
		}
		return m, nil

	case "down", "j":
		if m.activeModal == ModalLogs {
			total := m.countFilteredLogs()
			if m.logsScroll < total-10 {
				m.logsScroll++
			}
		} else if m.activeModal == ModalInspect {
			total := m.countInspectLines()
			if m.inspectScroll < total-10 {
				m.inspectScroll++
			}
		}
		return m, nil

	case "pgup":
		if m.activeModal == ModalLogs {
			m.logsScroll = max(0, m.logsScroll-10)
		} else if m.activeModal == ModalInspect {
			m.inspectScroll = max(0, m.inspectScroll-10)
		}
		return m, nil

	case "pgdown":
		if m.activeModal == ModalLogs {
			total := m.countFilteredLogs()
			m.logsScroll = min(max(0, total-10), m.logsScroll+10)
		} else if m.activeModal == ModalInspect {
			total := m.countInspectLines()
			m.inspectScroll = min(max(0, total-10), m.inspectScroll+10)
		}
		return m, nil

	case "y", "Y":
		targetToCopy := m.inspectCopyID
		if targetToCopy == "" && m.inspectTarget.ID != "" {
			targetToCopy = m.inspectTarget.ID
		}
		if m.activeModal == ModalInspect && targetToCopy != "" {
			m.copyToClipboard(targetToCopy)
			return m, nil
		}
	}
	return m, nil
}

func (m Model) countInspectLines() int {
	if len(m.inspectCustomLines) > 0 {
		return len(m.inspectCustomLines)
	}
	return len(modals.BuildInspectLines(m.inspectTarget))
}

func (m Model) countFilteredLogs() int {
	return modals.CountMatchingLogs(m.logsLines, m.logsFilterQuery)
}

func (m Model) handleSwarmScaleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.closeModal()
		return m, nil

	case "+", "=", "up", "k":
		if m.swarmScaleDesired < 999 {
			m.swarmScaleDesired++
		}
		m.swarmScaleTyped = true
		return m, nil

	case "-", "_", "down", "j":
		if m.swarmScaleDesired > 0 {
			m.swarmScaleDesired--
		}
		m.swarmScaleTyped = true
		return m, nil

	case "backspace":
		m.swarmScaleDesired /= 10
		m.swarmScaleTyped = true
		return m, nil

	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		digit := int(msg.String()[0] - '0')
		if !m.swarmScaleTyped {
			m.swarmScaleDesired = digit
			m.swarmScaleTyped = true
		} else {
			if m.swarmScaleDesired < 100 {
				m.swarmScaleDesired = m.swarmScaleDesired*10 + digit
			}
		}
		return m, nil

	case "enter":
		targetID := m.swarmScaleTargetID
		targetName := m.swarmScaleTargetName
		desired := m.swarmScaleDesired
		m.closeModal()
		m.showToast(fmt.Sprintf("Scaling service %s to %d replicas...", targetName, desired))
		return m, scaleSwarmServiceCmd(targetID, targetName, desired)
	}
	return m, nil
}

func (m Model) handleHelpInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "?":
		m.showHelp = false
		return m, nil
	}
	return m, nil
}
