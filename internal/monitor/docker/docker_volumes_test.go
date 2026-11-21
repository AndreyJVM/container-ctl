package docker

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestGetDockerVolumesViaSocket(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		t.Fatalf("CLI must not run after a successful socket request: %v", args)
		return nil, errors.New("unexpected CLI call")
	}
	SocketClientFactory = func(_ string) *http.Client {
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
	wantVolume := DockerVolume{Name: "shop_data", Driver: "local", Scope: "local", Mountpoint: "/var/lib/docker/volumes/shop_data/_data", Size: 4096, RefCount: 2, ComposeProject: "shop"}
	if !reflect.DeepEqual(volumes, []DockerVolume{wantVolume}) {
		t.Fatalf("unexpected volumes: %#v", volumes)
	}
}

func TestGetDockerVolumesCLIFallback(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
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
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
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
