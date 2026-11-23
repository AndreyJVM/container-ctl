package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	SocketDialTimeout    = 500 * time.Millisecond
	SocketClientTimeout  = 5 * time.Second
	DefaultDockerTimeout = 5 * time.Second
)

var (
	customSocketPath       string
	socketClientsMu        sync.Mutex
	socketClients          = make(map[string]*http.Client)
	dockerSocketFinderPtr  *func() string
	socketClientFactoryPtr *func(string) *http.Client
	dockerCLIRunnerPtr     *func(context.Context, ...string) ([]byte, error)
	execLookPathPtr        *func(string) (string, error)

	DockerSocketFinder = func() string {
		if dockerSocketFinderPtr != nil && *dockerSocketFinderPtr != nil {
			return (*dockerSocketFinderPtr)()
		}
		return FindDockerSocket()
	}
	SocketClientFactory = func(sockPath string) *http.Client {
		if socketClientFactoryPtr != nil && *socketClientFactoryPtr != nil {
			return (*socketClientFactoryPtr)(sockPath)
		}
		return GetSocketHTTPClient(sockPath)
	}
	DockerCLIRunner = func(ctx context.Context, args ...string) ([]byte, error) {
		if dockerCLIRunnerPtr != nil && *dockerCLIRunnerPtr != nil {
			return (*dockerCLIRunnerPtr)(ctx, args...)
		}
		return DefaultCLIRunner(ctx, args...)
	}
	ExecLookPath = func(file string) (string, error) {
		if execLookPathPtr != nil && *execLookPathPtr != nil {
			return (*execLookPathPtr)(file)
		}
		return exec.LookPath(file)
	}
)

// RegisterHookPointers registers pointers to root docker variables to synchronize mocks across packages.
func RegisterHookPointers(
	sf *func() string,
	scf *func(string) *http.Client,
	cr *func(context.Context, ...string) ([]byte, error),
	elp *func(string) (string, error),
) {
	dockerSocketFinderPtr = sf
	socketClientFactoryPtr = scf
	dockerCLIRunnerPtr = cr
	execLookPathPtr = elp
}

// DefaultCLIRunner executes a command using the detected container CLI.
func DefaultCLIRunner(ctx context.Context, args ...string) ([]byte, error) {
	cli := GetContainerCLI()
	if ctx == nil {
		return exec.Command(cli, args...).CombinedOutput()
	}
	return exec.CommandContext(ctx, cli, args...).CombinedOutput()
}

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

// IsRootlessSocket returns true if the socket is located in a per-user rootless runtime directory.
func IsRootlessSocket(sockPath string) bool {
	p := strings.ToLower(sockPath)
	return strings.Contains(p, "/run/user/") ||
		strings.Contains(p, "/.local/share/containers/") ||
		strings.Contains(p, "/.docker/")
}

// CanConnectToSocket checks if a Unix socket is actively listening and accessible.
func CanConnectToSocket(sockPath string) bool {
	if sockPath == "" {
		return false
	}
	conn, err := net.DialTimeout("unix", sockPath, 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func deduplicatePaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	var res []string
	for _, p := range paths {
		cleaned := filepath.Clean(p)
		if cleaned != "" && cleaned != "." && !seen[cleaned] {
			seen[cleaned] = true
			res = append(res, cleaned)
		}
	}
	return res
}

// GetCandidateSockets returns an ordered list of candidate Unix socket paths.
func GetCandidateSockets() []string {
	var candidates []string

	// Rootless Podman and Docker sockets via XDG_RUNTIME_DIR and /run/user/$UID
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
	)

	// Standard system Docker and Podman sockets
	candidates = append(candidates,
		"/var/run/docker.sock",
		"/run/docker.sock",
		"/run/podman/podman.sock",
		"/var/run/podman/podman.sock",
	)

	// User home directory machine sockets (Podman machine & Docker Desktop)
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".local/share/containers/podman/machine/podman.sock"),
			filepath.Join(home, ".local/share/containers/podman/machine/applehv/podman.sock"),
			filepath.Join(home, ".docker/run/docker.sock"),
			filepath.Join(home, ".docker/desktop/docker.sock"),
		)
	}

	return deduplicatePaths(candidates)
}

// FindDockerSocket finds an available Docker or Podman Unix socket.
func FindDockerSocket() string {
	// Explicit overrides:
	if customSocketPath != "" {
		p := filepath.Clean(strings.TrimPrefix(customSocketPath, "unix://"))
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

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

	if podmanSocket := os.Getenv("PODMAN_SOCKET"); podmanSocket != "" {
		p := strings.TrimPrefix(podmanSocket, "unix://")
		p = strings.TrimPrefix(p, "unix:")
		cleaned := filepath.Clean(p)
		if filepath.IsAbs(cleaned) {
			if _, err := os.Stat(cleaned); err == nil { // #nosec G703
				return cleaned
			}
		}
	}

	candidates := GetCandidateSockets()

	// 1. First pass: find an active, listening socket that we can connect to
	for _, p := range candidates {
		if CanConnectToSocket(p) {
			return p
		}
	}

	// 2. Second pass: fallback to the first candidate that exists on the filesystem
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// GetSocketHTTPClient returns an http.Client connected via Unix socket dialer.
func GetSocketHTTPClient(sockPath string) *http.Client {
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

// FetchDockerResourceJSON performs an HTTP GET over the Docker socket and deserializes the JSON response.
func FetchDockerResourceJSON(sockPath, path string, target interface{}) error {
	client := SocketClientFactory(sockPath)
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDockerTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+path, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Docker API %s returned status %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}
