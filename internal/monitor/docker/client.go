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
		cli := GetContainerCLI()
		if ctx == nil {
			return exec.Command(cli, args...).CombinedOutput()
		}
		return exec.CommandContext(ctx, cli, args...).CombinedOutput()
	}
)

// SetCustomSocketPath sets a custom Docker socket path override.
func SetCustomSocketPath(path string) {
	customSocketPath = strings.TrimSpace(path)
	ResetSocketClientsPool()
}

// GetContainerCLI returns the available CLI binary name ("docker" or "podman").
// If "docker" is found in PATH, it is preferred; otherwise if "podman" is present, it is used.
func GetContainerCLI() string {
	if _, err := ExecLookPath("docker"); err == nil {
		return "docker"
	}
	if _, err := ExecLookPath("podman"); err == nil {
		return "podman"
	}
	return "docker"
}

// IsPodmanSocket returns true if the specified socket path indicates a Podman engine.
func IsPodmanSocket(sockPath string) bool {
	return strings.Contains(strings.ToLower(sockPath), "podman")
}

// FindDockerSocket finds an available Docker or Podman Unix socket.
func FindDockerSocket() string {
	if customSocketPath != "" {
		p := filepath.Clean(strings.TrimPrefix(customSocketPath, "unix://"))
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// 1. DOCKER_HOST environment variable override
	if dockerHost := os.Getenv("DOCKER_HOST"); dockerHost != "" {
		p := strings.TrimPrefix(dockerHost, "unix://")
		p = strings.TrimPrefix(p, "unix:")
		cleaned := filepath.Clean(p)
		if filepath.IsAbs(cleaned) {
			if _, err := os.Stat(cleaned); err == nil { // #nosec G703
				return cleaned
			}
		}
	}

	// 2. CONTAINER_HOST environment variable override (Podman standard)
	if containerHost := os.Getenv("CONTAINER_HOST"); containerHost != "" {
		p := strings.TrimPrefix(containerHost, "unix://")
		p = strings.TrimPrefix(p, "unix:")
		cleaned := filepath.Clean(p)
		if filepath.IsAbs(cleaned) {
			if _, err := os.Stat(cleaned); err == nil { // #nosec G703
				return cleaned
			}
		}
	}

	// 3. Standard system Docker sockets
	candidates := []string{
		"/var/run/docker.sock",
		"/run/docker.sock",
	}

	// 4. Rootless Podman and Docker sockets via XDG_RUNTIME_DIR and /run/user/$UID
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		candidates = append(candidates,
			filepath.Join(xdg, "podman", "podman.sock"),
			filepath.Join(xdg, "podman.sock"),
			filepath.Join(xdg, "docker.sock"),
		)
	}

	uid := os.Getuid()
	candidates = append(candidates,
		fmt.Sprintf("/run/user/%d/podman/podman.sock", uid),
		fmt.Sprintf("/run/user/%d/podman.sock", uid),
		fmt.Sprintf("/run/user/%d/docker.sock", uid),
		"/run/podman/podman.sock",
		"/var/run/podman/podman.sock",
	)

	// 5. User home directory machine sockets (Podman machine & Docker Desktop)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local/share/containers/podman/machine/podman.sock"),
			filepath.Join(home, ".local/share/containers/podman/machine/qemu/podman.sock"),
			filepath.Join(home, ".local/share/containers/podman/machine/applehv/podman.sock"),
			filepath.Join(home, ".docker/run/docker.sock"),
			filepath.Join(home, ".docker/desktop/docker.sock"),
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
	if _, err := ExecLookPath("docker"); err == nil {
		return true
	}
	if _, err := ExecLookPath("podman"); err == nil {
		return true
	}
	return false
}

// IsDockerCLIAvailable checks availability of the docker or podman CLI in PATH.
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
