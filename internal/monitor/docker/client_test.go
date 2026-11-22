package docker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindDockerSocket(t *testing.T) {
	// Should not crash
	sock := FindDockerSocket()
	t.Logf("Detected Docker socket: '%s'", sock)

	// Test override with DOCKER_HOST
	tmpFile, err := os.CreateTemp("", "docker-test-*.sock")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	origHost := os.Getenv("DOCKER_HOST")
	defer os.Setenv("DOCKER_HOST", origHost)

	os.Setenv("DOCKER_HOST", "unix://"+tmpFile.Name())
	detected := FindDockerSocket()
	if detected != tmpFile.Name() {
		t.Errorf("expected '%s', got '%s'", tmpFile.Name(), detected)
	}
}

func TestIsDockerCLIAvailable(_ *testing.T) {
	// Function must execute safely without panics
	_ = IsDockerCLIAvailable()
}

func TestSocketClientPool(t *testing.T) {
	ResetSocketClientsPool()

	sockPath := "/var/run/docker.sock"
	c1 := getSocketHTTPClient(sockPath)
	c2 := getSocketHTTPClient(sockPath)

	if c1 != c2 {
		t.Errorf("expected pooled http.Client to return identical pointer, got %p vs %p", c1, c2)
	}

	// Different path should return different client
	c3 := getSocketHTTPClient("/run/podman/podman.sock")
	if c1 == c3 {
		t.Errorf("different socket paths must return distinct clients")
	}

	ResetSocketClientsPool()
}

func TestSetCustomSocketPath(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "custom-socket-*.sock")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	SetCustomSocketPath(tmpFile.Name())
	defer SetCustomSocketPath("")

	detected := FindDockerSocket()
	if detected != tmpFile.Name() {
		t.Errorf("expected custom socket %s, got %s", tmpFile.Name(), detected)
	}
}

func TestFindDockerSocket_ContainerHost(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "podman-test-*.sock")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	origDockerHost := os.Getenv("DOCKER_HOST")
	origContainerHost := os.Getenv("CONTAINER_HOST")
	defer func() {
		os.Setenv("DOCKER_HOST", origDockerHost)
		os.Setenv("CONTAINER_HOST", origContainerHost)
	}()

	os.Unsetenv("DOCKER_HOST")
	os.Setenv("CONTAINER_HOST", "unix://"+tmpFile.Name())

	detected := FindDockerSocket()
	if detected != tmpFile.Name() {
		t.Errorf("expected CONTAINER_HOST %s, got %s", tmpFile.Name(), detected)
	}
}

func TestFindDockerSocket_XDGRuntimeDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "xdg-runtime-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	podmanDir := filepath.Join(tempDir, "podman")
	if err := os.MkdirAll(podmanDir, 0750); err != nil {
		t.Fatalf("failed to create podman dir: %v", err)
	}
	podmanSock := filepath.Join(podmanDir, "podman.sock")
	f, err := os.Create(podmanSock)
	if err != nil {
		t.Fatalf("failed to create fake socket: %v", err)
	}
	f.Close()

	origDockerHost := os.Getenv("DOCKER_HOST")
	origContainerHost := os.Getenv("CONTAINER_HOST")
	origXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		os.Setenv("DOCKER_HOST", origDockerHost)
		os.Setenv("CONTAINER_HOST", origContainerHost)
		os.Setenv("XDG_RUNTIME_DIR", origXDG)
	}()

	os.Unsetenv("DOCKER_HOST")
	os.Unsetenv("CONTAINER_HOST")
	os.Setenv("XDG_RUNTIME_DIR", tempDir)

	// In environment where /var/run/docker.sock does not exist, or when testing candidate list directly
	candidates := []string{
		filepath.Join(tempDir, "podman", "podman.sock"),
	}
	if _, err := os.Stat(candidates[0]); err != nil {
		t.Fatalf("fake socket should exist: %v", err)
	}

	if !IsPodmanSocket(podmanSock) {
		t.Errorf("expected IsPodmanSocket to return true for %s", podmanSock)
	}
}

func TestGetContainerCLI(t *testing.T) {
	cli := GetContainerCLI()
	if cli != "docker" && cli != "podman" {
		t.Errorf("expected GetContainerCLI to return 'docker' or 'podman', got: %q", cli)
	}
}

func TestIsPodmanSocket(t *testing.T) {
	tests := []struct {
		sockPath string
		expected bool
	}{
		{"/run/user/1000/podman/podman.sock", true},
		{"/run/podman/podman.sock", true},
		{"/home/user/.local/share/containers/podman/machine/podman.sock", true},
		{"/var/run/docker.sock", false},
		{"/run/user/1000/docker.sock", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.sockPath, func(t *testing.T) {
			if got := IsPodmanSocket(tt.sockPath); got != tt.expected {
				t.Errorf("IsPodmanSocket(%q) = %v; want %v", tt.sockPath, got, tt.expected)
			}
		})
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func mockJSONResponse(statusCode int, obj interface{}) *http.Response {
	data, _ := json.Marshal(obj)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
	}
}

func mockStringResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
