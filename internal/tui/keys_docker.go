package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
)

var browserOpener = defaultBrowserOpener

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

// openBrowser attempts to open the given URL in the default browser.
func openBrowser(url string) bool {
	return browserOpener(url)
}

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
		m.showToast(fmt.Sprintf("Cannot connect: container %s is not running", c.Name))
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
		cmdText := fmt.Sprintf("docker exec -it %s sh", c.Name)
		if err := clipboard.Write(cmdText); err != nil {
			m.showToast(fmt.Sprintf("Clipboard error: %v", err))
		} else {
			m.showToast(fmt.Sprintf("Copied to clipboard: %s", cmdText))
		}
	}
	return m, nil
}

func (m Model) handleDockerOpenURL() (Model, tea.Cmd) {
	containers := m.getFilteredContainers()
	if len(containers) > 0 && m.cursorIdx < len(containers) {
		c := containers[m.cursorIdx]
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
