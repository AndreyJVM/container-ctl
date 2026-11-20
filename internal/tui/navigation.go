package tui

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

	if len(m.compose) == 0 {
		m.composeCursorIdx = 0
	} else if m.composeCursorIdx >= len(m.compose) {
		m.composeCursorIdx = len(m.compose) - 1
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

	if len(m.networks) == 0 {
		m.networkCursorIdx = 0
	} else if m.networkCursorIdx >= len(m.networks) {
		m.networkCursorIdx = len(m.networks) - 1
	} else if m.networkCursorIdx < 0 {
		m.networkCursorIdx = 0
	}

	if len(m.volumes) == 0 {
		m.volumeCursorIdx = 0
	} else if m.volumeCursorIdx >= len(m.volumes) {
		m.volumeCursorIdx = len(m.volumes) - 1
	} else if m.volumeCursorIdx < 0 {
		m.volumeCursorIdx = 0
	}
}

// IsSwarmVisible returns true if Swarm tab is explicitly enabled or active on the host.
func (m *Model) IsSwarmVisible() bool {
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
		if len(m.compose) > 0 {
			m.composeCursorIdx = (m.composeCursorIdx + delta) % len(m.compose)
			if m.composeCursorIdx < 0 {
				m.composeCursorIdx += len(m.compose)
			}
		}
	case TabNetworks:
		if len(m.networks) > 0 {
			m.networkCursorIdx = (m.networkCursorIdx + delta) % len(m.networks)
			if m.networkCursorIdx < 0 {
				m.networkCursorIdx += len(m.networks)
			}
		}
	case TabVolumes:
		if len(m.volumes) > 0 {
			m.volumeCursorIdx = (m.volumeCursorIdx + delta) % len(m.volumes)
			if m.volumeCursorIdx < 0 {
				m.volumeCursorIdx += len(m.volumes)
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
