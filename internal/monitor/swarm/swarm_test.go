package swarm

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/monitor/docker"
)

func TestGetSwarmSummary(t *testing.T) {
	summary := GetSwarmSummary()
	// Should safely return summary without panic
	if summary.Text == "" {
		t.Errorf("expected non-empty Swarm summary text")
	}
}

func TestGetSwarmSummary_SocketActive(t *testing.T) {
	origFinder := docker.DockerSocketFinder
	origFactory := docker.SocketClientFactory
	defer func() {
		docker.DockerSocketFinder = origFinder
		docker.SocketClientFactory = origFactory
	}()

	docker.DockerSocketFinder = func() string { return "/tmp/mock-docker.sock" }
	docker.SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			switch {
			case strings.Contains(path, "/info"):
				return mockStringResponse(http.StatusOK, `{
					"Swarm": {
						"LocalNodeState": "active",
						"ControlAvailable": true,
						"NodeID": "mgrnode12345678",
						"Nodes": 3,
						"Managers": 1
					}
				}`), nil
			case strings.Contains(path, "/tasks"):
				return mockStringResponse(http.StatusOK, `[
					{"ID":"t1","ServiceID":"s1","NodeID":"mgrnode12345678","Status":{"State":"running"}}
				]`), nil
			case strings.Contains(path, "/nodes"):
				return mockStringResponse(http.StatusOK, `[
					{"ID":"mgrnode12345678","Spec":{"Role":"manager"},"Description":{"Hostname":"master-1"},"Status":{"State":"ready"},"ManagerStatus":{"Leader":true}}
				]`), nil
			case strings.Contains(path, "/services"):
				return mockStringResponse(http.StatusOK, `[
					{"ID":"s1","Spec":{"Name":"web","TaskTemplate":{"ContainerSpec":{"Image":"nginx"}},"Mode":{"Replicated":{"Replicas":1}}}}
				]`), nil
			default:
				return mockStringResponse(http.StatusNotFound, "not found"), nil
			}
		})
	}

	summary := GetSwarmSummary()
	if !summary.IsActive || !summary.IsManager {
		t.Errorf("expected active manager Swarm summary, got %+v", summary)
	}
	if summary.NodesCount != 3 || len(summary.Nodes) != 1 || len(summary.Services) != 1 {
		t.Errorf("unexpected counts: nodes=%d, services=%d", len(summary.Nodes), len(summary.Services))
	}
	if !strings.Contains(summary.Text, "3 nodes") {
		t.Errorf("unexpected summary text: %s", summary.Text)
	}
}

func TestGetSwarmSummary_CLIFallback(t *testing.T) {
	origFinder := docker.DockerSocketFinder
	origRunner := docker.DockerCLIRunner
	defer func() {
		docker.DockerSocketFinder = origFinder
		docker.DockerCLIRunner = origRunner
	}()

	docker.DockerSocketFinder = func() string { return "" }
	docker.DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("active true 4 2"), nil
	}

	summary := GetSwarmSummary()
	if !summary.IsActive || !summary.IsManager {
		t.Errorf("expected active manager Swarm summary from CLI: %+v", summary)
	}
	if summary.NodesCount != 4 || summary.ManagersCount != 2 {
		t.Errorf("expected 4 nodes, 2 managers, got %d/%d", summary.NodesCount, summary.ManagersCount)
	}
}

func TestGetSwarmSummary_CLIFallback_Inactive(t *testing.T) {
	origFinder := docker.DockerSocketFinder
	origRunner := docker.DockerCLIRunner
	defer func() {
		docker.DockerSocketFinder = origFinder
		docker.DockerCLIRunner = origRunner
	}()

	docker.DockerSocketFinder = func() string { return "" }
	docker.DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("inactive false 0 0"), nil
	}

	summary := GetSwarmSummary()
	if summary.IsActive {
		t.Errorf("expected inactive Swarm summary from CLI")
	}
	if !strings.Contains(summary.Text, "not active") {
		t.Errorf("expected inactive text, got %q", summary.Text)
	}
}

func TestFetchSocketAPI_Errors(t *testing.T) {
	// Nil client
	if _, err := fetchSocketAPI(nil, "/test", swarmTimeout); err == nil {
		t.Errorf("expected error for nil client")
	}

	// Non-200 status code
	client404 := newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return mockStringResponse(http.StatusNotFound, "not found"), nil
	})
	if _, err := fetchSocketAPI(client404, "/test", swarmTimeout); err == nil {
		t.Errorf("expected error for 404 response")
	}
}
