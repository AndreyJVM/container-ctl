package docker

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetDockerSummary(t *testing.T) {
	// Call should safely complete whether Docker is installed/running or not
	summary := GetDockerSummary()
	if summary.Status != DockerStatusRunning && summary.Status != DockerStatusStopped && summary.Status != DockerStatusNotInstalled {
		t.Errorf("unexpected docker status: %s", summary.Status)
	}
}

func TestGetDockerSummary_NotInstalled(t *testing.T) {
	origLookPath := ExecLookPath
	origFinder := DockerSocketFinder
	defer func() {
		ExecLookPath = origLookPath
		DockerSocketFinder = origFinder
	}()

	ExecLookPath = func(_ string) (string, error) {
		return "", errors.New("command not found")
	}
	DockerSocketFinder = func() string {
		return ""
	}

	summary := GetDockerSummary()
	if summary.Status != DockerStatusNotInstalled {
		t.Errorf("expected status %q, got %q", DockerStatusNotInstalled, summary.Status)
	}
	if summary.StatusDisplay != DockerStatusDisplayNotInstalled {
		t.Errorf("expected display %q, got %q", DockerStatusDisplayNotInstalled, summary.StatusDisplay)
	}
}

func TestGetDockerSummary_SocketSuccess(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			switch {
			case path == "/info":
				return mockStringResponse(http.StatusOK, `{
					"Containers": 2,
					"ContainersRunning": 1,
					"ContainersStopped": 1,
					"Images": 5,
					"ServerVersion": "24.0.5"
				}`), nil
			case path == "/containers/json":
				return mockStringResponse(http.StatusOK, `[
					{
						"Id": "abc123456789012345",
						"Names": ["/my-web"],
						"Image": "nginx:latest",
						"State": "running",
						"Status": "Up 2 hours",
						"Created": 1672531199,
						"Ports": [{"PublicPort": 8080, "PrivatePort": 80, "Type": "tcp"}],
						"Labels": {
							"com.docker.compose.project": "webproj",
							"com.docker.compose.service": "frontend"
						},
						"NetworkSettings": {
							"Networks": {
								"webnet": {"IPAddress": "172.18.0.2"}
							}
						},
						"Mounts": [
							{"Source": "/var/www", "Destination": "/usr/share/nginx/html", "Mode": "ro"}
						]
					},
					{
						"Id": "def456789012345678",
						"Names": ["/my-db"],
						"Image": "postgres:15",
						"State": "exited",
						"Status": "Exited (0) 10 minutes ago",
						"Created": 0
					}
				]`), nil
			case strings.Contains(path, "/stats"):
				return mockStringResponse(http.StatusOK, `{}`), nil
			case strings.Contains(path, "/json"):
				return mockStringResponse(http.StatusOK, `{
					"HostConfig": {"RestartPolicy": {"Name": "always"}},
					"State": {"StartedAt": "2026-01-01T12:00:00Z"}
				}`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	summary := GetDockerSummary()
	if summary.Status != DockerStatusRunning {
		t.Fatalf("expected status %q, got %q", DockerStatusRunning, summary.Status)
	}
	if summary.Version != "24.0.5" {
		t.Errorf("expected version 24.0.5, got %q", summary.Version)
	}
	if summary.RunningContainers != 1 || summary.StoppedContainers != 1 || summary.TotalContainers != 2 {
		t.Errorf("unexpected counts: %+v", summary)
	}
	if len(summary.Containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(summary.Containers))
	}
	c0 := summary.Containers[0]
	if c0.ID != "abc123456789" {
		t.Errorf("expected truncated ID abc123456789, got %q", c0.ID)
	}
	if c0.Name != "my-web" || c0.ComposeProject != "webproj" || c0.ComposeService != "frontend" {
		t.Errorf("unexpected container metadata: %+v", c0)
	}
	if c0.IPAddress != "172.18.0.2" || c0.NetworkName != "webnet" {
		t.Errorf("unexpected network: IP=%s Net=%s", c0.IPAddress, c0.NetworkName)
	}
	if len(c0.Mounts) != 1 || c0.Mounts[0] != "/var/www -> /usr/share/nginx/html (ro)" {
		t.Errorf("unexpected mounts: %v", c0.Mounts)
	}
}

func TestGetDockerSummary_CLIFallback(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	origLookPath := ExecLookPath
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
		ExecLookPath = origLookPath
	}()

	ExecLookPath = func(_ string) (string, error) {
		return "/mock/docker", nil
	}
	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) == 0 {
			return nil, errors.New("no args")
		}
		switch args[0] {
		case "info":
			return []byte("1 1 0 3 24.0.2"), nil
		case "ps":
			return []byte("c123\tweb-app\tnginx\trunning\tUp 5 minutes (healthy)\t80/tcp\t/src,/dst\tcom.docker.compose.project=app,com.docker.compose.service=web\tbridge\t2026-01-01\t5m"), nil
		case "inspect":
			return []byte("c123|always|172.17.0.5"), nil
		}
		return nil, errors.New("unknown cmd")
	}

	summary := GetDockerSummary()
	if summary.Status != DockerStatusRunning {
		t.Fatalf("expected status %q, got %q", DockerStatusRunning, summary.Status)
	}
	if summary.Version != "24.0.2" {
		t.Errorf("expected version 24.0.2, got %q", summary.Version)
	}
	if summary.TotalContainers != 1 || summary.RunningContainers != 1 {
		t.Errorf("unexpected counts: Total=%d Running=%d", summary.TotalContainers, summary.RunningContainers)
	}
	if len(summary.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(summary.Containers))
	}
	c := summary.Containers[0]
	if c.Name != "web-app" || c.ComposeProject != "app" || c.ComposeService != "web" {
		t.Errorf("unexpected container info: %+v", c)
	}
	if c.RestartPolicy != "always" || c.IPAddress != "172.17.0.5" {
		t.Errorf("unexpected inspect enrichment: restart=%s ip=%s", c.RestartPolicy, c.IPAddress)
	}
}

func TestGetDockerSummary_SocketInfoFails_ContainersSucceed(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			switch {
			case path == "/info":
				// Simulating /info endpoint failure
				return mockStringResponse(http.StatusInternalServerError, "internal error"), nil
			case path == "/containers/json":
				return mockStringResponse(http.StatusOK, `[
					{
						"Id": "111111111111111111",
						"Names": ["/c1"],
						"State": "running",
						"Status": "Up 5 minutes"
					},
					{
						"Id": "222222222222222222",
						"Names": ["/c2"],
						"State": "exited",
						"Status": "Exited (0) 10 minutes ago"
					}
				]`), nil
			case strings.Contains(path, "/stats"):
				return mockStringResponse(http.StatusOK, `{}`), nil
			case strings.Contains(path, "/json"):
				return mockStringResponse(http.StatusOK, `{}`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	summary := GetDockerSummary()
	if summary.Status != DockerStatusRunning {
		t.Fatalf("expected status %q, got %q", DockerStatusRunning, summary.Status)
	}
	if summary.TotalContainers != 2 {
		t.Errorf("expected TotalContainers 2, got %d", summary.TotalContainers)
	}
	if summary.RunningContainers != 1 {
		t.Errorf("expected RunningContainers 1, got %d", summary.RunningContainers)
	}
	if summary.StoppedContainers != 1 {
		t.Errorf("expected StoppedContainers 1, got %d", summary.StoppedContainers)
	}
}

func TestGetDockerSummary_NoCLI_SocketFails(t *testing.T) {
	origFinder := DockerSocketFinder
	origLookPath := ExecLookPath
	origRunner := DockerCLIRunner
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		ExecLookPath = origLookPath
		DockerCLIRunner = origRunner
		SocketClientFactory = origFactory
	}()

	ExecLookPath = func(_ string) (string, error) {
		return "", errors.New("docker not found")
	}
	DockerSocketFinder = func() string { return "/tmp/mock.sock" }

	cliInvoked := false
	DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		cliInvoked = true
		return nil, errors.New("cli should not be invoked")
	}

	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		})
	}

	summary := GetDockerSummary()
	if summary.Status != DockerStatusStopped {
		t.Errorf("expected status %q, got %q", DockerStatusStopped, summary.Status)
	}
	if cliInvoked {
		t.Error("expected DockerCLIRunner NOT to be invoked when docker is not installed")
	}
}

func TestParseRawSocketContainers_DeterministicNetworks(t *testing.T) {
	raw := []rawSocketContainer{
		{
			ID:    "abc123456789012345",
			Names: []string{"/test-container"},
			NetworkSettings: struct {
				Networks map[string]struct {
					IPAddress string `json:"IPAddress"`
					Gateway   string `json:"Gateway"`
				} `json:"Networks"`
			}{
				Networks: map[string]struct {
					IPAddress string `json:"IPAddress"`
					Gateway   string `json:"Gateway"`
				}{
					"zebra-net": {IPAddress: "10.0.0.2"},
					"alpha-net": {IPAddress: "172.18.0.2"},
					"beta-net":  {IPAddress: "192.168.1.2"},
				},
			},
		},
	}

	// Repeated parses must yield identical, sorted output
	for i := 0; i < 10; i++ {
		items := parseRawSocketContainers(raw)
		if len(items) != 1 {
			t.Fatalf("expected 1 item, got %d", len(items))
		}
		expectedNets := "alpha-net, beta-net, zebra-net"
		expectedIPs := "172.18.0.2, 192.168.1.2, 10.0.0.2"
		if items[0].NetworkName != expectedNets {
			t.Errorf("iteration %d: expected networks %q, got %q", i, expectedNets, items[0].NetworkName)
		}
		if items[0].IPAddress != expectedIPs {
			t.Errorf("iteration %d: expected IPs %q, got %q", i, expectedIPs, items[0].IPAddress)
		}
	}
}
