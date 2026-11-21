package docker

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	SocketDialTimeout   = 500 * time.Millisecond
	SocketClientTimeout = 5 * time.Second
)

var (
	customSocketPath    string
	socketClientsMu     sync.Mutex
	socketClients       = make(map[string]*http.Client)
	ExecLookPath        = exec.LookPath
	DockerSocketFinder  = FindDockerSocket
	SocketClientFactory = getSocketHTTPClient
	DockerCLIRunner     = func(ctx context.Context, args ...string) ([]byte, error) {
		if ctx == nil {
			return exec.Command("docker", args...).CombinedOutput()
		}
		return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	}
)

// SetCustomSocketPath sets a custom Docker socket path override.
func SetCustomSocketPath(path string) {
	customSocketPath = strings.TrimSpace(path)
	ResetSocketClientsPool()
}

// FindDockerSocket finds an available Docker or Podman Unix socket.
func FindDockerSocket() string {
	if customSocketPath != "" {
		p := filepath.Clean(strings.TrimPrefix(customSocketPath, "unix://"))
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	if dockerHost := os.Getenv("DOCKER_HOST"); strings.HasPrefix(dockerHost, "unix://") {
		p := filepath.Clean(strings.TrimPrefix(dockerHost, "unix://"))
		if filepath.IsAbs(p) {
			if _, err := os.Stat(p); err == nil { // #nosec G703 - docker socket path is cleaned and checked
				return p
			}
		}
	}

	candidates := []string{
		"/var/run/docker.sock",
		"/run/docker.sock",
	}

	uid := os.Getuid()
	candidates = append(candidates,
		fmt.Sprintf("/run/user/%d/docker.sock", uid),
		fmt.Sprintf("/run/user/%d/podman/podman.sock", uid),
		"/run/podman/podman.sock",
	)

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".docker/run/docker.sock"),
			filepath.Join(home, ".docker/desktop/docker.sock"),
			filepath.Join(home, ".local/share/containers/podman/machine/podman.sock"),
		)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func getSocketHTTPClient(sockPath string) *http.Client {
	socketClientsMu.Lock()
	defer socketClientsMu.Unlock()

	if client, ok := socketClients[sockPath]; ok {
		return client
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				d.Timeout = SocketDialTimeout
				return d.DialContext(ctx, "unix", sockPath)
			},
			MaxIdleConns:        30,
			MaxIdleConnsPerHost: 30,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: SocketClientTimeout,
	}
	socketClients[sockPath] = client
	return client
}

// ResetSocketClientsPool closes idle connections and clears the client cache.
func ResetSocketClientsPool() {
	socketClientsMu.Lock()
	defer socketClientsMu.Unlock()
	for _, client := range socketClients {
		if tr, ok := client.Transport.(*http.Transport); ok {
			tr.CloseIdleConnections()
		}
	}
	socketClients = make(map[string]*http.Client)
}

var dockerCLIAvailableChecker = func() bool {
	_, err := ExecLookPath("docker")
	return err == nil
}

// IsDockerCLIAvailable checks availability of the docker CLI in PATH.
func IsDockerCLIAvailable() bool {
	return dockerCLIAvailableChecker()
}

// SetDockerCLIAvailableCheckerForTesting overrides the docker CLI check for unit tests and returns a restore function.
func SetDockerCLIAvailableCheckerForTesting(checker func() bool) func() {
	orig := dockerCLIAvailableChecker
	dockerCLIAvailableChecker = checker
	return func() {
		dockerCLIAvailableChecker = orig
	}
}
