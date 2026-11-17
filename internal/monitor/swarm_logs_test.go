package monitor

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestGetSwarmServiceLogs_Socket(t *testing.T) {
	origFinder := dockerSocketFinder
	origFactory := socketClientFactory
	defer func() {
		dockerSocketFinder = origFinder
		socketClientFactory = origFactory
	}()

	dockerSocketFinder = func() string { return "/tmp/mock-docker.sock" }
	socketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/services/svc-1/logs") {
				return mockStringResponse(http.StatusOK, "Swarm service live logs output\nLine 2"), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	logs, err := GetSwarmServiceLogs("svc-1", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(logs, "Swarm service live logs output") {
		t.Errorf("unexpected logs: %s", logs)
	}
}

func TestGetSwarmServiceLogs_CLIFallback(t *testing.T) {
	origFinder := dockerSocketFinder
	origRunner := dockerCLICommandRunner
	defer func() {
		dockerSocketFinder = origFinder
		dockerCLICommandRunner = origRunner
	}()

	dockerSocketFinder = func() string { return "" }
	dockerCLICommandRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("CLI Swarm log line 1\nCLI Swarm log line 2"), nil
	}

	logs, err := GetSwarmServiceLogs("svc-2", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(logs, "CLI Swarm log line 1") {
		t.Errorf("unexpected CLI logs: %s", logs)
	}
}

func TestGetSwarmServiceLogs_CLIError(t *testing.T) {
	origFinder := dockerSocketFinder
	origRunner := dockerCLICommandRunner
	defer func() {
		dockerSocketFinder = origFinder
		dockerCLICommandRunner = origRunner
	}()

	dockerSocketFinder = func() string { return "" }
	dockerCLICommandRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("Error: no such service: svc-3"), errors.New("exit status 1")
	}

	_, err := GetSwarmServiceLogs("svc-3", 10)
	if err == nil {
		t.Errorf("expected error when fetching logs for non-existent service")
	}
}

func TestGetSwarmServiceLogs_DefaultTail(t *testing.T) {
	origFinder := dockerSocketFinder
	origRunner := dockerCLICommandRunner
	defer func() {
		dockerSocketFinder = origFinder
		dockerCLICommandRunner = origRunner
	}()

	dockerSocketFinder = func() string { return "" }
	var capturedArgs []string
	dockerCLICommandRunner = func(_ context.Context, args ...string) ([]byte, error) {
		capturedArgs = args
		return []byte("log output"), nil
	}

	_, err := GetSwarmServiceLogs("svc-test", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Check that tail defaulted to 100
	foundTail := false
	for i, arg := range capturedArgs {
		if arg == "--tail" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "100" {
			foundTail = true
			break
		}
	}
	if !foundTail {
		t.Errorf("expected default tail 100 in args: %v", capturedArgs)
	}
}

func TestGetSwarmServiceLogs_EmptyID(t *testing.T) {
	if _, err := GetSwarmServiceLogs("   ", 100); err == nil {
		t.Errorf("expected error for empty service ID")
	}
}
