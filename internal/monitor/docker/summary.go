package docker

import (
	"context"
	"encoding/json"
	"net/http"
)

func fetchDockerSummaryViaSocket(sockPath string, summary *DockerSummary) bool {
	client := SocketClientFactory(sockPath)

	ctx, cancel := context.WithTimeout(context.Background(), defaultDockerTimeout)
	defer cancel()

	// 1. Query /info endpoint
	var infoData struct {
		Containers        int    `json:"Containers"`
		ContainersRunning int    `json:"ContainersRunning"`
		ContainersStopped int    `json:"ContainersStopped"`
		Images            int    `json:"Images"`
		ServerVersion     string `json:"ServerVersion"`
	}
	infoSuccess := false

	reqInfo, errInfo := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/info", nil)
	if errInfo == nil {
		resp, err := client.Do(reqInfo)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				if err := json.NewDecoder(resp.Body).Decode(&infoData); err == nil {
					infoSuccess = true
				}
			}
			_ = resp.Body.Close()
		} else {
			// If dialing the socket failed (connection refused, permission denied, etc.),
			// there is no point in trying /containers/json.
			return false
		}
	}

	// 2. Query /containers/json?all=1 endpoint
	reqC, errCReq := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/containers/json?all=1", nil)
	if errCReq != nil {
		return false
	}
	respC, errC := client.Do(reqC)
	if errC != nil {
		return false
	}
	defer respC.Body.Close()
	if respC.StatusCode != http.StatusOK {
		return false
	}

	var rawContainers []rawSocketContainer
	if err := json.NewDecoder(respC.Body).Decode(&rawContainers); err != nil {
		return false
	}

	// Update summary atomically
	summary.Status = DockerStatusRunning
	summary.StatusDisplay = DockerStatusDisplayRunning
	if infoSuccess {
		summary.TotalContainers = infoData.Containers
		summary.RunningContainers = infoData.ContainersRunning
		summary.StoppedContainers = infoData.ContainersStopped
		summary.ImagesCount = infoData.Images
		summary.Version = infoData.ServerVersion
	}

	summary.Containers = parseRawSocketContainers(rawContainers)

	// Fallback count calculation if /info was unavailable or returned zeroes
	if summary.TotalContainers == 0 {
		summary.TotalContainers = len(summary.Containers)
	}
	if !infoSuccess {
		var running, stopped int
		for _, c := range summary.Containers {
			if c.State == "running" {
				running++
			} else {
				stopped++
			}
		}
		summary.RunningContainers = running
		summary.StoppedContainers = stopped
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

	if cliErr == nil {
		fetchDockerSummaryViaCLI(&summary)
	}
	return summary
}
