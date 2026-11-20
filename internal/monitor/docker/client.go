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
	"time"
)

const (
	SocketDialTimeout   = 500 * time.Millisecond
	SocketClientTimeout = 5 * time.Second
)

var (
	customSocketPath    string
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
			if _, err := os.Stat(p); err == nil {
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
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				d.Timeout = SocketDialTimeout
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
		Timeout: SocketClientTimeout,
	}
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

// ExecOptions specifies options for launching an interactive session in a container.
type ExecOptions struct {
	Mode      string // "auto", "sh", "zsh", "root", "attach", "custom"
	User      string // optional user (e.g., "0" or "root")
	CustomCmd string // optional custom command when Mode == "custom"
	Terminal  string // optional TERM value for embedded terminal emulation
}

// GetContainerExecCommandWithOptions creates an exec.Cmd based on specified ExecOptions.
func GetContainerExecCommandWithOptions(containerID string, opts ExecOptions) *exec.Cmd {
	cleanID := strings.TrimSpace(containerID)
	if cleanID == "" {
		return nil
	}

	if opts.Mode == "attach" {
		return exec.Command("docker", "attach", "--sig-proxy=false", cleanID)
	}

	args := []string{"exec", "-it"}
	if opts.User != "" {
		args = append(args, "-u", opts.User)
	}
	if terminal := strings.TrimSpace(opts.Terminal); terminal != "" {
		args = append(args, "-e", "TERM="+terminal)
	}
	args = append(args, cleanID)

	switch opts.Mode {
	case "sh":
		args = append(args, "sh")
	case "zsh":
		args = append(args, "zsh")
	case "root":
		if opts.User == "" {
			args = []string{"exec", "-it", "-u", "0", cleanID, "sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi"}
		} else {
			args = append(args, "sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi")
		}
	case "custom":
		cmdStr := strings.TrimSpace(opts.CustomCmd)
		if cmdStr == "" {
			cmdStr = "sh"
		}
		args = append(args, "sh", "-c", cmdStr)
	case "auto":
		fallthrough
	default:
		args = append(args, "sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi")
	}

	return exec.Command("docker", args...)
}

// GetContainerExecCommand creates an exec.Cmd for an interactive container shell.
func GetContainerExecCommand(containerID string) *exec.Cmd {
	return GetContainerExecCommandWithOptions(containerID, ExecOptions{Mode: "auto"})
}
