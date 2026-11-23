package tui

import (
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

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
