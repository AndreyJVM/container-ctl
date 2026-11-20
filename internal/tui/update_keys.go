package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/clipboard"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
)

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

	case "1":
		m.setTab(TabDocker)
	case "2":
		m.setTab(TabImages)
	case "3":
		m.setTab(TabCompose)
	case "4":
		m.setTab(TabNetworks)
	case "5":
		m.setTab(TabVolumes)
	case "6":
		if m.IsSwarmVisible() {
			m.setTab(TabSwarm)
		} else {
			m.showToast("Host is not part of an active Swarm cluster.")
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

	// Search & filter on Docker and Images tabs.
	case "/":
		if m.activeTab == TabDocker || m.activeTab == TabImages {
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

	// List navigation for Docker, Compose, and Swarm.
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)

	// Container, Compose project, or Swarm service logs.
	case "l", "L":
		if m.activeTab == TabDocker {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				m.logsTargetName = c.Name
				m.logsTargetID = c.ID
				logs, err := monitor.GetContainerLogs(c.ID, 120)
				if err != nil {
					m.showToast("Logs error: " + err.Error())
				} else {
					m.logsLines = strings.Split(strings.TrimSpace(logs), "\n")
					m.logsScroll = max(0, len(m.logsLines)-18)
					m.openModal(ModalLogs)
				}
			}
		} else if m.activeTab == TabCompose && len(m.compose) > 0 && m.composeCursorIdx < len(m.compose) {
			proj := m.compose[m.composeCursorIdx]
			m.logsTargetName = "Compose: " + proj.Name
			m.logsTargetID = proj.Name
			logs, err := monitor.GetComposeProjectLogs(proj.Name, m.dockerInfo.Containers, 150)
			if err != nil {
				m.showToast("Logs error: " + err.Error())
			} else {
				m.logsLines = strings.Split(strings.TrimSpace(logs), "\n")
				m.logsScroll = max(0, len(m.logsLines)-18)
				m.openModal(ModalLogs)
			}
		} else if m.activeTab == TabSwarm && len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
			svc := m.swarm.Services[m.swarmCursorIdx]
			m.logsTargetName = "Swarm: " + svc.Name
			m.logsTargetID = svc.ID
			logs, err := monitor.GetSwarmServiceLogs(svc.ID, 150)
			if err != nil {
				m.showToast("Logs error: " + err.Error())
			} else {
				m.logsLines = strings.Split(strings.TrimSpace(logs), "\n")
				m.logsScroll = max(0, len(m.logsLines)-18)
				m.openModal(ModalLogs)
			}
		}

	// Container inspection.
	case "enter", "i", "I":
		containers := m.getFilteredContainers()
		if m.activeTab == TabDocker && len(containers) > 0 && m.cursorIdx < len(containers) {
			c := containers[m.cursorIdx]
			detail, err := monitor.GetContainerDetail(c.ID)
			if err != nil {
				m.showToast("Inspect error: " + err.Error())
			} else {
				m.inspectTarget = detail
				m.openModal(ModalInspect)
			}
		}

	// Start or stop a container or Compose project.
	case "s", "S":
		if m.activeTab == TabDocker {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				if c.State == "running" {
					m.showToast(fmt.Sprintf("Stopping container %s...", c.Name))
					return m, stopContainerCmd(c.ID, c.Name)
				}
				m.showToast(fmt.Sprintf("Starting container %s...", c.Name))
				return m, startContainerCmd(c.ID, c.Name)
			}
		} else if m.activeTab == TabCompose && len(m.compose) > 0 && m.composeCursorIdx < len(m.compose) {
			proj := m.compose[m.composeCursorIdx]
			m.showToast(fmt.Sprintf("Stopping Compose project %s...", proj.Name))
			return m, stopComposeCmd(proj.Name, m.dockerInfo.Containers)
		}

	// Toggle unused images filter on Images tab, or Start a Compose project.
	case "u", "U":
		if m.activeTab == TabImages && !m.isFiltering {
			m.onlyUnusedImages = !m.onlyUnusedImages
			m.clampCursor()
			if m.onlyUnusedImages {
				m.showToast("Filter: Unused images only")
			} else {
				m.showToast("Showing all images")
			}
			return m, nil
		} else if m.activeTab == TabCompose && len(m.compose) > 0 && m.composeCursorIdx < len(m.compose) {
			proj := m.compose[m.composeCursorIdx]
			m.showToast(fmt.Sprintf("Starting Compose project %s...", proj.Name))
			return m, startComposeCmd(proj.Name, m.dockerInfo.Containers)
		}

	// Down a Compose project, remove a container, or remove an image.
	case "d", "D":
		if m.activeTab == TabDocker && !m.isFiltering {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				m.removeContainerTargetID = c.ID
				m.removeContainerTargetName = c.Name
				m.openModal(ModalRemoveContainer)
				return m, nil
			}
		} else if m.activeTab == TabImages && !m.isFiltering {
			images := m.getFilteredImages()
			if len(images) > 0 && m.imageCursorIdx < len(images) {
				img := images[m.imageCursorIdx]
				displayName := img.Repository
				if img.Tag != "" && img.Tag != "<none>" {
					displayName = fmt.Sprintf("%s:%s", img.Repository, img.Tag)
				}
				m.removeImageTargetID = img.ID
				m.removeImageTargetName = displayName
				m.removeImageTargetInUse = img.InUse
				m.openModal(ModalRemoveImage)
				return m, nil
			}
		} else if m.activeTab == TabCompose && len(m.compose) > 0 && m.composeCursorIdx < len(m.compose) {
			proj := m.compose[m.composeCursorIdx]
			m.composeDownTarget = proj.Name
			m.openModal(ModalComposeDown)
			return m, nil
		}

	// Open an interactive container shell.
	case "e", "E":
		if m.activeTab == TabDocker {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				if c.State != "running" {
					m.showToast(fmt.Sprintf("Cannot connect: container %s is not running", c.Name))
					return m, nil
				}
				if !monitor.IsDockerCLIAvailable() {
					m.showToast("Error: docker binary not found in $PATH")
					return m, nil
				}
				m.openModal(ModalExec)
				m.execTargetID = c.ID
				m.execTargetName = c.Name
				m.execCursorIdx = 0
				m.execIsCustomInput = false
				m.execCustomCmd = ""
				return m, nil
			}
		}

	// Force-kill a container with SIGKILL.
	case "K":
		if m.activeTab == TabDocker && !m.isFiltering {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				if c.State == "running" {
					m.showToast(fmt.Sprintf("Force killing %s...", c.Name))
					return m, killContainerCmd(c.ID, c.Name)
				}
				m.showToast(fmt.Sprintf("Cannot connect: container %s is not running", c.Name))
				return m, nil
			}
		}

	// Remove a container or an image.
	case "x", "X":
		if m.activeTab == TabDocker && !m.isFiltering {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				m.removeContainerTargetID = c.ID
				m.removeContainerTargetName = c.Name
				m.openModal(ModalRemoveContainer)
				return m, nil
			}
		} else if m.activeTab == TabImages && !m.isFiltering {
			images := m.getFilteredImages()
			if len(images) > 0 && m.imageCursorIdx < len(images) {
				img := images[m.imageCursorIdx]
				displayName := img.Repository
				if img.Tag != "" && img.Tag != "<none>" {
					displayName = fmt.Sprintf("%s:%s", img.Repository, img.Tag)
				}
				m.removeImageTargetID = img.ID
				m.removeImageTargetName = displayName
				m.removeImageTargetInUse = img.InUse
				m.openModal(ModalRemoveImage)
				return m, nil
			}
		}

	// Restart a container or Compose project.
	case "p", "P":
		if m.activeTab == TabDocker {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				m.showToast(fmt.Sprintf("Restarting %s...", c.Name))
				return m, restartContainerCmd(c.ID, c.Name)
			}
		} else if m.activeTab == TabCompose && len(m.compose) > 0 && m.composeCursorIdx < len(m.compose) {
			proj := m.compose[m.composeCursorIdx]
			m.showToast(fmt.Sprintf("Restarting Compose project %s...", proj.Name))
			return m, restartComposeCmd(proj.Name, m.dockerInfo.Containers)
		}

	// Prune unused Docker resources.
	case "c", "C":
		if (m.activeTab == TabDocker || m.activeTab == TabImages || m.activeTab == TabCompose) && !m.isFiltering {
			m.openModal(ModalPrune)
			m.pruneConfirm = 0
			usage, err := monitor.GetDockerDiskUsage()
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
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				urls := views.ExtractPortURLs(c.Ports)
				if len(urls) == 0 {
					m.showToast(fmt.Sprintf("No mapped ports for %s", c.Name))
					return m, nil
				}
				targetURL := urls[0]
				_ = clipboard.Write(targetURL)
				if openBrowser(targetURL) {
					m.showToast(fmt.Sprintf("Opened %s (copied to clipboard)", targetURL))
				} else {
					m.showToast(fmt.Sprintf("Copied URL: %s", targetURL))
				}
				return m, nil
			}
		}

	// Copy command or ID to clipboard via terminal OSC 52.
	case "y", "Y":
		if m.activeTab == TabDocker && !m.isFiltering {
			containers := m.getFilteredContainers()
			if len(containers) > 0 && m.cursorIdx < len(containers) {
				c := containers[m.cursorIdx]
				cmdText := fmt.Sprintf("docker exec -it %s sh", c.Name)
				_ = clipboard.Write(cmdText)
				m.showToast(fmt.Sprintf("Copied to clipboard: %s", cmdText))
				return m, nil
			}
		} else if m.activeTab == TabImages && !m.isFiltering {
			images := m.getFilteredImages()
			if len(images) > 0 && m.imageCursorIdx < len(images) {
				img := images[m.imageCursorIdx]
				name := img.Repository
				if img.Tag != "" && img.Tag != "<none>" {
					name = fmt.Sprintf("%s:%s", img.Repository, img.Tag)
				}
				_ = clipboard.Write(name)
				m.showToast(fmt.Sprintf("Copied to clipboard: %s", name))
				return m, nil
			}
		} else if m.activeTab == TabCompose && !m.isFiltering {
			if len(m.compose) > 0 && m.composeCursorIdx < len(m.compose) {
				proj := m.compose[m.composeCursorIdx]
				_ = clipboard.Write(proj.Name)
				m.showToast(fmt.Sprintf("Copied to clipboard: %s", proj.Name))
				return m, nil
			}
		}
	}
	return m, nil
}

// openBrowser attempts to open the given URL in the default browser.
func openBrowser(url string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err == nil {
		go func() { _ = cmd.Wait() }()
		return true
	}
	return false
}
