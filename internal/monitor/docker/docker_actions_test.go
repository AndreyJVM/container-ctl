package docker

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestContainerOperationsEmptyID(t *testing.T) {
	emptyID := "   "

	if err := StartContainer(emptyID); err == nil {
		t.Errorf("expected error when starting empty container ID")
	}

	if err := StopContainer(emptyID); err == nil {
		t.Errorf("expected error when stopping empty container ID")
	}

	if err := RestartContainer(emptyID); err == nil {
		t.Errorf("expected error when restarting empty container ID")
	}

	if err := KillContainer(emptyID); err == nil {
		t.Errorf("expected error when killing empty container ID")
	}

	if err := RemoveContainer(emptyID); err == nil {
		t.Errorf("expected error when removing empty container ID")
	}

	if _, err := GetContainerLogs(emptyID, 10); err == nil {
		t.Errorf("expected error when fetching logs for empty container ID")
	}

	if _, err := GetContainerDetail(emptyID); err == nil {
		t.Errorf("expected error when fetching details for empty container ID")
	}

	if cmd := GetContainerExecCommand(emptyID); cmd != nil {
		t.Errorf("expected nil exec command for empty container ID")
	}
}

func TestContainerOperationsInvalidID(t *testing.T) {
	invalidID := "non-existent-container-99999"

	// All operations on invalid ID should safely return an error without panic
	if err := StartContainer(invalidID); err == nil {
		t.Errorf("expected error when starting invalid container ID")
	}

	if err := StopContainer(invalidID); err == nil {
		t.Errorf("expected error when stopping invalid container ID")
	}

	if err := RestartContainer(invalidID); err == nil {
		t.Errorf("expected error when restarting invalid container ID")
	}

	if err := KillContainer(invalidID); err == nil {
		t.Errorf("expected error when killing invalid container ID")
	}

	if err := RemoveContainer(invalidID); err == nil {
		t.Errorf("expected error when removing invalid container ID")
	}

	if _, err := GetContainerLogs(invalidID, 10); err == nil {
		t.Errorf("expected error when fetching logs for invalid container ID")
	}

	if _, err := GetContainerDetail(invalidID); err == nil {
		t.Errorf("expected error when fetching details for invalid container ID")
	}
}

func TestContainerActions_Socket(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/containers/test-c/") {
				return mockStringResponse(http.StatusNoContent, ""), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	if err := StartContainer("test-c"); err != nil {
		t.Errorf("StartContainer failed: %v", err)
	}
	if err := StopContainer("test-c"); err != nil {
		t.Errorf("StopContainer failed: %v", err)
	}
	if err := RestartContainer("test-c"); err != nil {
		t.Errorf("RestartContainer failed: %v", err)
	}
	if err := KillContainer("test-c"); err != nil {
		t.Errorf("KillContainer failed: %v", err)
	}
}

func TestContainerActions_CLIFallback(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }
	var lastAction string
	DockerCLIRunner = func(_ context.Context, args ...string) ([]byte, error) {
		if len(args) > 0 {
			lastAction = args[0]
		}
		return []byte("ok"), nil
	}

	if err := StartContainer("test-cli"); err != nil || lastAction != "start" {
		t.Errorf("StartContainer fallback failed: %v, lastAction: %s", err, lastAction)
	}
	if err := StopContainer("test-cli"); err != nil || lastAction != "stop" {
		t.Errorf("StopContainer fallback failed: %v, lastAction: %s", err, lastAction)
	}
	if err := RestartContainer("test-cli"); err != nil || lastAction != "restart" {
		t.Errorf("RestartContainer fallback failed: %v, lastAction: %s", err, lastAction)
	}
	if err := KillContainer("test-cli"); err != nil || lastAction != "kill" {
		t.Errorf("KillContainer fallback failed: %v, lastAction: %s", err, lastAction)
	}
}

func TestRemoveContainer_Socket(t *testing.T) {
	origFinder := DockerSocketFinder
	origFactory := SocketClientFactory
	defer func() {
		DockerSocketFinder = origFinder
		SocketClientFactory = origFactory
	}()

	DockerSocketFinder = func() string { return "/tmp/mock.sock" }
	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if req.Method == http.MethodDelete && strings.Contains(req.URL.Path, "/containers/test-rm") {
				return mockStringResponse(http.StatusNoContent, ""), nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	if err := RemoveContainer("test-rm"); err != nil {
		t.Errorf("RemoveContainer socket failed: %v", err)
	}
}

func TestRemoveContainer_CLIFallback(t *testing.T) {
	origFinder := DockerSocketFinder
	origRunner := DockerCLIRunner
	defer func() {
		DockerSocketFinder = origFinder
		DockerCLIRunner = origRunner
	}()

	DockerSocketFinder = func() string { return "" }
	DockerCLIRunner = func(_ context.Context, _ ...string) ([]byte, error) {
		return []byte("test-rm-cli"), nil
	}

	if err := RemoveContainer("test-rm-cli"); err != nil {
		t.Errorf("RemoveContainer CLI failed: %v", err)
	}
}

func TestGetContainerExecCommand(t *testing.T) {
	containerID := "abc123def456"
	cmd := GetContainerExecCommand(containerID)
	if cmd == nil {
		t.Fatal("expected non-nil *exec.Cmd")
	}

	expectedArgs := []string{
		"docker", "exec", "-it", containerID, "sh", "-c",
		"if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
	}

	if len(cmd.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d (%v)", len(expectedArgs), len(cmd.Args), cmd.Args)
	}

	for i, arg := range cmd.Args {
		if arg != expectedArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, arg, expectedArgs[i])
		}
	}
}

func TestGetContainerExecCommandWithTerminal(t *testing.T) {
	containerID := "abc123def456"
	cmd := GetContainerExecCommandWithOptions(containerID, ExecOptions{
		Mode:     "auto",
		Terminal: "xterm-256color",
	})
	if cmd == nil {
		t.Fatal("expected non-nil *exec.Cmd")
	}

	want := []string{
		"docker", "exec", "-it", "-e", "TERM=xterm-256color", containerID,
		"sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
	}
	if strings.Join(cmd.Args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected command args: %#v", cmd.Args)
	}
}

func TestCleanLogSnippet(t *testing.T) {
	// Empty input
	if got := CleanLogSnippet("", 3); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if got := CleanLogSnippet("  \n\n  ", 3); got != "" {
		t.Errorf("expected empty string for whitespace, got %q", got)
	}

	// Fewer lines than max
	input := "line 1\nline 2"
	if got := CleanLogSnippet(input, 3); got != "line 1\nline 2" {
		t.Errorf("unexpected snippet: %q", got)
	}

	// More lines than max
	inputLong := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	expected := "line 3\nline 4\nline 5"
	if got := CleanLogSnippet(inputLong, 3); got != expected {
		t.Errorf("CleanLogSnippet = %q, want %q", got, expected)
	}
}
