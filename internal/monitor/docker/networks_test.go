package docker

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestGetDockerNetworksViaSocket(t *testing.T) {
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
			if req.URL.Path == "/networks" {
				return mockStringResponse(http.StatusOK, `[
					{"Id":"network1234567890","Name":"frontend","Driver":"bridge","Scope":"local","Internal":false,"Attachable":true,"Containers":{"c1":{}},"Labels":{"com.docker.compose.project":"shop"},"IPAM":{"Config":[{"Subnet":"172.20.0.0/16","Gateway":"172.20.0.1"}]}}
				]`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	networks, err := GetDockerNetworks()
	if err != nil {
		t.Fatalf("GetDockerNetworks() returned an error: %v", err)
	}
	wantNetwork := DockerNetwork{ID: "network12345", Name: "frontend", Driver: "bridge", Scope: "local", Attachable: true, Containers: 1, Subnet: "172.20.0.0/16", Gateway: "172.20.0.1", ComposeProject: "shop"}
	if !reflect.DeepEqual(networks, []DockerNetwork{wantNetwork}) {
		t.Fatalf("unexpected networks: %#v", networks)
	}
}

func TestGetDockerNetworksCLIFallback(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "network" {
			return []byte("network1234567890|frontend|bridge|local|false|com.docker.compose.project=shop\n"), nil
		}
		return nil, errors.New("unexpected command")
	}

	networks, err := GetDockerNetworks()
	if err != nil {
		t.Fatalf("GetDockerNetworks() returned an error: %v", err)
	}
	if len(networks) != 1 || networks[0].ComposeProject != "shop" {
		t.Fatalf("unexpected CLI network data: %#v", networks)
	}
}

func TestGetNetworkInspectJSON_Empty(t *testing.T) {
	if _, err := GetNetworkInspectJSON(""); err == nil {
		t.Errorf("expected error for empty network identifier")
	}
}

func TestGetNetworkInspectJSON_Socket(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/networks/net-1") {
				return mockStringResponse(http.StatusOK, `{"Name":"net-1","Driver":"bridge"}`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	out, err := GetNetworkInspectJSON("net-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"Name": "net-1"`) {
		t.Errorf("expected pretty JSON output, got: %s", out)
	}
}
