package tui

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case embeddedShellOutputMsg:
		return m.handleEmbeddedShellOutput(msg)

	case embeddedShellExitedMsg:
		return m.handleEmbeddedShellExited(msg)

	case execFinishedMsg:
		return m.handleExecFinished(msg)

	case actions.FinishedMsg:
		return m.handleActionResult(msg.Err, msg.Message, msg.Refresh)

	case actions.DataFetchedMsg:
		m.applyActionsData(msg)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.embeddedShell != nil {
			width, height := embeddedShellSize(m.width, m.height)
			if err := m.embeddedShell.Resize(width, height); err != nil {
				m.showToast(fmt.Sprintf("Shell error in %s: %v", m.shellContainerName, err))
				m.closeEmbeddedShell()
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.embeddedShell != nil && m.shellFocused {
			return m.handleEmbeddedShellKey(msg)
		}

		// Active modals intercept keyboard input.
		if m.activeModal != ModalNone {
			return m.handleModalInput(msg)
		}

		// Search and filter mode.
		if m.isFiltering && (m.activeTab == TabDocker || m.activeTab == TabImages || m.activeTab == TabCompose || m.activeTab == TabNetworks || m.activeTab == TabVolumes) {
			return m.handleFilterInput(msg)
		}

		// Help modal.
		if m.showHelp {
			return m.handleHelpInput(msg)
		}

		// Unfocused embedded shell hotkeys.
		if m.embeddedShell != nil {
			switch msg.Type {
			case tea.KeyCtrlX:
				m.closeEmbeddedShell()
				return m, nil
			}
			if msg.String() == "e" || msg.String() == "E" {
				m.shellFocused = true
				return m, nil
			}
		}

		// Main input mode.
		return m.handleMainKeys(msg)

	case tickMsg:
		return m, tea.Batch(
			tickCmd(m.refreshInterval),
			m.FetchDataCmd(),
		)
	}
	return m, nil
}

func (m Model) handleExecFinished(msg execFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		var exitErr *exec.ExitError
		if errors.As(msg.err, &exitErr) {
			if exitErr.ExitCode() == 130 {
				m.showToast(fmt.Sprintf("Shell session in %s ended", msg.containerName))
			} else {
				m.showToast(fmt.Sprintf("Shell session in %s ended (code %d)", msg.containerName, exitErr.ExitCode()))
			}
		} else {
			m.showToast(fmt.Sprintf("Shell error in %s: %v", msg.containerName, msg.err))
		}
	} else {
		m.showToast(fmt.Sprintf("Shell session in %s ended", msg.containerName))
	}
	return m, m.FetchDataCmd()
}

func (m *Model) handleActionResult(err error, message string, refresh bool) (tea.Model, tea.Cmd) {
	if err != nil {
		m.showToast(err.Error())
	} else if message != "" {
		m.showToast(message)
	}
	if refresh {
		return *m, m.FetchDataCmd()
	}
	return *m, nil
}
