package tui

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/modals"
	tea "github.com/charmbracelet/bubbletea"
)

// -----------------------------------------------------------------------------
// Top-Level Modal Input Router
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Confirmation Modal Helpers & Handlers
// -----------------------------------------------------------------------------

func isModalCancelKey(key string) bool {
	switch key {
	case "esc", "q", "n", "N":
		return true
	default:
		return false
	}
}

func isModalConfirmKey(key string) bool {
	switch key {
	case "y", "Y":
		return true
	default:
		return false
	}
}

func (m Model) handleComposeDownModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.composeDownTarget = ""
		return m, nil
	}
	if isModalConfirmKey(key) {
		target := m.composeDownTarget
		m.closeModal()
		m.composeDownTarget = ""
		m.showToast(fmt.Sprintf("Downing Compose project %s (down)...", target))
		return m, downComposeCmd(target, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleRemoveContainerModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeContainerTargetID = ""
		m.removeContainerTargetName = ""
		m.removeContainerBatch = nil
		return m, nil
	}
	if isModalConfirmKey(key) {
		if len(m.removeContainerBatch) > 0 {
			batch := m.removeContainerBatch
			m.closeModal()
			m.removeContainerTargetID = ""
			m.removeContainerTargetName = ""
			m.removeContainerBatch = nil
			m.clearSelectedContainers()
			m.showToast(fmt.Sprintf("Removing %d selected container(s)...", len(batch)))
			return m, batchRemoveContainersCmd(batch)
		}
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

func (m Model) handleRemoveImageModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeImageTargetID = ""
		m.removeImageTargetName = ""
		m.removeImageTargetInUse = false
		return m, nil
	}
	if isModalConfirmKey(key) {
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

func (m Model) handleRemoveNetworkModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeNetworkTargetID = ""
		m.removeNetworkTargetName = ""
		m.removeNetworkContainersCount = 0
		return m, nil
	}
	if isModalConfirmKey(key) {
		targetID := m.removeNetworkTargetID
		targetName := m.removeNetworkTargetName
		m.closeModal()
		m.removeNetworkTargetID = ""
		m.removeNetworkTargetName = ""
		m.removeNetworkContainersCount = 0
		m.showToast(fmt.Sprintf("Removing network %s...", targetName))
		return m, removeNetworkCmd(targetID, targetName)
	}
	return m, nil
}

func (m Model) handleRemoveVolumeModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeVolumeTargetName = ""
		m.removeVolumeTargetInUse = false
		return m, nil
	}
	if isModalConfirmKey(key) {
		targetName := m.removeVolumeTargetName
		inUse := m.removeVolumeTargetInUse
		m.closeModal()
		m.removeVolumeTargetName = ""
		m.removeVolumeTargetInUse = false
		m.showToast(fmt.Sprintf("Removing volume %s...", targetName))
		return m, removeVolumeCmd(targetName, inUse)
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Exec Modal Handlers
// -----------------------------------------------------------------------------

func (m Model) launchExecProcess(opts monitor.ExecOptions) (tea.Model, tea.Cmd) {
	targetID := m.execTargetID
	targetName := m.execTargetName
	m.closeModal()
	m.clearExecTarget()

	cmd := monitor.GetContainerExecCommandWithOptions(targetID, opts)
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execFinishedMsg{
			containerName: targetName,
			err:           err,
		}
	})
}

func (m Model) handleExecModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.execIsCustomInput {
		switch msg.String() {
		case "esc":
			m.execIsCustomInput = false
			return m, nil
		case "enter":
			if strings.TrimSpace(m.execCustomCmd) == "" {
				m.execIsCustomInput = false
				return m, nil
			}
			return m.launchExecProcess(monitor.ExecOptions{
				Mode:      "custom",
				CustomCmd: m.execCustomCmd,
			})
		case "backspace":
			if len(m.execCustomCmd) > 0 {
				run := []rune(m.execCustomCmd)
				m.execCustomCmd = string(run[:len(run)-1])
			}
			return m, nil
		default:
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				m.execCustomCmd += msg.String()
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "esc", "q":
		m.closeModal()
		m.execTargetID = ""
		m.execTargetName = ""
		return m, nil
	case "up", "k":
		if m.execCursorIdx > 0 {
			m.execCursorIdx--
		} else {
			m.execCursorIdx = 6
		}
		return m, nil
	case "down", "j":
		if m.execCursorIdx < 6 {
			m.execCursorIdx++
		} else {
			m.execCursorIdx = 0
		}
		return m, nil
	case "1", "b", "B":
		return m.launchEmbeddedShell()
	case "2", "s", "S":
		return m.launchExecProcess(monitor.ExecOptions{Mode: "sh"})
	case "3", "z", "Z":
		return m.launchExecProcess(monitor.ExecOptions{Mode: "zsh"})
	case "4", "a", "A":
		return m.launchExecProcess(monitor.ExecOptions{Mode: "attach"})
	case "5", "r", "R":
		return m.launchExecProcess(monitor.ExecOptions{Mode: "root"})
	case "6", "c", "C":
		m.execIsCustomInput = true
		m.execCustomCmd = ""
		return m, nil
	case "7":
		return m.launchExecProcess(monitor.ExecOptions{Mode: "auto"})
	case "enter":
		switch m.execCursorIdx {
		case 0:
			return m.launchEmbeddedShell()
		case 1:
			return m.launchExecProcess(monitor.ExecOptions{Mode: "sh"})
		case 2:
			return m.launchExecProcess(monitor.ExecOptions{Mode: "zsh"})
		case 3:
			return m.launchExecProcess(monitor.ExecOptions{Mode: "attach"})
		case 4:
			return m.launchExecProcess(monitor.ExecOptions{Mode: "root"})
		case 5:
			m.execIsCustomInput = true
			m.execCustomCmd = ""
			return m, nil
		case 6:
			return m.launchExecProcess(monitor.ExecOptions{Mode: "auto"})
		}
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Swarm Scale Modal Handler
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Viewer Modals Handlers (Logs, Inspect, Prune, Help)
// -----------------------------------------------------------------------------

func (m Model) handleLogsModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.logsFiltering {
		switch msg.Type {
		case tea.KeyEsc, tea.KeyEnter:
			if msg.Type == tea.KeyEsc {
				m.logsFilterQuery = ""
			}
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
		m.logsFiltering = true
		return m, nil

	case "esc":
		if m.logsFilterQuery != "" {
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
		if m.logsScroll > 0 {
			m.logsScroll--
		}
		return m, nil

	case "down", "j":
		total := m.countFilteredLogs()
		if m.logsScroll < total-10 {
			m.logsScroll++
		}
		return m, nil

	case "pgup":
		m.logsScroll = max(0, m.logsScroll-10)
		return m, nil

	case "pgdown":
		total := m.countFilteredLogs()
		m.logsScroll = min(max(0, total-10), m.logsScroll+10)
		return m, nil
	}
	return m, nil
}

func (m Model) handleInspectModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.closeModal()
		return m, nil

	case "up", "k":
		if m.inspectScroll > 0 {
			m.inspectScroll--
		}
		return m, nil

	case "down", "j":
		total := m.countInspectLines()
		if m.inspectScroll < total-10 {
			m.inspectScroll++
		}
		return m, nil

	case "pgup":
		m.inspectScroll = max(0, m.inspectScroll-10)
		return m, nil

	case "pgdown":
		total := m.countInspectLines()
		m.inspectScroll = min(max(0, total-10), m.inspectScroll+10)
		return m, nil

	case "y", "Y":
		targetToCopy := m.inspectCopyID
		if targetToCopy == "" && m.inspectTarget.ID != "" {
			targetToCopy = m.inspectTarget.ID
		}
		if targetToCopy != "" {
			m.copyToClipboard(targetToCopy)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handlePruneModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			m.pruneConfirm = int(msg.String()[0] - '0')
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

func (m Model) handleHelpInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "?":
		m.showHelp = false
		return m, nil
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
