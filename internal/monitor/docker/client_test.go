package docker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
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
