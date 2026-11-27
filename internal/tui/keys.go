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
		return m, m.FetchDataCmd()

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
		if m.isFilterableTab() {
			m.isFiltering = true
			return m, nil
		}

	case "esc":
		if m.filterQuery != "" {
			m.filterQuery = ""
			m.clampCursor()
			return m, nil
		}
		switch m.activeTab {
		case TabDocker:
			if len(m.selectedContainers) > 0 {
				m.clearSelectedContainers()
				m.showToast("Selection cleared")
				return m, nil
			}
		case TabImages:
			if len(m.selectedImages) > 0 {
				m.clearSelectedImages()
				m.showToast("Selection cleared")
				return m, nil
			}
		case TabNetworks:
			if len(m.selectedNetworks) > 0 {
				m.clearSelectedNetworks()
				m.showToast("Selection cleared")
				return m, nil
			}
		case TabVolumes:
			if len(m.selectedVolumes) > 0 {
				m.clearSelectedVolumes()
				m.showToast("Selection cleared")
				return m, nil
			}
		}
		if m.activeTab == TabImages && m.onlyUnusedImages {
			m.onlyUnusedImages = false
			m.clampCursor()
			m.showToast("Showing all images")
			return m, nil
		}

	// Multi-select toggle and select-all.
	case " ":
		if !m.isFiltering {
			switch m.activeTab {
			case TabDocker:
				m.toggleContainerSelection()
				return m, nil
			case TabImages:
				m.toggleImageSelection()
				return m, nil
			case TabNetworks:
				m.toggleNetworkSelection()
				return m, nil
			case TabVolumes:
				m.toggleVolumeSelection()
				return m, nil
			}
		}

	case "a", "A", "ctrl+a":
		if !m.isFiltering {
			switch m.activeTab {
			case TabDocker:
				m.toggleSelectAllContainers()
				return m, nil
			case TabImages:
				m.toggleSelectAllImages()
				return m, nil
			case TabNetworks:
				m.toggleSelectAllNetworks()
				return m, nil
			case TabVolumes:
				m.toggleSelectAllVolumes()
				return m, nil
			}
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

	// Start or stop: Container, Compose project; Scale: Swarm service.
	case "s", "S":
		switch m.activeTab {
		case TabDocker:
			if len(m.selectedContainers) > 0 {
				return m.handleDockerBatchStartStop()
			}
			return m.handleDockerStartStop()
		case TabCompose:
			return m.handleComposeStop()
		case TabSwarm:
			return m.handleSwarmScaleModal()
		}

	// Quick replica scale: Swarm service.
	case "+", "=":
		if m.activeTab == TabSwarm {
			return m.handleSwarmQuickScale(1)
		}

	case "-", "_":
		if m.activeTab == TabSwarm {
			return m.handleSwarmQuickScale(-1)
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

	// Down Compose, Remove Container, Remove Image, Remove Network, Remove Volume.
	case "d", "D":
		if !m.isFiltering {
			switch m.activeTab {
			case TabDocker:
				if len(m.selectedContainers) > 0 {
					return m.handleDockerBatchRemoveModal()
				}
				return m.handleDockerRemoveModal()
			case TabImages:
				if len(m.selectedImages) > 0 {
					return m.handleImagesBatchRemoveModal()
				}
				return m.handleImagesRemoveModal()
			case TabCompose:
				return m.handleComposeDownModal()
			case TabNetworks:
				if len(m.selectedNetworks) > 0 {
					return m.handleNetworksBatchRemoveModal()
				}
				return m.handleNetworksRemoveModal()
			case TabVolumes:
				if len(m.selectedVolumes) > 0 {
					return m.handleVolumesBatchRemoveModal()
				}
				return m.handleVolumesRemoveModal()
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
			if len(m.selectedContainers) > 0 {
				return m.handleDockerBatchKill()
			}
			return m.handleDockerKill()
		}

	// Remove a container, image, network, or volume.
	case "x", "X":
		if !m.isFiltering {
			switch m.activeTab {
			case TabDocker:
				if len(m.selectedContainers) > 0 {
					return m.handleDockerBatchRemoveModal()
				}
				return m.handleDockerRemoveModal()
			case TabImages:
				if len(m.selectedImages) > 0 {
					return m.handleImagesBatchRemoveModal()
				}
				return m.handleImagesRemoveModal()
			case TabNetworks:
				if len(m.selectedNetworks) > 0 {
					return m.handleNetworksBatchRemoveModal()
				}
				return m.handleNetworksRemoveModal()
			case TabVolumes:
				if len(m.selectedVolumes) > 0 {
					return m.handleVolumesBatchRemoveModal()
				}
				return m.handleVolumesRemoveModal()
			}
		}

	// Restart a container or Compose project.
	case "p", "P":
		switch m.activeTab {
		case TabDocker:
			if len(m.selectedContainers) > 0 {
				return m.handleDockerBatchRestart()
			}
			return m.handleDockerRestart()
		case TabCompose:
			return m.handleComposeRestart()
		}

	// Prune unused Docker resources.
	case "c", "C":
		if m.isFilterableTab() && !m.isFiltering {
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

	// Toggle lower details card visibility.
	case "v", "V":
		if !m.isFiltering {
			m.ToggleDetails()
			if m.hideDetails {
				m.showToast("Details card hidden (press 'V' to restore)")
			} else {
				m.showToast("Details card visible")
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) isFilterableTab() bool {
	switch m.activeTab {
	case TabDocker, TabImages, TabCompose, TabNetworks, TabVolumes:
		return true
	default:
		return false
	}
}
