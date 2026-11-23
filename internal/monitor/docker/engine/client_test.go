package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func mockStringResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestFindDockerSocket(t *testing.T) {
	sock := FindDockerSocket()
	t.Logf("Detected Docker socket: '%s'", sock)

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
	_ = IsDockerCLIAvailable()
}

func TestSocketClientPool(t *testing.T) {
	ResetSocketClientsPool()

	sockPath := "/var/run/docker.sock"
	c1 := GetSocketHTTPClient(sockPath)
	c2 := GetSocketHTTPClient(sockPath)

	if c1 != c2 {
		t.Errorf("expected pooled http.Client to return identical pointer, got %p vs %p", c1, c2)
	}

	c3 := GetSocketHTTPClient("/run/podman/podman.sock")
	if c1 == c3 {
		t.Errorf("different socket paths must return distinct clients")
	}

	ResetSocketClientsPool()
}

func TestCanConnectToSocket(t *testing.T) {
	if CanConnectToSocket("") {
		t.Errorf("expected empty path to return false")
	}
	if CanConnectToSocket("/nonexistent/socket.sock") {
		t.Errorf("expected nonexistent socket to return false")
	}

	tmpDir := t.TempDir()
	sockPath := tmpDir + "/test.sock"
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to create unix listener: %v", err)
	}
	defer listener.Close()

	if !CanConnectToSocket(sockPath) {
		t.Errorf("expected active unix listener to return true")
	}
}

func TestSocketClassification(t *testing.T) {
	if !IsPodmanSocket("/run/podman/podman.sock") {
		t.Errorf("expected podman socket to be detected")
	}
	if IsPodmanSocket("/var/run/docker.sock") {
		t.Errorf("docker socket detected as podman")
	}

	if !IsRootlessSocket("/run/user/1000/docker.sock") {
		t.Errorf("expected rootless socket to be detected")
	}
	if IsRootlessSocket("/var/run/docker.sock") {
		t.Errorf("system socket detected as rootless")
	}
}

func TestCustomSocketPath(_ *testing.T) {
	SetCustomSocketPath("/custom/docker.sock")
	defer SetCustomSocketPath("")
}

func TestDefaultCLIRunner(_ *testing.T) {
	ctx := context.Background()
	_, _ = DefaultCLIRunner(ctx, "--version")
	_, _ = DefaultCLIRunner(nil, "--version")
}

func TestFetchDockerResourceJSON(t *testing.T) {
	origFactory := SocketClientFactory
	defer func() { SocketClientFactory = origFactory }()

	SocketClientFactory = func(_ string) *http.Client {
		return newMockHTTPClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/test" {
				data, _ := json.Marshal(map[string]string{"key": "value"})
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(data)),
				}, nil
			}
			return mockStringResponse(http.StatusNotFound, "not found"), nil
		})
	}

	var res map[string]string
	err := FetchDockerResourceJSON("/tmp/mock.sock", "/test", &res)
	if err != nil {
		t.Fatalf("FetchDockerResourceJSON failed: %v", err)
	}
	if res["key"] != "value" {
		t.Errorf("expected value, got %s", res["key"])
	}

	err = FetchDockerResourceJSON("/tmp/mock.sock", "/bad", &res)
	if err == nil {
		t.Errorf("expected error for 404 response")
	}
}
