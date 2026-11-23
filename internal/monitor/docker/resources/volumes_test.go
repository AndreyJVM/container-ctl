package resources

import (
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/engine"
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/types"
)

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestGetDockerVolumesViaSocket(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origFactory := engine.SocketClientFactory
	origRunner := engine.DockerCLIRunner
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.SocketClientFactory = origFactory
		engine.DockerCLIRunner = origRunner
	}()

	engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	engine.DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		t.Fatalf("CLI must not run after a successful socket request: %v", args)
		return nil, errors.New("unexpected CLI call")
	}
	engine.SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/volumes" {
				return mockStringResponse(http.StatusOK, `{"Volumes":[
					{"Name":"shop_data","Driver":"local","Scope":"local","Mountpoint":"/var/lib/docker/volumes/shop_data/_data","Labels":{"com.docker.compose.project":"shop"},"UsageData":{"Size":4096,"RefCount":2}}
				]}`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	volumes, err := GetDockerVolumes()
	if err != nil {
		t.Fatalf("GetDockerVolumes() returned an error: %v", err)
	}
	wantVolume := types.DockerVolume{Name: "shop_data", Driver: "local", Scope: "local", Mountpoint: "/var/lib/docker/volumes/shop_data/_data", Size: 4096, RefCount: 2, ComposeProject: "shop"}
	if !reflect.DeepEqual(volumes, []types.DockerVolume{wantVolume}) {
		t.Fatalf("unexpected volumes: %#v", volumes)
	}
}

func TestGetDockerVolumesCLIFallback(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origRunner := engine.DockerCLIRunner
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.DockerCLIRunner = origRunner
	}()

	engine.DockerSocketFinder = func() string { return "" }
	engine.DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "volume" {
			return []byte("shop_data|local|local|/data|com.docker.compose.project=shop\n"), nil
		}
		return nil, errors.New("unexpected command")
	}

	volumes, err := GetDockerVolumes()
	if err != nil {
		t.Fatalf("GetDockerVolumes() returned an error: %v", err)
	}
	if len(volumes) != 1 || volumes[0].ComposeProject != "shop" || volumes[0].Size != -1 {
		t.Fatalf("unexpected CLI volume data: %#v", volumes)
	}
}

func TestGetVolumeInspectJSON_Empty(t *testing.T) {
	if _, err := GetVolumeInspectJSON(""); err == nil {
		t.Errorf("expected error on empty volume name")
	}
}

func TestGetVolumeInspectJSON_Socket(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origFactory := engine.SocketClientFactory
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.SocketClientFactory = origFactory
	}()

	engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	engine.SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/volumes/my-vol") {
				return mockStringResponse(http.StatusOK, `{"Name":"my-vol","Driver":"local"}`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}
	out, err := GetVolumeInspectJSON("my-vol")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"Name": "my-vol"`) {
		t.Errorf("expected pretty JSON output, got: %s", out)
	}
}

func TestRemoveDockerVolume(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origFactory := engine.SocketClientFactory
	origRunner := engine.DockerCLIRunner
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.SocketClientFactory = origFactory
		engine.DockerCLIRunner = origRunner
	}()

	t.Run("empty volume name error", func(t *testing.T) {
		if err := RemoveDockerVolume("", false); err == nil {
			t.Errorf("expected error for empty volume name")
		}
	})

	t.Run("socket success force false", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodDelete && strings.Contains(req.URL.Path, "/volumes/vol-1") {
					if !strings.Contains(req.URL.RawQuery, "force=false") {
						t.Errorf("expected force=false in query, got: %s", req.URL.RawQuery)
					}
					return mockStringResponse(http.StatusNoContent, ""), nil
				}
				return mockStringResponse(http.StatusNotFound, "not found"), nil
			})
		}
		if err := RemoveDockerVolume("vol-1", false); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("socket success force true", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodDelete && strings.Contains(req.URL.Path, "/volumes/vol-1") {
					if !strings.Contains(req.URL.RawQuery, "force=true") {
						t.Errorf("expected force=true in query, got: %s", req.URL.RawQuery)
					}
					return mockStringResponse(http.StatusOK, ""), nil
				}
				return mockStringResponse(http.StatusNotFound, "not found"), nil
			})
		}
		if err := RemoveDockerVolume("vol-1", true); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("socket error 409 in use", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
				return mockStringResponse(http.StatusConflict, `{"message":"volume is in use"}`), nil
			})
		}
		err := RemoveDockerVolume("used-vol", false)
		if err == nil || !strings.Contains(err.Error(), "HTTP 409") {
			t.Errorf("expected HTTP 409 error, got: %v", err)
		}
	})

	t.Run("cli fallback success force true", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "" }
		engine.DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) == 4 && args[0] == "volume" && args[1] == "rm" && args[2] == "-f" && args[3] == "custom-vol" {
				return []byte("custom-vol\n"), nil
			}
			return nil, errors.New("unexpected command")
		}
		if err := RemoveDockerVolume("custom-vol", true); err != nil {
			t.Errorf("unexpected error on CLI fallback: %v", err)
		}
	})

	t.Run("cli fallback failure", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "" }
		engine.DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte("Error response from daemon: volume is in use"), errors.New("exit 1")
		}
		err := RemoveDockerVolume("locked-vol", false)
		if err == nil || !strings.Contains(err.Error(), "volume is in use") {
			t.Errorf("expected volume is in use error, got: %v", err)
		}
	})
}

func TestMatchContainersForVolume(t *testing.T) {
	if MatchContainersForVolume("", nil) != nil {
		t.Errorf("expected nil for empty volume name")
	}

	containers := []types.ContainerItem{
		{Name: "web", Mounts: []string{"app_data:/data", "logs:/var/log"}},
		{Name: "worker", Mounts: []string{"logs:/var/log"}},
		{Name: "db", Mounts: []string{"/host/path:/db"}},
	}

	matchedApp := MatchContainersForVolume("app_data", containers)
	if len(matchedApp) != 1 || matchedApp[0] != "web" {
		t.Errorf("expected ['web'] for app_data, got: %v", matchedApp)
	}

	matchedLogs := MatchContainersForVolume("logs", containers)
	if len(matchedLogs) != 2 || matchedLogs[0] != "web" || matchedLogs[1] != "worker" {
		t.Errorf("expected ['web', 'worker'] for logs, got: %v", matchedLogs)
	}

	matchedMissing := MatchContainersForVolume("unrelated", containers)
	if len(matchedMissing) != 0 {
		t.Errorf("expected empty for unrelated volume, got: %v", matchedMissing)
	}
}

func TestEnrichVolumesWithContainers(t *testing.T) {
	if EnrichVolumesWithContainers(nil, nil) != nil {
		t.Errorf("expected nil for nil volumes")
	}

	volumes := []types.DockerVolume{
		{Name: "app_data"},
		{Name: "logs", Containers: []string{"pre-existing"}},
		{Name: "unused"},
	}

	containers := []types.ContainerItem{
		{Name: "web", Mounts: []string{"app_data:/data"}},
	}

	enriched := EnrichVolumesWithContainers(volumes, containers)
	if len(enriched) != 3 {
		t.Fatalf("expected 3 volumes, got %d", len(enriched))
	}
	if len(enriched[0].Containers) != 1 || enriched[0].Containers[0] != "web" {
		t.Errorf("expected app_data to be enriched with web, got: %v", enriched[0].Containers)
	}
	// logs had pre-existing, should not be overwritten
	if len(enriched[1].Containers) != 1 || enriched[1].Containers[0] != "pre-existing" {
		t.Errorf("expected pre-existing containers preserved, got: %v", enriched[1].Containers)
	}
	if len(enriched[2].Containers) != 0 {
		t.Errorf("expected empty containers for unused, got: %v", enriched[2].Containers)
	}
}
