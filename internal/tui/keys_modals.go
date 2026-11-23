package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.activeModal {
	case ModalComposeDown:
		return m.handleComposeDownModalInput(msg)
	case ModalRemoveContainer:
		return m.handleRemoveContainerModalInput(msg)
	case ModalRemoveImage:
		return m.handleRemoveImageModalInput(msg)
	case ModalRemoveNetwork:
		return m.handleRemoveNetworkModalInput(msg)
	case ModalRemoveVolume:
		return m.handleRemoveVolumeModalInput(msg)
	case ModalExec:
		return m.handleExecModalInput(msg)
	case ModalSwarmScale:
		return m.handleSwarmScaleModalInput(msg)
	case ModalPrune:
		return m.handlePruneModalInput(msg)
	case ModalLogs:
		return m.handleLogsModalInput(msg)
	case ModalInspect:
		return m.handleInspectModalInput(msg)
	default:
		return m, nil
	}
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
