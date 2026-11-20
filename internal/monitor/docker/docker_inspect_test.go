package docker

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetContainerDetail_Validation(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tab and spaces", "\t \n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetContainerDetail(tt.id)
			if err == nil {
				t.Errorf("expected error for empty/whitespace container ID %q", tt.id)
			}
			if !strings.Contains(err.Error(), "container ID cannot be empty") {
				t.Errorf("unexpected error message: %v", err)
			}
		})
	}
}

func TestGetContainerDetail_Socket_Errors(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }

	t.Run("http client request error falls back to CLI", func(t *testing.T) {
		SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			})
		}
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte(`[{"Name":"/fallback-app","State":{"Status":"running"}}]`), nil
		}

		detail, err := GetContainerDetail("c-conn-err")
		if err != nil {
			t.Fatalf("expected CLI fallback on socket conn error, got: %v", err)
		}
		if detail.Name != "fallback-app" {
			t.Errorf("expected fallback-app, got: %s", detail.Name)
		}
	})

	t.Run("non-200 status code falls back to CLI", func(t *testing.T) {
		SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
				return mockStringResponse(http.StatusNotFound, "no such container"), nil
			})
		}
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte(`[{"Name":"/fallback-app-404","State":{"Status":"running"}}]`), nil
		}

		detail, err := GetContainerDetail("c-404")
		if err != nil {
			t.Fatalf("expected CLI fallback on 404, got: %v", err)
		}
		if detail.Name != "fallback-app-404" {
			t.Errorf("expected fallback-app-404, got: %s", detail.Name)
		}
	})

	t.Run("malformed json body falls back to CLI", func(t *testing.T) {
		SocketClientFactory = func(_ string) *http.Client {
			return newMockHTTPClient(func(_ *http.Request) (*http.Response, error) {
				return mockStringResponse(http.StatusOK, "{invalid-json"), nil
			})
		}
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte(`[{"Name":"/fallback-app-json","State":{"Status":"running"}}]`), nil
		}

		detail, err := GetContainerDetail("c-bad-json")
		if err != nil {
			t.Fatalf("expected CLI fallback on bad json, got: %v", err)
		}
		if detail.Name != "fallback-app-json" {
			t.Errorf("expected fallback-app-json, got: %s", detail.Name)
		}
	})
}

func TestGetContainerDetail_CLIFallback_Extended(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }

	t.Run("cli command error", func(t *testing.T) {
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return nil, errors.New("docker daemon down")
		}

		_, err := GetContainerDetail("c-cli-fail")
		if err == nil {
			t.Errorf("expected error when CLI inspect fails")
		}
		if !strings.Contains(err.Error(), "container not found: c-cli-fail") {
			t.Errorf("unexpected error text: %v", err)
		}
	})

	t.Run("cli empty json array or invalid format", func(t *testing.T) {
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte(`[]`), nil
		}

		_, err := GetContainerDetail("c-empty-arr")
		if err == nil {
			t.Errorf("expected error when CLI returns empty array")
		}
		if !strings.Contains(err.Error(), "container not found: c-empty-arr") {
			t.Errorf("unexpected error text: %v", err)
		}
	})

	t.Run("cli full inspect fields with restart policy and network settings", func(t *testing.T) {
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte(`[{
				"Name": "/db-server",
				"State": {
					"Status": "running",
					"Health": {"Status": "unhealthy"}
				},
				"HostConfig": {
					"RestartPolicy": {"Name": "unless-stopped"}
				},
				"NetworkSettings": {
					"Networks": {
						"custom_nw": {
							"IPAddress": "10.0.1.5",
							"Gateway": "10.0.1.1",
							"MacAddress": "02:42:0a:00:01:05"
						}
					}
				}
			}]`), nil
		}

		detail, err := GetContainerDetail("db-c")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.Name != "db-server" {
			t.Errorf("expected db-server, got %q", detail.Name)
		}
		if detail.State != "running" || detail.Status != "running" || detail.Health != "unhealthy" {
			t.Errorf("unexpected state/status/health: %+v", detail)
		}
		if detail.RestartPolicy != "unless-stopped" {
			t.Errorf("expected RestartPolicy unless-stopped, got %q", detail.RestartPolicy)
		}
		if detail.IP != "10.0.1.5" || detail.Gateway != "10.0.1.1" || detail.MACAddress != "02:42:0a:00:01:05" {
			t.Errorf("unexpected network values: IP=%s GW=%s MAC=%s", detail.IP, detail.Gateway, detail.MACAddress)
		}
	})

	t.Run("cli health inferred from status string if health block absent", func(t *testing.T) {
		DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte(`[{
				"Name": "/web-status-health",
				"State": {
					"Status": "Up 2 hours (healthy)"
				}
			}]`), nil
		}

		detail, err := GetContainerDetail("web-h")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detail.Health != "healthy" {
			t.Errorf("expected inferred healthy, got %q", detail.Health)
		}
	})
}

func TestGetNetworkInspectJSON_SocketAndCLI(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
		DockerCLIRunner = origRunner
	}()

	if _, err := GetNetworkInspectJSON(""); err == nil {
		t.Errorf("expected error on empty network id")
	}

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/networks/net-1") {
				return mockStringResponse(http.StatusOK, `{"Name":"bridge","Id":"net-1","Driver":"bridge"}`), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}
	out, err := GetNetworkInspectJSON("net-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"Name": "bridge"`) {
		t.Errorf("expected pretty JSON output, got: %s", out)
	}

	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "network" && args[1] == "inspect" && args[2] == "cli-net" {
			return []byte(`[{"Name":"cli-bridge"}]`), nil
		}
		return nil, errors.New("command failed")
	}
	cliOut, err := GetNetworkInspectJSON("cli-net")
	if err != nil {
		t.Fatalf("unexpected CLI error: %v", err)
	}
	if !strings.Contains(cliOut, "cli-bridge") {
		t.Errorf("expected cli-bridge in output, got: %s", cliOut)
	}
}

func TestGetVolumeInspectJSON_SocketAndCLI(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
		DockerCLIRunner = origRunner
	}()

	if _, err := GetVolumeInspectJSON(""); err == nil {
		t.Errorf("expected error on empty volume name")
	}

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

	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) >= 3 && args[0] == "volume" && args[1] == "inspect" && args[2] == "cli-vol" {
			return []byte(`[{"Name":"cli-vol"}]`), nil
		}
		return nil, errors.New("command failed")
	}
	cliOut, err := GetVolumeInspectJSON("cli-vol")
	if err != nil {
		t.Fatalf("unexpected CLI error: %v", err)
	}
	if !strings.Contains(cliOut, "cli-vol") {
		t.Errorf("expected cli-vol in output, got: %s", cliOut)
	}
}
