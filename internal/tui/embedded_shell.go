package tui

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/shell"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	embeddedShellWideLayoutMin = shell.WideLayoutMin
	embeddedShellMinWidth      = shell.MinWidth
	embeddedShellMinHeight     = shell.MinHeight
)

func readEmbeddedShellCmd(session embeddedShellSession) tea.Cmd {
	return func() tea.Msg {
		data, err := session.Read()
		return embeddedShellOutputMsg{session: session, data: data, err: err}
	}
}

func waitEmbeddedShellCmd(session embeddedShellSession) tea.Cmd {
	return func() tea.Msg {
		return embeddedShellExitedMsg{session: session, err: session.Wait()}
	}
}

func (m Model) launchEmbeddedShell() (tea.Model, tea.Cmd) {
	targetID := m.execTargetID
	targetName := m.execTargetName
	m.closeModal()
	m.clearExecTarget()

	width, height := embeddedShellSize(m.width, m.height)
	cmd := monitor.GetContainerExecCommandWithOptions(targetID, monitor.ExecOptions{
		Mode:     "auto",
		Terminal: "xterm-256color",
	})
	session, err := startEmbeddedShell(cmd, width, height)
	if err != nil {
		m.showToast(fmt.Sprintf("Shell error in %s: %v", targetName, err))
		return m, nil
	}

	m.embeddedShell = session
	m.shellFocused = true
	m.shellContainerName = targetName
	return m, tea.Batch(readEmbeddedShellCmd(session), waitEmbeddedShellCmd(session))
}

func (m *Model) clearExecTarget() {
	m.execTargetID = ""
	m.execTargetName = ""
	m.execIsCustomInput = false
	m.execCustomCmd = ""
}

func (m *Model) closeEmbeddedShell() {
	if m.embeddedShell != nil {
		_ = m.embeddedShell.Close()
	}
	m.embeddedShell = nil
	m.shellFocused = false
	m.shellContainerName = ""
}

func (m Model) handleEmbeddedShellOutput(msg embeddedShellOutputMsg) (tea.Model, tea.Cmd) {
	if m.embeddedShell == nil || msg.session != m.embeddedShell {
		return m, nil
	}
	if len(msg.data) > 0 {
		if err := m.embeddedShell.Feed(msg.data); err != nil {
			m.showToast(fmt.Sprintf("Shell error in %s: %v", m.shellContainerName, err))
			m.closeEmbeddedShell()
			return m, nil
		}
	}
	if msg.err != nil {
		return m, nil
	}
	return m, readEmbeddedShellCmd(m.embeddedShell)
}

func (m Model) handleEmbeddedShellExited(msg embeddedShellExitedMsg) (tea.Model, tea.Cmd) {
	if m.embeddedShell == nil || msg.session != m.embeddedShell {
		return m, nil
	}
	name := m.shellContainerName
	m.closeEmbeddedShell()
	var exitErr *exec.ExitError
	if msg.err == nil {
		m.showToast(fmt.Sprintf("Shell session in %s ended", name))
	} else if errors.As(msg.err, &exitErr) {
		if exitErr.ExitCode() == 130 {
			m.showToast(fmt.Sprintf("Shell session in %s ended", name))
		} else {
			m.showToast(fmt.Sprintf("Shell session in %s ended (code %d)", name, exitErr.ExitCode()))
		}
	} else {
		m.showToast(fmt.Sprintf("Shell error in %s: %v", name, msg.err))
	}
	return m, fetchDataCmd()
}

func (m Model) handleEmbeddedShellKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlCloseBracket {
		m.shellFocused = false
		return m, nil
	}
	if msg.Type == tea.KeyCtrlX {
		name := m.shellContainerName
		m.closeEmbeddedShell()
		m.showToast(fmt.Sprintf("Shell session in %s closed", name))
		return m, fetchDataCmd()
	}
	data := shellKeyBytes(msg)
	if len(data) == 0 {
		return m, nil
	}
	if err := m.embeddedShell.Write(data); err != nil {
		m.showToast(fmt.Sprintf("Shell error in %s: %v", m.shellContainerName, err))
		m.closeEmbeddedShell()
	}
	return m, nil
}

func shellKeyBytes(msg tea.KeyMsg) []byte {
	return shell.KeyBytes(msg)
}

func embeddedShellSize(width, height int) (int, int) {
	return shell.PanelSize(width, height)
}
