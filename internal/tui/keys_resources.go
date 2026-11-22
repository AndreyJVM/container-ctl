package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleNetworksInspect() (Model, tea.Cmd) {
	networks := m.getFilteredNetworks()
	if len(networks) > 0 && m.networkCursorIdx < len(networks) {
		net := networks[m.networkCursorIdx]
		inspectStr, err := monitor.GetNetworkInspectJSON(net.ID)
		if err != nil || strings.TrimSpace(inspectStr) == "" {
			if b, mErr := json.MarshalIndent(net, "", "  "); mErr == nil {
				inspectStr = string(b)
			}
		}
		if strings.TrimSpace(inspectStr) == "" {
			m.showToast("Inspect error: unable to inspect network " + net.Name)
		} else {
			m.inspectTarget = monitor.ContainerDetail{}
			m.inspectCustomTitle = fmt.Sprintf("NETWORK INSPECTION: %s", net.Name)
			m.inspectCustomLines = strings.Split(strings.TrimSpace(inspectStr), "\n")
			m.inspectCopyID = net.ID
			if m.inspectCopyID == "" {
				m.inspectCopyID = net.Name
			}
			m.inspectScroll = 0
			m.openModal(ModalInspect)
		}
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

func (m Model) handleVolumesInspect() (Model, tea.Cmd) {
	volumes := m.getFilteredVolumes()
	if len(volumes) > 0 && m.volumeCursorIdx < len(volumes) {
		vol := volumes[m.volumeCursorIdx]
		inspectStr, err := monitor.GetVolumeInspectJSON(vol.Name)
		if err != nil || strings.TrimSpace(inspectStr) == "" {
			if b, mErr := json.MarshalIndent(vol, "", "  "); mErr == nil {
				inspectStr = string(b)
			}
		}
		if strings.TrimSpace(inspectStr) == "" {
			m.showToast("Inspect error: unable to inspect volume " + vol.Name)
		} else {
			m.inspectTarget = monitor.ContainerDetail{}
			m.inspectCustomTitle = fmt.Sprintf("VOLUME INSPECTION: %s", vol.Name)
			m.inspectCustomLines = strings.Split(strings.TrimSpace(inspectStr), "\n")
			m.inspectCopyID = vol.Name
			m.inspectScroll = 0
			m.openModal(ModalInspect)
		}
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
		return m, scaleSwarmServiceCmd(svc.ID, svc.Name, desired)
	}
	return m, nil
}
