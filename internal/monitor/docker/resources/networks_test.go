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

func TestGetDockerNetworksViaSocket(t *testing.T) {
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
	wantNetwork := types.DockerNetwork{
		ID:             "network12345",
		Name:           "frontend",
		Driver:         "bridge",
		Scope:          "local",
		Attachable:     true,
		Containers:     1,
		Subnet:         "172.20.0.0/16",
		Gateway:        "172.20.0.1",
		ComposeProject: "shop",
		Labels:         map[string]string{"com.docker.compose.project": "shop"},
	}
	if !reflect.DeepEqual(networks, []types.DockerNetwork{wantNetwork}) {
		t.Fatalf("unexpected networks: %#v", networks)
	}
}

func TestGetDockerNetworksCLIFallback(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origRunner := engine.DockerCLIRunner
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.DockerCLIRunner = origRunner
	}()

	engine.DockerSocketFinder = func() string { return "" }
	engine.DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
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
	origFinder := engine.DockerSocketFinder
	origFactory := engine.SocketClientFactory
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.SocketClientFactory = origFactory
	}()

	engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	engine.SocketClientFactory = func(_ string) *http.Client {
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

func TestGetDockerNetworksConnectedContainers(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origFactory := engine.SocketClientFactory
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.SocketClientFactory = origFactory
	}()

	engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	engine.SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/networks" {
				return mockStringResponse(http.StatusOK, `[
					{
						"Id": "net-with-containers",
						"Name": "app-net",
						"Driver": "bridge",
						"Containers": {
							"c1": {"Name": "/web", "IPv4Address": "172.20.0.2/16"},
							"c2": {"Name": "db", "IPv4Address": ""}
						}
					}
				]`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	networks, err := GetDockerNetworks()
	if err != nil {
		t.Fatalf("GetDockerNetworks() error: %v", err)
	}
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	if networks[0].Containers != 2 {
		t.Errorf("expected 2 containers count, got %d", networks[0].Containers)
	}
	if len(networks[0].ConnectedContainers) != 2 {
		t.Fatalf("expected 2 connected containers, got: %v", networks[0].ConnectedContainers)
	}
	// "db" and "web (172.20.0.2)"
	containsWeb := false
	containsDB := false
	for _, c := range networks[0].ConnectedContainers {
		if strings.Contains(c, "web (172.20.0.2)") {
			containsWeb = true
		}
		if strings.Contains(c, "db") {
			containsDB = true
		}
	}
	if !containsWeb || !containsDB {
		t.Errorf("unexpected connected containers: %v", networks[0].ConnectedContainers)
	}
}

func TestRemoveDockerNetwork(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origFactory := engine.SocketClientFactory
	origRunner := engine.DockerCLIRunner
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.SocketClientFactory = origFactory
		engine.DockerCLIRunner = origRunner
	}()

	t.Run("empty identifier error", func(t *testing.T) {
		if err := RemoveDockerNetwork(""); err == nil {
			t.Errorf("expected error for empty network identifier")
		}
	})

	t.Run("socket success 204", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodDelete && strings.Contains(req.URL.Path, "/networks/net-1") {
					return mockStringResponse(http.StatusNoContent, ""), nil
				}
				return mockStringResponse(http.StatusNotFound, "not found"), nil
			})
		}
		if err := RemoveDockerNetwork("net-1"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("socket error 404 with message", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
		engine.SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
				return mockStringResponse(http.StatusNotFound, `{"message":"network net-not-found not found"}`), nil
			})
		}
		err := RemoveDockerNetwork("net-not-found")
		if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
			t.Errorf("expected HTTP 404 error, got: %v", err)
		}
	})

	t.Run("cli fallback success", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "" }
		engine.DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
			if len(args) == 3 && args[0] == "network" && args[1] == "rm" && args[2] == "custom-net" {
				return []byte("custom-net\n"), nil
			}
			return nil, errors.New("unexpected command")
		}
		if err := RemoveDockerNetwork("custom-net"); err != nil {
			t.Errorf("unexpected error on CLI fallback: %v", err)
		}
	})

	t.Run("cli fallback failure", func(t *testing.T) {
		engine.DockerSocketFinder = func() string { return "" }
		engine.DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte("Error response from daemon: network has active endpoints"), errors.New("exit 1")
		}
		err := RemoveDockerNetwork("active-net")
		if err == nil || !strings.Contains(err.Error(), "has active endpoints") {
			t.Errorf("expected active endpoints error, got: %v", err)
		}
	})
}

func TestGetDockerNetworks_DetailedIPAMAndContainers(t *testing.T) {
	origFinder := engine.DockerSocketFinder
	origFactory := engine.SocketClientFactory
	defer func() {
		engine.DockerSocketFinder = origFinder
		engine.SocketClientFactory = origFactory
	}()

	engine.DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	engine.SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/networks" {
				return mockStringResponse(http.StatusOK, `[
					{
						"Id": "abc1234567890def",
						"Name": "custom-bridge",
						"Created": "2026-11-23T10:00:00Z",
						"Driver": "bridge",
						"Scope": "local",
						"Internal": true,
						"Attachable": true,
						"Ingress": false,
						"EnableIPv6": true,
						"IPAM": {
							"Driver": "default",
							"Config": [
								{
									"Subnet": "172.28.0.0/16",
									"Gateway": "172.28.0.1",
									"IPRange": "172.28.5.0/24"
								}
							]
						},
						"Options": {
							"com.docker.network.bridge.name": "br-custom",
							"com.docker.network.driver.mtu": "1450"
						},
						"Labels": {
							"environment": "production"
						},
						"Containers": {
							"cont1234567890": {
								"Name": "api-server",
								"EndpointID": "ep1234567890",
								"MacAddress": "02:42:ac:1c:05:02",
								"IPv4Address": "172.28.5.2/16",
								"IPv6Address": "fd00::2"
							}
						}
					}
				]`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	networks, err := GetDockerNetworks()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	net := networks[0]
	if net.ID != "abc123456789" {
		t.Errorf("expected truncated ID abc123456789, got %s", net.ID)
	}
	if !net.Internal || !net.Attachable || !net.EnableIPv6 {
		t.Errorf("expected Internal, Attachable, EnableIPv6 true")
	}
	if net.Subnet != "172.28.0.0/16" || net.Gateway != "172.28.0.1" || net.IPRange != "172.28.5.0/24" {
		t.Errorf("unexpected IPAM config: subnet=%s, gw=%s, range=%s", net.Subnet, net.Gateway, net.IPRange)
	}
	if net.IPAMDriver != "default" {
		t.Errorf("expected IPAMDriver default, got %s", net.IPAMDriver)
	}
	if net.Options["com.docker.network.bridge.name"] != "br-custom" {
		t.Errorf("expected custom bridge name in options")
	}
	if len(net.ContainerDetails) != 1 {
		t.Fatalf("expected 1 container detail, got %d", len(net.ContainerDetails))
	}
	cd := net.ContainerDetails[0]
	if cd.Name != "api-server" || cd.IPv4Address != "172.28.5.2/16" || cd.MacAddress != "02:42:ac:1c:05:02" {
		t.Errorf("unexpected container detail: %+v", cd)
	}
}
