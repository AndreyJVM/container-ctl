package tui

import (
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/filter"
)

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

func containerStatusScore(c monitor.ContainerItem) int {
	return filter.ContainerStatusScore(c)
}
