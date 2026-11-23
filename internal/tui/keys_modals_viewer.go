package tui

import (
	"github.com/AndreyJVM/container-ctl/internal/tui/modals"
	tea "github.com/charmbracelet/bubbletea"
)

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
