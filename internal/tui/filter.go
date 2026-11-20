package tui

import (
	"sort"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func containerStatusScore(c monitor.ContainerItem) int {
	switch {
	case c.State == "running" && c.Health == "unhealthy":
		return 1
	case c.State == "running" && c.Health == "starting":
		return 2
	case c.State == "running":
		return 3
	case c.State == "restarting":
		return 4
	case c.State == "paused":
		return 5
	default:
		return 6
	}
}

func (m Model) getFilteredContainers() []monitor.ContainerItem {
	var list []monitor.ContainerItem
	trimmedQuery := strings.TrimSpace(m.filterQuery)
	if trimmedQuery == "" {
		list = make([]monitor.ContainerItem, len(m.dockerInfo.Containers))
		copy(list, m.dockerInfo.Containers)
	} else {
		q := strings.ToLower(trimmedQuery)
		for _, c := range m.dockerInfo.Containers {
			if strings.Contains(strings.ToLower(c.Name), q) ||
				strings.Contains(strings.ToLower(c.Image), q) ||
				strings.Contains(strings.ToLower(c.ID), q) ||
				strings.Contains(strings.ToLower(c.ComposeProject), q) ||
				strings.Contains(strings.ToLower(c.Health), q) {
				list = append(list, c)
			}
		}
	}

	switch m.containerSortMode {
	case SortCPU:
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].CPUPerc != list[j].CPUPerc {
				return list[i].CPUPerc > list[j].CPUPerc
			}
			return list[i].Name < list[j].Name
		})
	case SortMem:
		sort.SliceStable(list, func(i, j int) bool {
			if list[i].MemUsageMB != list[j].MemUsageMB {
				return list[i].MemUsageMB > list[j].MemUsageMB
			}
			return list[i].Name < list[j].Name
		})
	case SortName:
		sort.SliceStable(list, func(i, j int) bool {
			return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
		})
	case SortStatus:
		sort.SliceStable(list, func(i, j int) bool {
			si, sj := containerStatusScore(list[i]), containerStatusScore(list[j])
			if si != sj {
				return si < sj
			}
			return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
		})
	}

	return list
}

func (m Model) getFilteredImages() []monitor.DockerImage {
	trimmedQuery := strings.TrimSpace(strings.ToLower(m.filterQuery))
	var filtered []monitor.DockerImage
	for _, img := range m.images {
		if m.onlyUnusedImages && img.InUse {
			continue
		}
		if trimmedQuery != "" {
			matchesQuery := strings.Contains(strings.ToLower(img.ID), trimmedQuery) ||
				strings.Contains(strings.ToLower(img.Repository), trimmedQuery) ||
				strings.Contains(strings.ToLower(img.Tag), trimmedQuery) ||
				strings.Contains(strings.ToLower(img.Digest), trimmedQuery) ||
				(trimmedQuery == "unused" && !img.InUse) ||
				(trimmedQuery == "inuse" && img.InUse) ||
				(trimmedQuery == "in use" && img.InUse) ||
				(trimmedQuery == "dangling" && img.Dangling)
			if !matchesQuery {
				continue
			}
		}
		filtered = append(filtered, img)
	}
	return filtered
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
