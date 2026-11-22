package swarm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker"
)

func TestScaleSwarmService_Validation(t *testing.T) {
	t.Parallel()

	if err := ScaleSwarmService("", 3); err == nil || !strings.Contains(err.Error(), "service ID cannot be empty") {
		t.Errorf("expected error for empty service ID, got %v", err)
	}

	if err := ScaleSwarmService("   ", 3); err == nil || !strings.Contains(err.Error(), "service ID cannot be empty") {
		t.Errorf("expected error for whitespace service ID, got %v", err)
	}

	if err := ScaleSwarmService("svc1", -1); err == nil || !strings.Contains(err.Error(), "replicas cannot be negative") {
		t.Errorf("expected error for negative replicas, got %v", err)
	}
}

func TestScaleSwarmServiceViaSocket_Success(t *testing.T) {
	var receivedReplicas float64
	var receivedVersion string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/services/web-svc":
			spec := map[string]interface{}{
				"Name": "web-svc",
				"Mode": map[string]interface{}{
					"Replicated": map[string]interface{}{
						"Replicas": 1,
					},
				},
			}
			resp := map[string]interface{}{
				"ID": "web-svc",
				"Version": map[string]interface{}{
					"Index": 42,
				},
				"Spec": spec,
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.Method == http.MethodPost && r.URL.Path == "/services/web-svc/update":
			receivedVersion = r.URL.Query().Get("version")
			var updatedSpec map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&updatedSpec); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if mode, ok := updatedSpec["Mode"].(map[string]interface{}); ok {
				if repl, ok := mode["Replicated"].(map[string]interface{}); ok {
					if v, ok := repl["Replicas"].(float64); ok {
						receivedReplicas = v
					}
				}
			}
			w.WriteHeader(http.StatusOK)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := server.Client()
	// Override URL to route through test server
	origFactory := docker.SocketClientFactory
	origFinder := docker.DockerSocketFinder
	defer func() {
		docker.SocketClientFactory = origFactory
		docker.DockerSocketFinder = origFinder
	}()

	docker.DockerSocketFinder = func() string { return "/mock/docker.sock" }
	docker.SocketClientFactory = func(_ string) *http.Client { return client }

	// Test scaleSwarmServiceViaSocket directly with custom transport targeting test server
	err := scaleSwarmServiceViaSocket(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server.Listener.Addr().String()
			return server.Client().Transport.RoundTrip(req)
		}),
	}, "web-svc", 5)

	if err != nil {
		t.Fatalf("unexpected error scaling via socket: %v", err)
	}
	if receivedVersion != "42" {
		t.Errorf("expected version 42, got %q", receivedVersion)
	}
	if receivedReplicas != 5 {
		t.Errorf("expected 5 replicas received by server, got %v", receivedReplicas)
	}
}

func TestScaleSwarmServiceViaSocket_Errors(t *testing.T) {
	t.Parallel()

	// Nil client
	if err := scaleSwarmServiceViaSocket(nil, "svc", 2); err == nil {
		t.Errorf("expected error for nil http client")
	}

	// Server returns 404 on inspect
	server404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer server404.Close()

	client404 := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server404.Listener.Addr().String()
			return server404.Client().Transport.RoundTrip(req)
		}),
	}

	if err := scaleSwarmServiceViaSocket(client404, "missing-svc", 3); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("expected 404 error from scaleSwarmServiceViaSocket, got %v", err)
	}

	// Server returns 500 on update
	server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"Version": map[string]interface{}{"Index": 1},
				"Spec":    map[string]interface{}{},
			})
			return
		}
		http.Error(w, "raft conflict", http.StatusInternalServerError)
	}))
	defer server500.Close()

	client500 := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			req.URL.Scheme = "http"
			req.URL.Host = server500.Listener.Addr().String()
			return server500.Client().Transport.RoundTrip(req)
		}),
	}

	if err := scaleSwarmServiceViaSocket(client500, "err-svc", 3); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 error on update, got %v", err)
	}
}

func TestScaleSwarmService_CLIFallback(t *testing.T) {
	origFinder := docker.DockerSocketFinder
	origRunner := docker.DockerCLIRunner
	defer func() {
		docker.DockerSocketFinder = origFinder
		docker.DockerCLIRunner = origRunner
	}()

	docker.DockerSocketFinder = func() string { return "" } // No socket, force CLI

	var calledArgs []string
	docker.DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		calledArgs = args
		return []byte("web-service scaled to 4\n"), nil
	}

	err := ScaleSwarmService("web-service", 4)
	if err != nil {
		t.Fatalf("unexpected error on CLI fallback: %v", err)
	}
	expectedCmd := "service scale web-service=4"
	if strings.Join(calledArgs, " ") != expectedCmd {
		t.Errorf("expected CLI command %q, got %q", expectedCmd, strings.Join(calledArgs, " "))
	}

	// Error case from CLI
	docker.DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("Error response from daemon: service not found\n"), errors.New("exit status 1")
	}
	err = ScaleSwarmService("bad-service", 2)
	if err == nil || !strings.Contains(err.Error(), "service not found") {
		t.Errorf("expected service not found error, got %v", err)
	}
}
