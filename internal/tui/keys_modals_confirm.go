package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func isModalCancelKey(key string) bool {
	switch key {
	case "esc", "q", "n", "N":
		return true
	default:
		return false
	}
}

func isModalConfirmKey(key string) bool {
	switch key {
	case "y", "Y":
		return true
	default:
		return false
	}
}

func (m Model) handleComposeDownModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.composeDownTarget = ""
		return m, nil
	}
	if isModalConfirmKey(key) {
		target := m.composeDownTarget
		m.closeModal()
		m.composeDownTarget = ""
		m.showToast(fmt.Sprintf("Downing Compose project %s (down)...", target))
		return m, downComposeCmd(target, m.dockerInfo.Containers)
	}
	return m, nil
}

func (m Model) handleRemoveContainerModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeContainerTargetID = ""
		m.removeContainerTargetName = ""
		return m, nil
	}
	if isModalConfirmKey(key) {
		targetID := m.removeContainerTargetID
		targetName := m.removeContainerTargetName
		m.closeModal()
		m.removeContainerTargetID = ""
		m.removeContainerTargetName = ""
		m.showToast(fmt.Sprintf("Removing container %s...", targetName))
		return m, removeContainerCmd(targetID, targetName)
	}
	return m, nil
}

func (m Model) handleRemoveImageModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeImageTargetID = ""
		m.removeImageTargetName = ""
		m.removeImageTargetInUse = false
		return m, nil
	}
	if isModalConfirmKey(key) {
		targetID := m.removeImageTargetID
		targetName := m.removeImageTargetName
		inUse := m.removeImageTargetInUse
		m.closeModal()
		m.removeImageTargetID = ""
		m.removeImageTargetName = ""
		m.removeImageTargetInUse = false
		m.showToast(fmt.Sprintf("Removing image %s...", targetName))
		return m, removeImageCmd(targetID, targetName, inUse)
	}
	return m, nil
}

func (m Model) handleRemoveNetworkModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeNetworkTargetID = ""
		m.removeNetworkTargetName = ""
		m.removeNetworkContainersCount = 0
		return m, nil
	}
	if isModalConfirmKey(key) {
		targetID := m.removeNetworkTargetID
		targetName := m.removeNetworkTargetName
		m.closeModal()
		m.removeNetworkTargetID = ""
		m.removeNetworkTargetName = ""
		m.removeNetworkContainersCount = 0
		m.showToast(fmt.Sprintf("Removing network %s...", targetName))
		return m, removeNetworkCmd(targetID, targetName)
	}
	return m, nil
}

func (m Model) handleRemoveVolumeModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if isModalCancelKey(key) {
		m.closeModal()
		m.removeVolumeTargetName = ""
		m.removeVolumeTargetInUse = false
		return m, nil
	}
	if isModalConfirmKey(key) {
		targetName := m.removeVolumeTargetName
		inUse := m.removeVolumeTargetInUse
		m.closeModal()
		m.removeVolumeTargetName = ""
		m.removeVolumeTargetInUse = false
		m.showToast(fmt.Sprintf("Removing volume %s...", targetName))
		return m, removeVolumeCmd(targetName, inUse)
	}
	return m, nil
}
