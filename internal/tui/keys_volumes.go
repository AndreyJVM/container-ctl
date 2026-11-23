package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/modals"
	tea "github.com/charmbracelet/bubbletea"
)

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
