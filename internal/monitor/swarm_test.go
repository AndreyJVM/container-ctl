package monitor

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetSwarmSummary(t *testing.T) {
	summary := GetSwarmSummary()
	// Should safely return summary without panic
	if summary.Text == "" {
		t.Errorf("expected non-empty Swarm summary text")
	}
}

func TestFetchSwarmTasks(t *testing.T) {
	// Success case
	clientOK := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/tasks") {
			return mockStringResponse(http.StatusOK, `[
				{"ID":"t1","ServiceID":"s1","NodeID":"n1","Status":{"State":"running"}},
				{"ID":"t2","ServiceID":"s1","NodeID":"n2","Status":{"State":"running"}}
			]`), nil
		}
		return mockStringResponse(http.StatusNotFound, "not found"), nil
	})

	svc, node := fetchSwarmTasks(clientOK)
	if svc["s1"] != 2 || node["n1"] != 1 || node["n2"] != 1 {
		t.Errorf("unexpected fetchSwarmTasks result: svc=%+v node=%+v", svc, node)
	}

	// Failure case
	clientErr := newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return mockStringResponse(http.StatusInternalServerError, "error"), nil
	})
	svcErr, nodeErr := fetchSwarmTasks(clientErr)
	if len(svcErr) != 0 || len(nodeErr) != 0 {
		t.Errorf("expected empty maps on HTTP error")
	}
}

func TestFetchSwarmNodes(t *testing.T) {
	// Success case
	clientOK := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/nodes") {
			return mockStringResponse(http.StatusOK, `[
				{"ID":"n1","Spec":{"Role":"manager"},"Description":{"Hostname":"node-1"},"Status":{"State":"ready"}}
			]`), nil
		}
		return mockStringResponse(http.StatusNotFound, "not found"), nil
	})

	nodes := fetchSwarmNodes(clientOK, map[string]int{"n1": 3})
	if len(nodes) != 1 || nodes[0].Hostname != "node-1" || nodes[0].ActiveTasks != 3 {
		t.Errorf("unexpected fetchSwarmNodes result: %+v", nodes)
	}

	// Failure case
	clientErr := newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("network failure")
	})
	nodesErr := fetchSwarmNodes(clientErr, nil)
	if nodesErr != nil {
		t.Errorf("expected nil on error")
	}
}

func TestFetchSwarmServices(t *testing.T) {
	// Success case
	clientOK := newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/services") {
			return mockStringResponse(http.StatusOK, `[
				{"ID":"s1","Spec":{"Name":"api-service","TaskTemplate":{"ContainerSpec":{"Image":"api:latest"}},"Mode":{"Replicated":{"Replicas":2}}}}
			]`), nil
		}
		return mockStringResponse(http.StatusNotFound, "not found"), nil
	})

	svcs := fetchSwarmServices(clientOK, map[string]int{"s1": 2})
	if len(svcs) != 1 || svcs[0].Name != "api-service" || svcs[0].Replicas != "2/2" {
		t.Errorf("unexpected fetchSwarmServices result: %+v", svcs)
	}

	// Failure case
	clientErr := newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
		return mockStringResponse(http.StatusInternalServerError, "server error"), nil
	})
	svcsErr := fetchSwarmServices(clientErr, nil)
	if svcsErr != nil {
		t.Errorf("expected nil on error")
	}
}

func TestGetSwarmSummary_SocketActive(t *testing.T) {
	origFinder := dockerSocketFinder
	origFactory := socketClientFactory
	defer func() {
		dockerSocketFinder = origFinder
		socketClientFactory = origFactory
	}()

	dockerSocketFinder = func() string { return "/tmp/mock-docker.sock" }
	socketClientFactory = func(_ string) *http.Client {
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
	origFinder := dockerSocketFinder
	origRunner := dockerCLICommandRunner
	defer func() {
		dockerSocketFinder = origFinder
		dockerCLICommandRunner = origRunner
	}()

	dockerSocketFinder = func() string { return "" }
	dockerCLICommandRunner = func(_ context.Context, _ ...string) ([]byte, error) {
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
	origFinder := dockerSocketFinder
	origRunner := dockerCLICommandRunner
	defer func() {
		dockerSocketFinder = origFinder
		dockerCLICommandRunner = origRunner
	}()

	dockerSocketFinder = func() string { return "" }
	dockerCLICommandRunner = func(_ context.Context, _ ...string) ([]byte, error) {
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
