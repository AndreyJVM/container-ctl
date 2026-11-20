package swarm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/AndreyJVM/dockpulse/internal/monitor/docker"
)

const (
	swarmTimeout        = 5 * time.Second
	defaultSwarmTail    = 100
	swarmIDTruncateLen  = 12
	swarmStatusActive   = "Active"
	swarmStatusInactive = "Inactive"
	swarmTextInactive   = "Docker Swarm is not active"
)

// SwarmNode represents a single cluster node in Docker Swarm.
type SwarmNode struct {
	ID           string
	Hostname     string
	Role         string
	State        string
	Availability string
	IsLeader     bool
	ActiveTasks  int
}

// SwarmService represents a deployed cluster service in Docker Swarm.
type SwarmService struct {
	ID              string
	Name            string
	Image           string
	Mode            string
	Replicas        string
	RunningReplicas int
	TargetReplicas  int
	Ports           string
}

// SwarmSummary represents high-level Swarm cluster status and topology.
type SwarmSummary struct {
	IsActive      bool
	StatusDisplay string
	NodeID        string
	IsManager     bool
	NodesCount    int
	ManagersCount int
	Nodes         []SwarmNode
	Services      []SwarmService
	Text          string
}

func fetchSocketAPI(client *http.Client, path string, timeout time.Duration) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("http client is nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+path, nil)
	if err != nil {
		return nil, err
	}

	resp, doErr := client.Do(req)
	if doErr != nil {
		return nil, doErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, path)
	}

	return io.ReadAll(resp.Body)
}

func fetchSwarmSummaryViaSocket(sockPath string, summary *SwarmSummary) bool {
	client := docker.SocketClientFactory(sockPath)
	body, err := fetchSocketAPI(client, "/info", swarmTimeout)
	if err != nil {
		return false
	}

	parsed, ok := parseSwarmInfo(body)
	if !ok {
		return false
	}
	*summary = parsed

	// Concurrent querying of /tasks, /nodes, and /services to minimize latency
	var (
		tasksBody    []byte
		nodesBody    []byte
		servicesBody []byte
		wg           sync.WaitGroup
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		tasksBody, _ = fetchSocketAPI(client, "/tasks", swarmTimeout)
	}()
	go func() {
		defer wg.Done()
		nodesBody, _ = fetchSocketAPI(client, "/nodes", swarmTimeout)
	}()
	go func() {
		defer wg.Done()
		servicesBody, _ = fetchSocketAPI(client, "/services", swarmTimeout)
	}()
	wg.Wait()

	svcTasks, nodeTasks := parseSwarmTasks(tasksBody)
	summary.Nodes = parseSwarmNodes(nodesBody, nodeTasks)
	summary.Services = parseSwarmServices(servicesBody, svcTasks)
	summary.Text = fmt.Sprintf("Swarm cluster: %d nodes (%d managers), %d services", summary.NodesCount, summary.ManagersCount, len(summary.Services))
	return true
}

func fetchSwarmSummaryViaCLI(summary *SwarmSummary) {
	ctx, cancel := context.WithTimeout(context.Background(), swarmTimeout)
	defer cancel()
	out, err := docker.DockerCLIRunner(ctx, "info", "--format", "{{.Swarm.LocalNodeState}} {{.Swarm.ControlAvailable}} {{.Swarm.Nodes}} {{.Swarm.Managers}}")
	if err != nil {
		return
	}

	active, isManager, nodes, managers := parseSwarmCLIInfo(string(out))
	if !active {
		return
	}

	summary.IsActive = true
	summary.StatusDisplay = swarmStatusActive
	summary.IsManager = isManager
	summary.NodesCount = nodes
	summary.ManagersCount = managers

	if isManager {
		if nodeOut, nErr := docker.DockerCLIRunner(ctx, "node", "ls", "--format", "{{.ID}}\t{{.Hostname}}\t{{.Role}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}"); nErr == nil {
			summary.Nodes = parseSwarmCLINodes(string(nodeOut))
		}
		if svcOut, sErr := docker.DockerCLIRunner(ctx, "service", "ls", "--format", "{{.ID}}\t{{.Name}}\t{{.Mode}}\t{{.Replicas}}\t{{.Image}}\t{{.Ports}}"); sErr == nil {
			summary.Services = parseSwarmCLIServices(string(svcOut))
		}
	}

	if len(summary.Services) > 0 {
		summary.Text = fmt.Sprintf("Swarm cluster: %d nodes (%d managers), %d services", summary.NodesCount, summary.ManagersCount, len(summary.Services))
	} else {
		summary.Text = fmt.Sprintf("Swarm cluster: %d nodes (%d managers)", nodes, managers)
	}
}

// GetSwarmSummary collects information about Docker Swarm cluster state and topology.
func GetSwarmSummary() SwarmSummary {
	summary := SwarmSummary{
		IsActive:      false,
		StatusDisplay: swarmStatusInactive,
		Text:          swarmTextInactive,
	}

	sockPath := docker.DockerSocketFinder()
	if sockPath != "" {
		if fetchSwarmSummaryViaSocket(sockPath, &summary) {
			return summary
		}
	}

	fetchSwarmSummaryViaCLI(&summary)
	return summary
}

func fetchSwarmTasks(client *http.Client) (map[string]int, map[string]int) {
	body, err := fetchSocketAPI(client, "/tasks", swarmTimeout)
	if err != nil {
		return make(map[string]int), make(map[string]int)
	}
	return parseSwarmTasks(body)
}

func fetchSwarmNodes(client *http.Client, nodeTasks map[string]int) []SwarmNode {
	body, err := fetchSocketAPI(client, "/nodes", swarmTimeout)
	if err != nil {
		return nil
	}
	return parseSwarmNodes(body, nodeTasks)
}

func fetchSwarmServices(client *http.Client, runningTasks map[string]int) []SwarmService {
	body, err := fetchSocketAPI(client, "/services", swarmTimeout)
	if err != nil {
		return nil
	}
	return parseSwarmServices(body, runningTasks)
}
