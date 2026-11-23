package docker

import (
	"net/http"
	"os/exec"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/engine"
)

const (
	SocketDialTimeout   = engine.SocketDialTimeout
	SocketClientTimeout = engine.SocketClientTimeout
)

var (
	DockerSocketFinder  = engine.FindDockerSocket
	SocketClientFactory = engine.GetSocketHTTPClient
	DockerCLIRunner     = engine.DefaultCLIRunner
	ExecLookPath        = exec.LookPath
)

func init() {
	engine.RegisterHookPointers(
		&DockerSocketFinder,
		&SocketClientFactory,
		&DockerCLIRunner,
		&ExecLookPath,
	)
}

// SetCustomSocketPath sets a custom Docker socket path override.
func SetCustomSocketPath(path string) {
	engine.SetCustomSocketPath(path)
}

// GetContainerCLI returns the available CLI binary name ("docker" or "podman").
func GetContainerCLI() string {
	return engine.GetContainerCLI()
}

// IsPodmanSocket returns true if the specified socket path indicates a Podman engine.
func IsPodmanSocket(sockPath string) bool {
	return engine.IsPodmanSocket(sockPath)
}

// IsRootlessSocket returns true if the socket is located in a per-user rootless runtime directory.
func IsRootlessSocket(sockPath string) bool {
	return engine.IsRootlessSocket(sockPath)
}

// CanConnectToSocket checks if a Unix socket is actively listening and accessible.
func CanConnectToSocket(sockPath string) bool {
	return engine.CanConnectToSocket(sockPath)
}

// GetCandidateSockets returns an ordered list of candidate Unix socket paths.
func GetCandidateSockets() []string {
	return engine.GetCandidateSockets()
}

// FindDockerSocket finds an available Docker or Podman Unix socket.
func FindDockerSocket() string {
	return DockerSocketFinder()
}

func getSocketHTTPClient(sockPath string) *http.Client {
	return SocketClientFactory(sockPath)
}

// ResetSocketClientsPool closes idle connections and clears the client cache.
func ResetSocketClientsPool() {
	engine.ResetSocketClientsPool()
}

// IsDockerCLIAvailable checks availability of the docker or podman CLI in PATH.
func IsDockerCLIAvailable() bool {
	return engine.IsDockerCLIAvailable()
}

// SetDockerCLIAvailableCheckerForTesting overrides the docker CLI check for unit tests.
func SetDockerCLIAvailableCheckerForTesting(checker func() bool) func() {
	return engine.SetDockerCLIAvailableCheckerForTesting(checker)
}
