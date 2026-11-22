package tui

import (
	"github.com/AndreyJVM/container-ctl/internal/module"
)

// clampCursor ensures all tab cursor indices remain within bounds after updates or filtering.
func (m *Model) clampCursor() {
	containers := m.getFilteredContainers()
	if len(containers) == 0 {
		m.cursorIdx = 0
	} else if m.cursorIdx >= len(containers) {
		m.cursorIdx = len(containers) - 1
	} else if m.cursorIdx < 0 {
		m.cursorIdx = 0
	}

	images := m.getFilteredImages()
	if len(images) == 0 {
		m.imageCursorIdx = 0
	} else if m.imageCursorIdx >= len(images) {
		m.imageCursorIdx = len(images) - 1
	} else if m.imageCursorIdx < 0 {
		m.imageCursorIdx = 0
	}

	compose := m.getFilteredCompose()
	if len(compose) == 0 {
		m.composeCursorIdx = 0
	} else if m.composeCursorIdx >= len(compose) {
		m.composeCursorIdx = len(compose) - 1
	} else if m.composeCursorIdx < 0 {
		m.composeCursorIdx = 0
	}

	if len(m.swarm.Services) == 0 {
		m.swarmCursorIdx = 0
	} else if m.swarmCursorIdx >= len(m.swarm.Services) {
		m.swarmCursorIdx = len(m.swarm.Services) - 1
	} else if m.swarmCursorIdx < 0 {
		m.swarmCursorIdx = 0
	}

	networks := m.getFilteredNetworks()
	if len(networks) == 0 {
		m.networkCursorIdx = 0
	} else if m.networkCursorIdx >= len(networks) {
		m.networkCursorIdx = len(networks) - 1
	} else if m.networkCursorIdx < 0 {
		m.networkCursorIdx = 0
	}

	volumes := m.getFilteredVolumes()
	if len(volumes) == 0 {
		m.volumeCursorIdx = 0
	} else if m.volumeCursorIdx >= len(volumes) {
		m.volumeCursorIdx = len(volumes) - 1
	} else if m.volumeCursorIdx < 0 {
		m.volumeCursorIdx = 0
	}
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

// nextTab switches to the next tab cyclically based on visible tabs.
func (m *Model) nextTab() {
	m.activeTab++
	if m.activeTab > m.maxTab() {
		m.activeTab = TabDocker
	}
}

// prevTab switches to the previous tab cyclically based on visible tabs.
func (m *Model) prevTab() {
	m.activeTab--
	if m.activeTab < TabDocker {
		m.activeTab = m.maxTab()
	}
}

// setTab switches directly to the specified tab if within valid range.
func (m *Model) setTab(tab TabID) {
	if tab >= TabDocker && tab <= TabCount {
		m.activeTab = tab
	}
}

// moveCursor adjusts cursor index for the active tab list by delta with cyclic wrapping.
func (m *Model) moveCursor(delta int) {
	switch m.activeTab {
	case TabDocker:
		containers := m.getFilteredContainers()
		if len(containers) > 0 {
			m.cursorIdx = (m.cursorIdx + delta) % len(containers)
			if m.cursorIdx < 0 {
				m.cursorIdx += len(containers)
			}
		}
	case TabImages:
		images := m.getFilteredImages()
		if len(images) > 0 {
			m.imageCursorIdx = (m.imageCursorIdx + delta) % len(images)
			if m.imageCursorIdx < 0 {
				m.imageCursorIdx += len(images)
			}
		}
	case TabCompose:
		compose := m.getFilteredCompose()
		if len(compose) > 0 {
			m.composeCursorIdx = (m.composeCursorIdx + delta) % len(compose)
			if m.composeCursorIdx < 0 {
				m.composeCursorIdx += len(compose)
			}
		}
	case TabNetworks:
		networks := m.getFilteredNetworks()
		if len(networks) > 0 {
			m.networkCursorIdx = (m.networkCursorIdx + delta) % len(networks)
			if m.networkCursorIdx < 0 {
				m.networkCursorIdx += len(networks)
			}
		}
	case TabVolumes:
		volumes := m.getFilteredVolumes()
		if len(volumes) > 0 {
			m.volumeCursorIdx = (m.volumeCursorIdx + delta) % len(volumes)
			if m.volumeCursorIdx < 0 {
				m.volumeCursorIdx += len(volumes)
			}
		}
	case TabSwarm:
		if len(m.swarm.Services) > 0 {
			m.swarmCursorIdx = (m.swarmCursorIdx + delta) % len(m.swarm.Services)
			if m.swarmCursorIdx < 0 {
				m.swarmCursorIdx += len(m.swarm.Services)
			}
		}
	}
}

// cursorToStart sets the cursor index of the active tab to 0.
func (m *Model) cursorToStart() {
	switch m.activeTab {
	case TabDocker:
		m.cursorIdx = 0
	case TabImages:
		m.imageCursorIdx = 0
	case TabCompose:
		m.composeCursorIdx = 0
	case TabNetworks:
		m.networkCursorIdx = 0
	case TabVolumes:
		m.volumeCursorIdx = 0
	case TabSwarm:
		m.swarmCursorIdx = 0
	}
}

// cursorToEnd sets the cursor index of the active tab to the last item.
func (m *Model) cursorToEnd() {
	switch m.activeTab {
	case TabDocker:
		if n := len(m.getFilteredContainers()); n > 0 {
			m.cursorIdx = n - 1
		}
	case TabImages:
		if n := len(m.getFilteredImages()); n > 0 {
			m.imageCursorIdx = n - 1
		}
	case TabCompose:
		if n := len(m.getFilteredCompose()); n > 0 {
			m.composeCursorIdx = n - 1
		}
	case TabNetworks:
		if n := len(m.getFilteredNetworks()); n > 0 {
			m.networkCursorIdx = n - 1
		}
	case TabVolumes:
		if n := len(m.getFilteredVolumes()); n > 0 {
			m.volumeCursorIdx = n - 1
		}
	case TabSwarm:
		if n := len(m.swarm.Services); n > 0 {
			m.swarmCursorIdx = n - 1
		}
	}
}
