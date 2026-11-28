package tui

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/modals"
	"github.com/AndreyJVM/container-ctl/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
)

// -----------------------------------------------------------------------------
// Browser opener helper
// -----------------------------------------------------------------------------

var (
	browserOpenerMu sync.RWMutex
	browserOpener   = defaultBrowserOpener
)

func defaultBrowserOpener(url string) bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err == nil {
		go func() { _ = cmd.Wait() }()
		return true
	}
	return false
}

func openBrowser(url string) bool {
	browserOpenerMu.RLock()
	fn := browserOpener
	browserOpenerMu.RUnlock()
	return fn(url)
}

// -----------------------------------------------------------------------------
// Docker Tab Handlers
// -----------------------------------------------------------------------------

func (m Model) handleDockerInspect() (Model, tea.Cmd) {
	containers := m.getFilteredContainers()
	if len(containers) > 0 && m.cursorIdx < len(containers) {
		c := containers[m.cursorIdx]
		detail, err := monitor.GetContainerDetail(c.ID)
		if err != nil {
			m.showToast("Inspect error: " + err.Error())
		} else {
			m.inspectTarget = detail
			m.inspectCustomTitle = ""
			m.inspectCustomLines = nil
			m.inspectCopyID = c.ID
			m.inspectScroll = 0
			m.openModal(ModalInspect)
		}
	}
	return m, nil
}

func (m Model) handleDockerLogs() (Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleDockerStartStop() (Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleDockerRestart() (Model, tea.Cmd) {
	containers := m.getFilteredContainers()
	if len(containers) > 0 && m.cursorIdx < len(containers) {
		c := containers[m.cursorIdx]
		m.showToast(fmt.Sprintf("Restarting %s...", c.Name))
		return m, restartContainerCmd(c.ID, c.Name)
	}
	return m, nil
}

func (m Model) handleDockerKill() (Model, tea.Cmd) {
	containers := m.getFilteredContainers()
	if len(containers) > 0 && m.cursorIdx < len(containers) {
		c := containers[m.cursorIdx]
		if c.State == "running" {
			m.showToast(fmt.Sprintf("Force killing %s...", c.Name))
			return m, killContainerCmd(c.ID, c.Name)
		}
		m.showToast(fmt.Sprintf("Cannot kill: container %s is not running", c.Name))
	}
	return m, nil
}

func (m Model) handleDockerExec() (Model, tea.Cmd) {
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
	}
	return m, nil
}

func (m Model) handleDockerRemoveModal() (Model, tea.Cmd) {
	containers := m.getFilteredContainers()
	if len(containers) > 0 && m.cursorIdx < len(containers) {
		c := containers[m.cursorIdx]
		m.removeContainerTargetID = c.ID
		m.removeContainerTargetName = c.Name
		m.openModal(ModalRemoveContainer)
	}
	return m, nil
}

func (m Model) handleDockerCopy() (Model, tea.Cmd) {
	containers := m.getFilteredContainers()
	if len(containers) > 0 && m.cursorIdx < len(containers) {
		c := containers[m.cursorIdx]
		cmdText := fmt.Sprintf("%s exec -it %s sh", monitor.GetContainerCLI(), c.Name)
		m.copyToClipboard(cmdText)
	}
	return m, nil
}

func (m Model) handleDockerOpenURL() (Model, tea.Cmd) {
	containers := m.getFilteredContainers()
	if len(containers) > 0 && m.cursorIdx < len(containers) {
		c := containers[m.cursorIdx]
		if c.State != "running" {
			m.showToast(fmt.Sprintf("Cannot open URL: container %s is not running", c.Name))
			return m, nil
		}
		urls := views.ExtractPortURLs(c.Ports)
		if len(urls) == 0 {
			m.showToast(fmt.Sprintf("No mapped ports for %s", c.Name))
			return m, nil
		}
		targetURL := urls[0]
		if err := clipboard.Write(targetURL); err != nil {
			m.showToast(fmt.Sprintf("Clipboard error: %v", err))
		} else if openBrowser(targetURL) {
			m.showToast(fmt.Sprintf("Opened %s (copied to clipboard)", targetURL))
		} else {
			m.showToast(fmt.Sprintf("Copied URL: %s", targetURL))
		}
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Images Tab Handlers
// -----------------------------------------------------------------------------

func (m Model) handleImagesToggleUnused() (Model, tea.Cmd) {
	m.onlyUnusedImages = !m.onlyUnusedImages
	m.clampCursor()
	if m.onlyUnusedImages {
		m.showToast("Filter: Unused images only")
	} else {
		m.showToast("Showing all images")
	}
	return m, nil
}

func (m Model) handleImagesRemoveModal() (Model, tea.Cmd) {
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
	}
	return m, nil
}

func (m Model) handleImagesCopy() (Model, tea.Cmd) {
	images := m.getFilteredImages()
	if len(images) > 0 && m.imageCursorIdx < len(images) {
		img := images[m.imageCursorIdx]
		name := img.Repository
		if img.Tag != "" && img.Tag != "<none>" {
			name = fmt.Sprintf("%s:%s", img.Repository, img.Tag)
		}
		m.copyToClipboard(name)
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Compose Tab Handlers
// -----------------------------------------------------------------------------

func (m Model) handleComposeLogs() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
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
	}
	return m, nil
}

func (m Model) handleComposeStart() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.showToast(fmt.Sprintf("Starting Compose project %s...", proj.Name))
		return m, startComposeCmd(proj.Name, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleComposeStop() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.showToast(fmt.Sprintf("Stopping Compose project %s...", proj.Name))
		return m, stopComposeCmd(proj.Name, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleComposeRestart() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.showToast(fmt.Sprintf("Restarting Compose project %s...", proj.Name))
		return m, restartComposeCmd(proj.Name, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleComposeDownModal() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.composeDownTarget = proj.Name
		m.openModal(ModalComposeDown)
	}
	return m, nil
}

func (m Model) handleComposeCopy() (Model, tea.Cmd) {
	compose := m.getFilteredCompose()
	if len(compose) > 0 && m.composeCursorIdx < len(compose) {
		proj := compose[m.composeCursorIdx]
		m.copyToClipboard(proj.Name)
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Networks Tab Handlers
// -----------------------------------------------------------------------------

func (m Model) handleNetworksInspect() (Model, tea.Cmd) {
	networks := m.getFilteredNetworks()
	if len(networks) > 0 && m.networkCursorIdx < len(networks) {
		net := networks[m.networkCursorIdx]
		inspectStr, _ := monitor.GetNetworkInspectJSON(net.ID)
		if strings.TrimSpace(inspectStr) == "" {
			if b, mErr := json.MarshalIndent(net, "", "  "); mErr == nil {
				inspectStr = string(b)
			}
		}
		m.inspectTarget = monitor.ContainerDetail{}
		m.inspectCustomTitle = fmt.Sprintf("NETWORK INSPECTION: %s", net.Name)
		m.inspectCustomLines = modals.BuildNetworkInspectLines(net, inspectStr)
		m.inspectCopyID = net.ID
		if m.inspectCopyID == "" {
			m.inspectCopyID = net.Name
		}
		m.inspectScroll = 0
		m.openModal(ModalInspect)
	}
	return m, nil
}

func (m Model) handleNetworksCopy() (Model, tea.Cmd) {
	networks := m.getFilteredNetworks()
	if len(networks) > 0 && m.networkCursorIdx < len(networks) {
		net := networks[m.networkCursorIdx]
		m.copyToClipboard(net.Name)
	}
	return m, nil
}

func (m Model) handleNetworksRemoveModal() (Model, tea.Cmd) {
	networks := m.getFilteredNetworks()
	if len(networks) > 0 && m.networkCursorIdx < len(networks) {
		net := networks[m.networkCursorIdx]
		m.removeNetworkTargetID = net.ID
		m.removeNetworkTargetName = net.Name
		m.removeNetworkContainersCount = net.Containers
		m.openModal(ModalRemoveNetwork)
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Volumes Tab Handlers
// -----------------------------------------------------------------------------

func (m Model) handleVolumesInspect() (Model, tea.Cmd) {
	volumes := m.getFilteredVolumes()
	if len(volumes) > 0 && m.volumeCursorIdx < len(volumes) {
		vol := volumes[m.volumeCursorIdx]
		inspectStr, _ := monitor.GetVolumeInspectJSON(vol.Name)
		if strings.TrimSpace(inspectStr) == "" {
			if b, mErr := json.MarshalIndent(vol, "", "  "); mErr == nil {
				inspectStr = string(b)
			}
		}
		m.inspectTarget = monitor.ContainerDetail{}
		m.inspectCustomTitle = fmt.Sprintf("VOLUME INSPECTION: %s", vol.Name)
		m.inspectCustomLines = modals.BuildVolumeInspectLines(vol, inspectStr)
		m.inspectCopyID = vol.Name
		m.inspectScroll = 0
		m.openModal(ModalInspect)
	}
	return m, nil
}

func (m Model) handleVolumesCopy() (Model, tea.Cmd) {
	volumes := m.getFilteredVolumes()
	if len(volumes) > 0 && m.volumeCursorIdx < len(volumes) {
		vol := volumes[m.volumeCursorIdx]
		m.copyToClipboard(vol.Name)
	}
	return m, nil
}

func (m Model) handleVolumesRemoveModal() (Model, tea.Cmd) {
	volumes := m.getFilteredVolumes()
	if len(volumes) > 0 && m.volumeCursorIdx < len(volumes) {
		vol := volumes[m.volumeCursorIdx]
		m.removeVolumeTargetName = vol.Name
		m.removeVolumeTargetInUse = vol.RefCount > 0 || len(vol.Containers) > 0
		m.openModal(ModalRemoveVolume)
	}
	return m, nil
}

// -----------------------------------------------------------------------------
// Swarm Tab Handlers
// -----------------------------------------------------------------------------

func (m Model) handleSwarmLogs() (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
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
	return m, nil
}

func (m Model) handleSwarmCopy() (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
		svc := m.swarm.Services[m.swarmCursorIdx]
		m.copyToClipboard(svc.Name)
	}
	return m, nil
}

func (m Model) handleSwarmScaleModal() (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
		svc := m.swarm.Services[m.swarmCursorIdx]
		if strings.EqualFold(svc.Mode, "global") {
			m.showToast("Cannot scale global Swarm service")
			return m, nil
		}
		m.swarmScaleTargetID = svc.ID
		m.swarmScaleTargetName = svc.Name
		m.swarmScaleCurrent = svc.TargetReplicas
		m.swarmScaleDesired = svc.TargetReplicas
		m.openModal(ModalSwarmScale)
	}
	return m, nil
}

func (m Model) handleSwarmQuickScale(delta int) (Model, tea.Cmd) {
	if len(m.swarm.Services) > 0 && m.swarmCursorIdx < len(m.swarm.Services) {
		svc := m.swarm.Services[m.swarmCursorIdx]
		if strings.EqualFold(svc.Mode, "global") {
			m.showToast("Cannot scale global Swarm service")
			return m, nil
		}
		desired := svc.TargetReplicas + delta
		if desired < 0 {
			desired = 0
		}
		m.showToast(fmt.Sprintf("Scaling service %s to %d replicas...", svc.Name, desired))
		return m, scaleSwarmServiceCmd(svc.ID, svc.Name, desired)
	}
	return m, nil
}
