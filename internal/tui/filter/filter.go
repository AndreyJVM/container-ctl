package filter

import (
	"sort"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

// Sort mode constants matching TUI modes.
const (
	SortDefault = iota
	SortCPU
	SortMem
	SortName
	SortStatus
)

func ContainerStatusScore(c monitor.ContainerItem) int {
	if failed, _ := monitor.IsContainerFailed(c.State, c.Status); failed {
		return 1
	}
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

// Containers filters and sorts container items based on query and sort mode.
func Containers(containers []monitor.ContainerItem, query string, sortMode int) []monitor.ContainerItem {
	var list []monitor.ContainerItem
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		list = make([]monitor.ContainerItem, len(containers))
		copy(list, containers)
	} else {
		q := strings.ToLower(trimmedQuery)
		for _, c := range containers {
			if strings.Contains(strings.ToLower(c.Name), q) ||
				strings.Contains(strings.ToLower(c.Image), q) ||
				strings.Contains(strings.ToLower(c.ID), q) ||
				strings.Contains(strings.ToLower(c.State), q) ||
				strings.Contains(strings.ToLower(c.Status), q) ||
				strings.Contains(strings.ToLower(c.Ports), q) ||
				strings.Contains(strings.ToLower(c.ComposeProject), q) ||
				strings.Contains(strings.ToLower(c.ComposeService), q) ||
				strings.Contains(strings.ToLower(c.Health), q) {
				list = append(list, c)
			}
		}
	}

	switch sortMode {
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
			si, sj := ContainerStatusScore(list[i]), ContainerStatusScore(list[j])
			if si != sj {
				return si < sj
			}
			return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
		})
	}

	return list
}

// Images filters images by query string and in-use/unused status.
func Images(images []monitor.DockerImage, query string, onlyUnused bool) []monitor.DockerImage {
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	var filtered []monitor.DockerImage
	for _, img := range images {
		if onlyUnused && img.InUse {
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

// Compose filters compose projects and their services by query string.
func Compose(projects []monitor.ComposeProject, query string) []monitor.ComposeProject {
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if trimmedQuery == "" {
		return projects
	}

	var filtered []monitor.ComposeProject
	for _, p := range projects {
		matches := strings.Contains(strings.ToLower(p.Name), trimmedQuery) ||
			strings.Contains(strings.ToLower(p.Status), trimmedQuery) ||
			strings.Contains(strings.ToLower(p.StatusDisplay), trimmedQuery)

		if !matches {
			for _, s := range p.Services {
				if strings.Contains(strings.ToLower(s.Name), trimmedQuery) ||
					strings.Contains(strings.ToLower(s.Image), trimmedQuery) ||
					strings.Contains(strings.ToLower(s.State), trimmedQuery) {
					matches = true
					break
				}
			}
		}

		if matches {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// Networks filters networks by query string.
func Networks(networks []monitor.DockerNetwork, query string) []monitor.DockerNetwork {
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if trimmedQuery == "" {
		return networks
	}

	var filtered []monitor.DockerNetwork
	for _, n := range networks {
		if strings.Contains(strings.ToLower(n.Name), trimmedQuery) ||
			strings.Contains(strings.ToLower(n.ID), trimmedQuery) ||
			strings.Contains(strings.ToLower(n.Driver), trimmedQuery) ||
			strings.Contains(strings.ToLower(n.Scope), trimmedQuery) ||
			strings.Contains(strings.ToLower(n.Subnet), trimmedQuery) ||
			strings.Contains(strings.ToLower(n.Gateway), trimmedQuery) ||
			strings.Contains(strings.ToLower(n.ComposeProject), trimmedQuery) {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

// Volumes filters volumes by query string.
func Volumes(volumes []monitor.DockerVolume, query string) []monitor.DockerVolume {
	trimmedQuery := strings.TrimSpace(strings.ToLower(query))
	if trimmedQuery == "" {
		return volumes
	}

	var filtered []monitor.DockerVolume
	for _, v := range volumes {
		if strings.Contains(strings.ToLower(v.Name), trimmedQuery) ||
			strings.Contains(strings.ToLower(v.Driver), trimmedQuery) ||
			strings.Contains(strings.ToLower(v.Scope), trimmedQuery) ||
			strings.Contains(strings.ToLower(v.Mountpoint), trimmedQuery) ||
			strings.Contains(strings.ToLower(v.ComposeProject), trimmedQuery) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
