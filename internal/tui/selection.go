package tui

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) isContainerSelected(id string) bool {
	if m.selectedContainers == nil {
		return false
	}
	_, ok := m.selectedContainers[id]
	return ok
}

func (m *Model) toggleContainerSelection() {
	if m.selectedContainers == nil {
		m.selectedContainers = make(map[string]struct{})
	}
	containers := m.getFilteredContainers()
	if len(containers) == 0 || m.cursorIdx >= len(containers) {
		return
	}
	c := containers[m.cursorIdx]
	if _, ok := m.selectedContainers[c.ID]; ok {
		delete(m.selectedContainers, c.ID)
	} else {
		m.selectedContainers[c.ID] = struct{}{}
	}
	if m.cursorIdx < len(containers)-1 {
		m.moveCursor(1)
	}
}

func (m *Model) toggleSelectAllContainers() {
	if m.selectedContainers == nil {
		m.selectedContainers = make(map[string]struct{})
	}
	containers := m.getFilteredContainers()
	if len(containers) == 0 {
		return
	}
	allSelected := true
	for _, c := range containers {
		if _, ok := m.selectedContainers[c.ID]; !ok {
			allSelected = false
			break
		}
	}

	if allSelected {
		for _, c := range containers {
			delete(m.selectedContainers, c.ID)
		}
		m.showToast("All visible containers deselected")
	} else {
		for _, c := range containers {
			m.selectedContainers[c.ID] = struct{}{}
		}
		m.showToast(fmt.Sprintf("Selected %d container(s)", len(containers)))
	}
}

func (m *Model) clearSelectedContainers() {
	m.selectedContainers = make(map[string]struct{})
}

func (m Model) getSelectedContainers() []monitor.ContainerItem {
	if len(m.selectedContainers) == 0 {
		return nil
	}
	var selected []monitor.ContainerItem
	for _, c := range m.dockerInfo.Containers {
		if _, ok := m.selectedContainers[c.ID]; ok {
			selected = append(selected, c)
		}
	}
	return selected
}

func (m Model) handleDockerBatchStartStop() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}

	allRunning := true
	for _, c := range selected {
		if c.State != "running" {
			allRunning = false
			break
		}
	}

	if allRunning {
		m.showToast(fmt.Sprintf("Stopping %d selected container(s)...", len(selected)))
		return m, batchStopContainersCmd(selected)
	}

	m.showToast(fmt.Sprintf("Starting %d selected container(s)...", len(selected)))
	return m, batchStartContainersCmd(selected)
}

func (m Model) handleDockerBatchRestart() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}
	m.showToast(fmt.Sprintf("Restarting %d selected container(s)...", len(selected)))
	return m, batchRestartContainersCmd(selected)
}

func (m Model) handleDockerBatchKill() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}
	m.showToast(fmt.Sprintf("Force killing %d selected container(s)...", len(selected)))
	return m, batchKillContainersCmd(selected)
}

func (m Model) handleDockerBatchRemoveModal() (Model, tea.Cmd) {
	selected := m.getSelectedContainers()
	if len(selected) == 0 {
		return m, nil
	}
	m.removeContainerBatch = selected
	var names []string
	for _, c := range selected {
		names = append(names, c.Name)
	}
	m.removeContainerTargetName = strings.Join(names, ", ")
	m.removeContainerTargetID = fmt.Sprintf("%d containers", len(selected))
	m.openModal(ModalRemoveContainer)
	return m, nil
}

// -----------------------------------------------------------------------------
// Images Multi-Select
// -----------------------------------------------------------------------------

func (m *Model) toggleImageSelection() {
	if m.selectedImages == nil {
		m.selectedImages = make(map[string]struct{})
	}
	images := m.getFilteredImages()
	if len(images) == 0 || m.imageCursorIdx >= len(images) {
		return
	}
	img := images[m.imageCursorIdx]
	if _, ok := m.selectedImages[img.ID]; ok {
		delete(m.selectedImages, img.ID)
	} else {
		m.selectedImages[img.ID] = struct{}{}
	}
	if m.imageCursorIdx < len(images)-1 {
		m.moveCursor(1)
	}
}

func (m *Model) toggleSelectAllImages() {
	if m.selectedImages == nil {
		m.selectedImages = make(map[string]struct{})
	}
	images := m.getFilteredImages()
	if len(images) == 0 {
		return
	}
	allSelected := true
	for _, img := range images {
		if _, ok := m.selectedImages[img.ID]; !ok {
			allSelected = false
			break
		}
	}
	if allSelected {
		for _, img := range images {
			delete(m.selectedImages, img.ID)
		}
		m.showToast("All visible images deselected")
	} else {
		for _, img := range images {
			m.selectedImages[img.ID] = struct{}{}
		}
		m.showToast(fmt.Sprintf("Selected %d image(s)", len(images)))
	}
}

func (m *Model) clearSelectedImages() {
	m.selectedImages = make(map[string]struct{})
}

func (m Model) getSelectedImages() []monitor.DockerImage {
	if len(m.selectedImages) == 0 {
		return nil
	}
	var selected []monitor.DockerImage
	for _, img := range m.images {
		if _, ok := m.selectedImages[img.ID]; ok {
			selected = append(selected, img)
		}
	}
	return selected
}

func (m Model) handleImagesBatchRemoveModal() (Model, tea.Cmd) {
	selected := m.getSelectedImages()
	if len(selected) == 0 {
		return m, nil
	}
	m.removeImageBatch = selected
	var names []string
	anyInUse := false
	for _, img := range selected {
		names = append(names, img.Repository)
		if img.InUse {
			anyInUse = true
		}
	}
	m.removeImageTargetName = strings.Join(names, ", ")
	m.removeImageTargetID = fmt.Sprintf("%d images", len(selected))
	m.removeImageTargetInUse = anyInUse
	m.openModal(ModalRemoveImage)
	return m, nil
}

// -----------------------------------------------------------------------------
// Networks Multi-Select
// -----------------------------------------------------------------------------

func (m *Model) toggleNetworkSelection() {
	if m.selectedNetworks == nil {
		m.selectedNetworks = make(map[string]struct{})
	}
	networks := m.getFilteredNetworks()
	if len(networks) == 0 || m.networkCursorIdx >= len(networks) {
		return
	}
	net := networks[m.networkCursorIdx]
	if _, ok := m.selectedNetworks[net.ID]; ok {
		delete(m.selectedNetworks, net.ID)
	} else {
		m.selectedNetworks[net.ID] = struct{}{}
	}
	if m.networkCursorIdx < len(networks)-1 {
		m.moveCursor(1)
	}
}

func (m *Model) toggleSelectAllNetworks() {
	if m.selectedNetworks == nil {
		m.selectedNetworks = make(map[string]struct{})
	}
	networks := m.getFilteredNetworks()
	if len(networks) == 0 {
		return
	}
	allSelected := true
	for _, net := range networks {
		if _, ok := m.selectedNetworks[net.ID]; !ok {
			allSelected = false
			break
		}
	}
	if allSelected {
		for _, net := range networks {
			delete(m.selectedNetworks, net.ID)
		}
		m.showToast("All visible networks deselected")
	} else {
		for _, net := range networks {
			m.selectedNetworks[net.ID] = struct{}{}
		}
		m.showToast(fmt.Sprintf("Selected %d network(s)", len(networks)))
	}
}

func (m *Model) clearSelectedNetworks() {
	m.selectedNetworks = make(map[string]struct{})
}

func (m Model) getSelectedNetworks() []monitor.DockerNetwork {
	if len(m.selectedNetworks) == 0 {
		return nil
	}
	var selected []monitor.DockerNetwork
	for _, net := range m.networks {
		if _, ok := m.selectedNetworks[net.ID]; ok {
			selected = append(selected, net)
		}
	}
	return selected
}

func (m Model) handleNetworksBatchRemoveModal() (Model, tea.Cmd) {
	selected := m.getSelectedNetworks()
	if len(selected) == 0 {
		return m, nil
	}
	m.removeNetworkBatch = selected
	var names []string
	totalContainers := 0
	for _, net := range selected {
		names = append(names, net.Name)
		totalContainers += net.Containers
	}
	m.removeNetworkTargetName = strings.Join(names, ", ")
	m.removeNetworkTargetID = fmt.Sprintf("%d networks", len(selected))
	m.removeNetworkContainersCount = totalContainers
	m.openModal(ModalRemoveNetwork)
	return m, nil
}

// -----------------------------------------------------------------------------
// Volumes Multi-Select
// -----------------------------------------------------------------------------

func (m *Model) toggleVolumeSelection() {
	if m.selectedVolumes == nil {
		m.selectedVolumes = make(map[string]struct{})
	}
	volumes := m.getFilteredVolumes()
	if len(volumes) == 0 || m.volumeCursorIdx >= len(volumes) {
		return
	}
	vol := volumes[m.volumeCursorIdx]
	if _, ok := m.selectedVolumes[vol.Name]; ok {
		delete(m.selectedVolumes, vol.Name)
	} else {
		m.selectedVolumes[vol.Name] = struct{}{}
	}
	if m.volumeCursorIdx < len(volumes)-1 {
		m.moveCursor(1)
	}
}

func (m *Model) toggleSelectAllVolumes() {
	if m.selectedVolumes == nil {
		m.selectedVolumes = make(map[string]struct{})
	}
	volumes := m.getFilteredVolumes()
	if len(volumes) == 0 {
		return
	}
	allSelected := true
	for _, vol := range volumes {
		if _, ok := m.selectedVolumes[vol.Name]; !ok {
			allSelected = false
			break
		}
	}
	if allSelected {
		for _, vol := range volumes {
			delete(m.selectedVolumes, vol.Name)
		}
		m.showToast("All visible volumes deselected")
	} else {
		for _, vol := range volumes {
			m.selectedVolumes[vol.Name] = struct{}{}
		}
		m.showToast(fmt.Sprintf("Selected %d volume(s)", len(volumes)))
	}
}

func (m *Model) clearSelectedVolumes() {
	m.selectedVolumes = make(map[string]struct{})
}

func (m Model) getSelectedVolumes() []monitor.DockerVolume {
	if len(m.selectedVolumes) == 0 {
		return nil
	}
	var selected []monitor.DockerVolume
	for _, vol := range m.volumes {
		if _, ok := m.selectedVolumes[vol.Name]; ok {
			selected = append(selected, vol)
		}
	}
	return selected
}

func (m Model) handleVolumesBatchRemoveModal() (Model, tea.Cmd) {
	selected := m.getSelectedVolumes()
	if len(selected) == 0 {
		return m, nil
	}
	m.removeVolumeBatch = selected
	var names []string
	anyInUse := false
	for _, vol := range selected {
		names = append(names, vol.Name)
		if vol.RefCount > 0 || len(vol.Containers) > 0 {
			anyInUse = true
		}
	}
	m.removeVolumeTargetName = strings.Join(names, ", ")
	m.removeVolumeTargetInUse = anyInUse
	m.openModal(ModalRemoveVolume)
	return m, nil
}
