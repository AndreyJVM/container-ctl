package swarm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AndreyJVM/dockpulse/internal/monitor/docker"
)

const swarmLogsTimeout = 10 * time.Second

// GetSwarmServiceLogs retrieves logs for a Swarm service by its ID or name.
func GetSwarmServiceLogs(serviceID string, tailLines int) (string, error) {
	cleanID := strings.TrimSpace(serviceID)
	if cleanID == "" {
		return "", fmt.Errorf("service ID cannot be empty")
	}
	if tailLines <= 0 {
		tailLines = defaultSwarmTail
	}

	sockPath := docker.DockerSocketFinder()
	if sockPath != "" {
		client := docker.SocketClientFactory(sockPath)
		urlPath := fmt.Sprintf("/services/%s/logs?stdout=1&stderr=1&tail=%d", url.PathEscape(cleanID), tailLines)
		ctx, cancel := context.WithTimeout(context.Background(), swarmLogsTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+urlPath, nil)
		if err == nil {
			resp, doErr := client.Do(req)
			if doErr == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return docker.DemuxDockerLogs(resp.Body), nil
				}
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), swarmLogsTimeout)
	defer cancel()
	out, err := docker.DockerCLIRunner(ctx, "service", "logs", "--tail", strconv.Itoa(tailLines), cleanID)
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
