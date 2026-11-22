package docker

import (
	"os/exec"
	"strings"
)

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

	cli := GetContainerCLI()

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
	return GetContainerExecCommandWithOptions(containerID, ExecOptions{Mode: "auto"})
}
