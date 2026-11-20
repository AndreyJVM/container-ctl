package tui

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/clipboard"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/modals"
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
			if err := clipboard.Write(targetToCopy); err != nil {
				m.showToast(fmt.Sprintf("Clipboard error: %v", err))
			} else {
				m.showToast(fmt.Sprintf("Copied to clipboard: %s", targetToCopy))
			}
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
	if m.logsFilterQuery == "" {
		return len(m.logsLines)
	}
	q := strings.ToLower(m.logsFilterQuery)
	cnt := 0
	for _, l := range m.logsLines {
		if strings.Contains(strings.ToLower(l), q) {
			cnt++
		}
	}
	return cnt
}

func (m Model) handleHelpInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "?":
		m.showHelp = false
		return m, nil
	}
	return m, nil
}

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
