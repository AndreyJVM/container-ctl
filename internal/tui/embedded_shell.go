package tui

import (
	"fmt"
	"os/exec"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	embeddedShellWideLayoutMin = 110
	embeddedShellMinWidth      = 20
	embeddedShellMinHeight     = 6
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
	if msg.err == nil {
		m.showToast(fmt.Sprintf("Shell session in %s ended", name))
	} else if exitErr, ok := msg.err.(*exec.ExitError); ok {
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
	if msg.Type == tea.KeyRunes {
		data := []byte(string(msg.Runes))
		if msg.Alt {
			return append([]byte{0x1b}, data...)
		}
		return data
	}

	if msg.Type >= 0 && msg.Type <= 31 {
		data := []byte{byte(msg.Type)} // #nosec G115 -- bounded by msg.Type >= 0 && msg.Type <= 31
		if msg.Alt {
			return append([]byte{0x1b}, data...)
		}
		return data
	}
	if msg.Type == 127 {
		data := []byte{127}
		if msg.Alt {
			return append([]byte{0x1b}, data...)
		}
		return data
	}

	sequences := map[tea.KeyType]string{
		tea.KeyUp:       "\x1b[A",
		tea.KeyDown:     "\x1b[B",
		tea.KeyRight:    "\x1b[C",
		tea.KeyLeft:     "\x1b[D",
		tea.KeyHome:     "\x1b[H",
		tea.KeyEnd:      "\x1b[F",
		tea.KeyPgUp:     "\x1b[5~",
		tea.KeyPgDown:   "\x1b[6~",
		tea.KeyDelete:   "\x1b[3~",
		tea.KeyInsert:   "\x1b[2~",
		tea.KeyShiftTab: "\x1b[Z",
	}
	sequence := sequences[msg.Type]
	if sequence == "" {
		return nil
	}
	if msg.Alt {
		sequence = "\x1b" + sequence
	}
	return []byte(sequence)
}

func embeddedShellSize(width, height int) (int, int) {
	panelWidth := width
	if width >= embeddedShellWideLayoutMin {
		panelWidth = width / 2
	}
	return max(embeddedShellMinWidth, panelWidth-4), max(embeddedShellMinHeight, height-12)
}
