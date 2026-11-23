package docker

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/operations"
)

type rawPort struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

func formatPorts(ports []rawPort) string {
	if len(ports) == 0 {
		return ""
	}
	portsList := make([]string, 0, len(ports))
	seen := make(map[string]bool)
	for _, p := range ports {
		var entry string
		if p.PublicPort > 0 {
			if p.IP != "" && p.IP != "0.0.0.0" && p.IP != "::" {
				entry = fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type)
			} else {
				entry = fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type)
			}
		} else if p.PrivatePort > 0 {
			entry = fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		}
		if entry != "" && !seen[entry] {
			seen[entry] = true
			portsList = append(portsList, entry)
		}
	}
	return strings.Join(portsList, ", ")
}

type rawSocketContainer struct {
	ID              string            `json:"Id"`
	Names           []string          `json:"Names"`
	Image           string            `json:"Image"`
	State           string            `json:"State"`
	Status          string            `json:"Status"`
	Ports           []rawPort         `json:"Ports"`
	Labels          map[string]string `json:"Labels"`
	Created         int64             `json:"Created"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
			Gateway   string `json:"Gateway"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
	Mounts []struct {
		Type        string `json:"Type"`
		Name        string `json:"Name"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		Driver      string `json:"Driver"`
		Mode        string `json:"Mode"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
}

func parseRawSocketContainers(rawContainers []rawSocketContainer) []ContainerItem {
	items := make([]ContainerItem, 0, len(rawContainers))
	for _, c := range rawContainers {
		cID := c.ID
		if len(cID) > 12 {
			cID = cID[:12]
		}

		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		composeProject := c.Labels[composeProjectLabel]
		composeService := c.Labels[composeServiceLabel]

		var mountsList []string
		for _, m := range c.Mounts {
			src := m.Source
			if src == "" {
				src = m.Name
			}
			mode := m.Mode
			if mode == "" {
				if m.RW {
					mode = "rw"
				} else {
					mode = "ro"
				}
			}
			if src != "" && m.Destination != "" {
				mountsList = append(mountsList, fmt.Sprintf("%s -> %s (%s)", src, m.Destination, mode))
			} else if m.Destination != "" {
				mountsList = append(mountsList, m.Destination)
			}
		}

		var ipAddr, netName string
		if len(c.NetworkSettings.Networks) > 0 {
			sortedNets := make([]string, 0, len(c.NetworkSettings.Networks))
			for nName := range c.NetworkSettings.Networks {
				sortedNets = append(sortedNets, nName)
			}
			sort.Strings(sortedNets)

			var ips []string
			var nets []string
			for _, nName := range sortedNets {
				nets = append(nets, nName)
				nData := c.NetworkSettings.Networks[nName]
				if nData.IPAddress != "" {
					ips = append(ips, nData.IPAddress)
				}
			}
			ipAddr = strings.Join(ips, ", ")
			netName = strings.Join(nets, ", ")
		}

		createdStr := ""
		if c.Created > 0 {
			createdStr = time.Unix(c.Created, 0).Format(containerTimeFormat)
		}

		items = append(items, ContainerItem{
			ID:             cID,
			Name:           name,
			Image:          c.Image,
			State:          c.State,
			Status:         c.Status,
			Health:         ParseHealthStatus(c.Status),
			Ports:          formatPorts(c.Ports),
			Mounts:         mountsList,
			ComposeProject: composeProject,
			ComposeService: composeService,
			Labels:         c.Labels,
			IPAddress:      ipAddr,
			NetworkName:    netName,
			Created:        createdStr,
		})
	}
	return items
}

func enrichContainersWithSocketMetrics(client *http.Client, containers []ContainerItem) {
	if len(containers) == 0 {
		return
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentStatsFetch)
	for i := range containers {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, cID, state string) {
			defer wg.Done()
			defer func() { <-sem }()
			if state == "running" {
				stats := operations.FetchContainerStats(client, cID)
				containers[idx].CPUPerc = stats.CPUPerc
				containers[idx].MemUsageMB = stats.MemUsageMB
				containers[idx].MemLimitMB = stats.MemLimitMB
				containers[idx].MemPerc = stats.MemPerc
				containers[idx].NetRxBytes = stats.NetRxBytes
				containers[idx].NetTxBytes = stats.NetTxBytes
				containers[idx].BlockReadBytes = stats.BlockReadBytes
				containers[idx].BlockWriteBytes = stats.BlockWriteBytes
			}

			failed, _ := IsContainerFailed(state, containers[idx].Status)
			if failed || containers[idx].Health == HealthStatusUnhealthy {
				if rawLogs, err := GetContainerLogs(cID, 3); err == nil && rawLogs != "" {
					containers[idx].ErrorSnippet = CleanLogSnippet(rawLogs, 3)
				}
			}

			rp, started := operations.FetchContainerInspect(client, cID)
			if rp != "" {
				containers[idx].RestartPolicy = rp
			}
			if started != "" {
				containers[idx].StartedAt = started
			}
		}(i, containers[i].ID, containers[i].State)
	}
	wg.Wait()
}
