package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/modals"
	tea "github.com/charmbracelet/bubbletea"
)

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
