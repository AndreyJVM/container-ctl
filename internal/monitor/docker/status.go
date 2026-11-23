package docker

import "github.com/AndreyJVM/container-ctl/internal/monitor/docker/types"

// ParseHealthStatus extracts a standardized healthcheck status from status string.
func ParseHealthStatus(val string) string {
	return types.ParseHealthStatus(val)
}

// ParseExitCode extracts whether an exit code is present and its integer value.
func ParseExitCode(status string) (bool, int) {
	return types.ParseExitCode(status)
}

// IsContainerFailed returns true and the exit code if the container is in an error state.
func IsContainerFailed(state, status string) (bool, int) {
	return types.IsContainerFailed(state, status)
}
