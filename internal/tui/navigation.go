package tui

import (
	"github.com/AndreyJVM/container-ctl/internal/module"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/filter"
)

// clamp keeps idx within [0, count-1]. If count <= 0 or idx < 0, it returns 0.
func clamp(idx, count int) int {
	if count <= 0 || idx < 0 {
		return 0
	}
	if idx >= count {
		return count - 1
	}
	return idx
}

// cycleIndex offsets idx by delta with cyclic wrapping within [0, count-1].
func cycleIndex(idx, delta, count int) int {
	if count <= 0 {
		return 0
	}
	newIdx := (idx + delta) % count
	if newIdx < 0 {
		newIdx += count
	}
	return newIdx
}

// clampCursor ensures all tab cursor indices remain within bounds after updates or filtering.
func (m *Model) clampCursor() {
	m.cursorIdx = clamp(m.cursorIdx, len(m.getFilteredContainers()))
	m.imageCursorIdx = clamp(m.imageCursorIdx, len(m.getFilteredImages()))
	m.composeCursorIdx = clamp(m.composeCursorIdx, len(m.getFilteredCompose()))
	m.networkCursorIdx = clamp(m.networkCursorIdx, len(m.getFilteredNetworks()))
	m.volumeCursorIdx = clamp(m.volumeCursorIdx, len(m.getFilteredVolumes()))
	m.swarmCursorIdx = clamp(m.swarmCursorIdx, len(m.swarm.Services))
}

// TabIDFromModuleID maps a module ID to its legacy TabID enum.
func TabIDFromModuleID(id module.ID) (TabID, bool) {
	switch id {
	case module.IDContainers:
		return TabDocker, true
	case module.IDImages:
		return TabImages, true
	case module.IDCompose:
		return TabCompose, true
	case module.IDNetworks:
		return TabNetworks, true
	case module.IDVolumes:
		return TabVolumes, true
	case module.IDSwarm:
		return TabSwarm, true
	default:
		return 0, false
	}
}

// ModuleIDFromTabID maps a TabID to its corresponding module ID.
func ModuleIDFromTabID(tab TabID) (module.ID, bool) {
	switch tab {
	case TabDocker:
		return module.IDContainers, true
	case TabImages:
		return module.IDImages, true
	case TabCompose:
		return module.IDCompose, true
	case TabNetworks:
		return module.IDNetworks, true
	case TabVolumes:
		return module.IDVolumes, true
	case TabSwarm:
		return module.IDSwarm, true
	default:
		return "", false
	}
}

// visibleModules returns the slice of currently visible modules from the registry.
func (m *Model) visibleModules() []module.VisibleModule {
	return m.Registry().VisibleModules(m.moduleContext())
}

// IsSwarmVisible returns true if Swarm tab is explicitly enabled or active on the host.
func (m *Model) IsSwarmVisible() bool {
	if m.registry != nil {
		return m.registry.IsModuleVisible(module.IDSwarm, m.moduleContext())
	}
	return m.showSwarm || m.swarm.IsActive
}

// SetShowSwarm configures whether the Swarm tab is forcibly shown even when inactive.
func (m *Model) SetShowSwarm(show bool) {
	m.showSwarm = show
}

func (m *Model) maxTab() TabID {
	if m.IsSwarmVisible() {
		return TabSwarm
	}
	return TabVolumes
}

// cycleTab advances or retreats the active tab cyclically by delta (+1 for next, -1 for prev).
func (m *Model) cycleTab(delta int) {
	visible := m.visibleModules()
	if len(visible) == 0 {
		maxT := m.maxTab()
		if delta > 0 {
			m.activeTab++
			if m.activeTab > maxT {
				m.activeTab = TabDocker
			}
		} else {
			m.activeTab--
			if m.activeTab < TabDocker {
				m.activeTab = maxT
			}
		}
		return
	}

	currIdx := -1
	for i, v := range visible {
		if tabID, ok := TabIDFromModuleID(v.ID); ok && tabID == m.activeTab {
			currIdx = i
			break
		}
	}
	if currIdx == -1 {
		if tabID, ok := TabIDFromModuleID(visible[0].ID); ok {
			m.activeTab = tabID
		}
		return
	}

	nextIdx := cycleIndex(currIdx, delta, len(visible))
	if tabID, ok := TabIDFromModuleID(visible[nextIdx].ID); ok {
		m.activeTab = tabID
	}
}

// nextTab switches to the next tab cyclically based on visible tabs.
func (m *Model) nextTab() {
	m.cycleTab(1)
}

// prevTab switches to the previous tab cyclically based on visible tabs.
func (m *Model) prevTab() {
	m.cycleTab(-1)
}

// setTabByIndex switches directly to the module at 1-based index if visible.
func (m *Model) setTabByIndex(index int) bool {
	visible := m.visibleModules()
	if index < 1 || index > len(visible) {
		return false
	}
	tabID, mapped := TabIDFromModuleID(visible[index-1].ID)
	if mapped {
		m.activeTab = tabID
		return true
	}
	return false
}

// setTab switches directly to the specified tab if within valid range.
func (m *Model) setTab(tab TabID) {
	if tab >= TabDocker && tab <= TabCount {
		m.activeTab = tab
	}
}

// activeListLength returns the item count of the currently active tab.
func (m Model) activeListLength() int {
	switch m.activeTab {
	case TabDocker:
		return len(m.getFilteredContainers())
	case TabImages:
		return len(m.getFilteredImages())
	case TabCompose:
		return len(m.getFilteredCompose())
	case TabNetworks:
		return len(m.getFilteredNetworks())
	case TabVolumes:
		return len(m.getFilteredVolumes())
	case TabSwarm:
		return len(m.swarm.Services)
	default:
		return 0
	}
}

// ActiveCursor returns the cursor index of the currently active tab.
func (m Model) ActiveCursor() int {
	return m.activeCursor()
}

// activeCursor returns the cursor index of the currently active tab.
func (m Model) activeCursor() int {
	switch m.activeTab {
	case TabDocker:
		return m.cursorIdx
	case TabImages:
		return m.imageCursorIdx
	case TabCompose:
		return m.composeCursorIdx
	case TabNetworks:
		return m.networkCursorIdx
	case TabVolumes:
		return m.volumeCursorIdx
	case TabSwarm:
		return m.swarmCursorIdx
	default:
		return 0
	}
}

// setActiveCursor sets the cursor index of the currently active tab.
func (m *Model) setActiveCursor(idx int) {
	switch m.activeTab {
	case TabDocker:
		m.cursorIdx = idx
	case TabImages:
		m.imageCursorIdx = idx
	case TabCompose:
		m.composeCursorIdx = idx
	case TabNetworks:
		m.networkCursorIdx = idx
	case TabVolumes:
		m.volumeCursorIdx = idx
	case TabSwarm:
		m.swarmCursorIdx = idx
	}
}

// moveCursor adjusts cursor index for the active tab list by delta with cyclic wrapping.
func (m *Model) moveCursor(delta int) {
	count := m.activeListLength()
	if count > 0 {
		m.setActiveCursor(cycleIndex(m.activeCursor(), delta, count))
	}
}

// cursorToStart sets the cursor index of the active tab to 0.
func (m *Model) cursorToStart() {
	m.setActiveCursor(0)
}

// cursorToEnd sets the cursor index of the active tab to the last item.
func (m *Model) cursorToEnd() {
	count := m.activeListLength()
	if count > 0 {
		m.setActiveCursor(count - 1)
	}
}

// -----------------------------------------------------------------------------
// Filtered Data Getters
// -----------------------------------------------------------------------------

func (m Model) getFilteredContainers() []monitor.ContainerItem {
	return filter.Containers(m.dockerInfo.Containers, m.filterQuery, m.containerSortMode)
}

func (m Model) getFilteredImages() []monitor.DockerImage {
	return filter.Images(m.images, m.filterQuery, m.onlyUnusedImages)
}

func (m Model) getFilteredCompose() []monitor.ComposeProject {
	return filter.Compose(m.compose, m.filterQuery)
}

func (m Model) getFilteredNetworks() []monitor.DockerNetwork {
	return filter.Networks(m.networks, m.filterQuery)
}

func (m Model) getFilteredVolumes() []monitor.DockerVolume {
	return filter.Volumes(m.volumes, m.filterQuery)
}

// IsOnlyUnusedImages returns true if the image list is filtered to only unused images.
func (m Model) IsOnlyUnusedImages() bool {
	return m.onlyUnusedImages
}

// SetOnlyUnusedImages enables or disables unused-only filtering on the Images tab.
func (m *Model) SetOnlyUnusedImages(val bool) {
	m.onlyUnusedImages = val
	m.clampCursor()
}

// SetContainerSortMode sets the active container sort mode.
func (m *Model) SetContainerSortMode(mode int) {
	if mode >= SortDefault && mode < sortModeCount {
		m.containerSortMode = mode
		m.clampCursor()
	}
}

// ContainerSortMode returns the currently active container sort mode.
func (m Model) ContainerSortMode() int {
	return m.containerSortMode
}

// FilteredContainers returns currently filtered and sorted containers.
func (m Model) FilteredContainers() []monitor.ContainerItem {
	return m.getFilteredContainers()
}

// FilteredImages returns currently filtered images.
func (m Model) FilteredImages() []monitor.DockerImage {
	return m.getFilteredImages()
}

// FilteredNetworks returns currently filtered networks.
func (m Model) FilteredNetworks() []monitor.DockerNetwork {
	return m.getFilteredNetworks()
}

// FilteredVolumes returns currently filtered volumes.
func (m Model) FilteredVolumes() []monitor.DockerVolume {
	return m.getFilteredVolumes()
}

func containerStatusScore(c monitor.ContainerItem) int {
	return filter.ContainerStatusScore(c)
}
