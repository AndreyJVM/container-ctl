package operations

import (
	"os/exec"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/engine"
	"github.com/AndreyJVM/container-ctl/internal/monitor/docker/types"
)

// GetContainerExecCommandWithOptions creates an exec.Cmd based on specified ExecOptions.
func GetContainerExecCommandWithOptions(containerID string, opts types.ExecOptions) *exec.Cmd {
	cleanID := strings.TrimSpace(containerID)
	if cleanID == "" {
		return nil
	}

	cli := engine.GetContainerCLI()

	if opts.Mode == "attach" {
		return exec.Command(cli, "attach", "--sig-proxy=false", cleanID)
	}

	args := []string{"exec", "-it"}
	user := opts.User
	if opts.Mode == "root" && user == "" {
		user = "0"
	}
	if user != "" {
		args = append(args, "-u", user)
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
	case "custom":
		cmdStr := strings.TrimSpace(opts.CustomCmd)
		if cmdStr == "" {
			cmdStr = "sh"
		}
		args = append(args, "sh", "-c", cmdStr)
	case "root", "auto":
		fallthrough
	default:
		args = append(args, "sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi")
	}

	return exec.Command(cli, args...)
}

// GetContainerExecCommand creates an exec.Cmd for an interactive container shell.
func GetContainerExecCommand(containerID string) *exec.Cmd {
	return GetContainerExecCommandWithOptions(containerID, types.ExecOptions{Mode: "auto"})
}
