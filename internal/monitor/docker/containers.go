package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ParseHealthStatus extracts a standardized healthcheck status ("healthy", "unhealthy", "starting" or "")
// from Docker's State.Health.Status or container Status string.
func ParseHealthStatus(val string) string {
	s := strings.ToLower(strings.TrimSpace(val))
	if s == "" {
		return ""
	}
	switch s {
	case HealthStatusHealthy:
		return HealthStatusHealthy
	case HealthStatusUnhealthy:
		return HealthStatusUnhealthy
	case HealthStatusStarting:
		return HealthStatusStarting
	}

	if strings.Contains(s, "("+HealthStatusHealthy+")") {
		return HealthStatusHealthy
	}
	if strings.Contains(s, "("+HealthStatusUnhealthy+")") {
		return HealthStatusUnhealthy
	}
	if strings.Contains(s, "(health: "+HealthStatusStarting+")") || (strings.Contains(s, HealthStatusStarting) && strings.Contains(s, "health")) {
		return HealthStatusStarting
	}

	return ""
}

// ParseExitCode extracts whether an exit code is present and its integer value
// from Docker status strings like "Exited (1) 2 minutes ago" or "Exited (0)".
func ParseExitCode(status string) (bool, int) {
	lower := strings.ToLower(status)
	idx := strings.Index(lower, "exited (")
	if idx == -1 {
		return false, 0
	}
	rest := status[idx+len("exited ("):]
	end := strings.IndexByte(rest, ')')
	if end == -1 {
		return false, 0
	}
	code, err := strconv.Atoi(strings.TrimSpace(rest[:end]))
	if err != nil {
		return false, 0
	}
	return true, code
}

// IsContainerFailed returns true and the exit code if the container is in an error state (dead or non-zero exit code).
func IsContainerFailed(state, status string) (bool, int) {
	if strings.ToLower(state) == "dead" {
		return true, -1
	}
	hasCode, code := ParseExitCode(status)
	if hasCode && code != 0 {
		return true, code
	}
	return false, 0
}

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
			entry = fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type)
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
			var ips []string
			var nets []string
			for nName, nData := range c.NetworkSettings.Networks {
				nets = append(nets, nName)
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
				stats := fetchContainerStats(client, cID)
				containers[idx].CPUPerc = stats.CPUPerc
				containers[idx].MemUsageMB = stats.MemUsageMB
				containers[idx].MemLimitMB = stats.MemLimitMB
				containers[idx].MemPerc = stats.MemPerc
				containers[idx].NetRxBytes = stats.NetRxBytes
				containers[idx].NetTxBytes = stats.NetTxBytes
				containers[idx].BlockReadBytes = stats.BlockReadBytes
				containers[idx].BlockWriteBytes = stats.BlockWriteBytes
			} else {
				if failed, _ := IsContainerFailed(state, containers[idx].Status); failed {
					if rawLogs, err := GetContainerLogs(cID, 3); err == nil && rawLogs != "" {
						containers[idx].ErrorSnippet = CleanLogSnippet(rawLogs, 3)
					}
				}
			}
			if containers[idx].Health == HealthStatusUnhealthy && containers[idx].ErrorSnippet == "" {
				if rawLogs, err := GetContainerLogs(cID, 3); err == nil && rawLogs != "" {
					containers[idx].ErrorSnippet = CleanLogSnippet(rawLogs, 3)
				}
			}
			rp, started := fetchContainerInspect(client, cID)
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

func fetchDockerSummaryViaSocket(sockPath string, summary *DockerSummary) bool {
	client := SocketClientFactory(sockPath)

	// 1. Query /info endpoint
	resp, err := client.Get("http://localhost/info")
	if err == nil && resp.StatusCode == http.StatusOK {
		func() {
			defer resp.Body.Close()
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				return
			}
			var infoData struct {
				Containers        int    `json:"Containers"`
				ContainersRunning int    `json:"ContainersRunning"`
				ContainersStopped int    `json:"ContainersStopped"`
				Images            int    `json:"Images"`
				ServerVersion     string `json:"ServerVersion"`
			}
			if json.Unmarshal(body, &infoData) == nil {
				summary.Status = DockerStatusRunning
				summary.StatusDisplay = DockerStatusDisplayRunning
				summary.TotalContainers = infoData.Containers
				summary.RunningContainers = infoData.ContainersRunning
				summary.StoppedContainers = infoData.ContainersStopped
				summary.ImagesCount = infoData.Images
				summary.Version = infoData.ServerVersion
			}
		}()
	}

	// 2. Query /containers/json?all=1 endpoint
	respC, errC := client.Get("http://localhost/containers/json?all=1")
	if errC != nil || respC.StatusCode != http.StatusOK {
		return false
	}
	defer respC.Body.Close()

	bodyC, readErr := io.ReadAll(respC.Body)
	if readErr != nil {
		return false
	}

	var rawContainers []rawSocketContainer
	if err := json.Unmarshal(bodyC, &rawContainers); err != nil {
		return false
	}

	summary.Status = DockerStatusRunning
	summary.StatusDisplay = DockerStatusDisplayRunning
	summary.Containers = parseRawSocketContainers(rawContainers)
	if summary.TotalContainers == 0 {
		summary.TotalContainers = len(summary.Containers)
	}
	enrichContainersWithSocketMetrics(client, summary.Containers)
	return true
}

// GetDockerSummary collects Docker daemon status and container list with live metrics.
func GetDockerSummary() DockerSummary {
	_, cliErr := ExecLookPath("docker")
	sockPath := DockerSocketFinder()

	if cliErr != nil && sockPath == "" {
		return DockerSummary{
			Status:        DockerStatusNotInstalled,
			StatusDisplay: DockerStatusDisplayNotInstalled,
		}
	}

	summary := DockerSummary{
		Status:        DockerStatusStopped,
		StatusDisplay: DockerStatusDisplayStopped,
	}

	if sockPath != "" {
		if fetchDockerSummaryViaSocket(sockPath, &summary) {
			return summary
		}
	}

	fetchDockerSummaryViaCLI(&summary)
	return summary
}
