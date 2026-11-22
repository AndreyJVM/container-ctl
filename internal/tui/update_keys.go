package tui

import (
	"fmt"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

var getDockerDiskUsage = monitor.GetDockerDiskUsage

func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.isFiltering = false
		m.filterQuery = ""
		m.clampCursor()
		return m, nil
	case tea.KeyEnter:
		m.isFiltering = false
		return m, nil
	case tea.KeyBackspace:
		if len(m.filterQuery) > 0 {
			runes := []rune(m.filterQuery)
			m.filterQuery = string(runes[:len(runes)-1])
			m.clampCursor()
		}
		return m, nil
	case tea.KeyRunes, tea.KeySpace:
		m.filterQuery += msg.String()
		m.clampCursor()
		return m, nil
	}
	return m, nil
}

func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.closeEmbeddedShell()
		return m, tea.Quit

	case "?":
		m.showHelp = true
		return m, nil

	// Tab navigation.
	case "tab", "right":
		m.nextTab()
	case "shift+tab", "left":
		m.prevTab()

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if len(msg.Runes) > 0 {
			idx := int(msg.Runes[0] - '0')
			if !m.setTabByIndex(idx) {
				if msg.String() == "6" && !m.IsSwarmVisible() {
					m.showToast("Host is not part of an active Swarm cluster.")
				}
			}
		}

	case "r", "R":
		m.showToast("Refreshing data...")
		return m, fetchDataCmd()

	// Container sorting on the Docker tab.
	case "o", "O":
		if m.activeTab == TabDocker && !m.isFiltering {
			m.containerSortMode = (m.containerSortMode + 1) % sortModeCount
			m.clampCursor()
			sortNames := []string{
				"Default",
				"CPU ↓",
				"Memory ↓",
				"Name ↑",
				"Status",
			}
			if m.containerSortMode >= 0 && m.containerSortMode < len(sortNames) {
				m.showToast(fmt.Sprintf("Sort: %s", sortNames[m.containerSortMode]))
			}
			return m, nil
		}

	// Search & filter on Docker, Images, Compose, Networks, and Volumes tabs.
	case "/":
		if m.activeTab == TabDocker || m.activeTab == TabImages || m.activeTab == TabCompose || m.activeTab == TabNetworks || m.activeTab == TabVolumes {
			m.isFiltering = true
			return m, nil
		}

	case "esc":
		if m.filterQuery != "" {
			m.filterQuery = ""
			m.clampCursor()
			return m, nil
		}
		if m.activeTab == TabImages && m.onlyUnusedImages {
			m.onlyUnusedImages = false
			m.clampCursor()
			m.showToast("Showing all images")
			return m, nil
		}

	// List navigation across all tabs.
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "pgup":
		m.moveCursor(-5)
	case "pgdown":
		m.moveCursor(5)
	case "home", "g":
		if !m.isFiltering {
			m.cursorToStart()
		}
	case "end", "G":
		if !m.isFiltering {
			m.cursorToEnd()
		}

	// Logs viewer: Containers, Compose, Swarm.
	case "l", "L":
		switch m.activeTab {
		case TabDocker:
			return m.handleDockerLogs()
		case TabCompose:
			return m.handleComposeLogs()
		case TabSwarm:
			return m.handleSwarmLogs()
		}

	// Inspection: Containers, Networks, Volumes.
	case "enter", "i", "I":
		switch m.activeTab {
		case TabDocker:
			return m.handleDockerInspect()
		case TabNetworks:
			return m.handleNetworksInspect()
		case TabVolumes:
			return m.handleVolumesInspect()
		}

	// Start or stop: Container, Compose project.
	case "s", "S":
		switch m.activeTab {
		case TabDocker:
			return m.handleDockerStartStop()
		case TabCompose:
			return m.handleComposeStop()
		}

	// Toggle unused images filter or Start Compose project.
	case "u", "U":
		if !m.isFiltering {
			switch m.activeTab {
			case TabImages:
				return m.handleImagesToggleUnused()
			case TabCompose:
				return m.handleComposeStart()
			}
		}

	// Down Compose, Remove Container, Remove Image.
	case "d", "D":
		if !m.isFiltering {
			switch m.activeTab {
			case TabDocker:
				return m.handleDockerRemoveModal()
			case TabImages:
				return m.handleImagesRemoveModal()
			case TabCompose:
				return m.handleComposeDownModal()
			}
		}

	// Open an interactive container shell.
	case "e", "E":
		if m.activeTab == TabDocker {
			return m.handleDockerExec()
		}

	// Force-kill a container with SIGKILL.
	case "K":
		if m.activeTab == TabDocker && !m.isFiltering {
			return m.handleDockerKill()
		}

	// Remove a container or an image.
	case "x", "X":
		if !m.isFiltering {
			switch m.activeTab {
			case TabDocker:
				return m.handleDockerRemoveModal()
			case TabImages:
				return m.handleImagesRemoveModal()
			}
		}

	// Restart a container or Compose project.
	case "p", "P":
		switch m.activeTab {
		case TabDocker:
			return m.handleDockerRestart()
		case TabCompose:
			return m.handleComposeRestart()
		}

	// Prune unused Docker resources.
	case "c", "C":
		if (m.activeTab == TabDocker || m.activeTab == TabImages || m.activeTab == TabCompose || m.activeTab == TabNetworks || m.activeTab == TabVolumes) && !m.isFiltering {
			m.openModal(ModalPrune)
			m.pruneConfirm = 0
			usage, err := getDockerDiskUsage()
			m.pruneUsage = usage
			if err != nil {
				m.pruneUsageErr = err.Error()
			} else {
				m.pruneUsageErr = ""
			}
			return m, nil
		}

	// Open container localhost port URL in browser / copy URL.
	case "w", "W":
		if m.activeTab == TabDocker && !m.isFiltering {
			return m.handleDockerOpenURL()
		}

	// Copy command or ID to clipboard via terminal OSC 52.
	case "y", "Y":
		if !m.isFiltering {
			switch m.activeTab {
			case TabDocker:
				return m.handleDockerCopy()
			case TabImages:
				return m.handleImagesCopy()
			case TabCompose:
				return m.handleComposeCopy()
			case TabNetworks:
				return m.handleNetworksCopy()
			case TabVolumes:
				return m.handleVolumesCopy()
			case TabSwarm:
				return m.handleSwarmCopy()
			}
		}
	}
	return m, nil
}
