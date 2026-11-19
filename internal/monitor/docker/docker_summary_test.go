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
