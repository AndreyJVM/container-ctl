package docker

import (
	"strconv"
	"strings"
)

// ParseHealthStatus extracts a standardized healthcheck status ("healthy", "unhealthy", "starting" or "")
// from Docker's State.Health.Status or container Status string.
func ParseHealthStatus(val string) string {
	s := strings.ToLower(strings.TrimSpace(val))
	if s == "" {
		return ""
	}
	switch s {
	case HealthStatusHealthy:
		return HealthStatusHealthy
	case HealthStatusUnhealthy:
		return HealthStatusUnhealthy
	case HealthStatusStarting:
		return HealthStatusStarting
	}

	if strings.Contains(s, "("+HealthStatusUnhealthy+")") || strings.Contains(s, "(health: "+HealthStatusUnhealthy+")") {
		return HealthStatusUnhealthy
	}
	if strings.Contains(s, "("+HealthStatusHealthy+")") || strings.Contains(s, "(health: "+HealthStatusHealthy+")") {
		return HealthStatusHealthy
	}
	if strings.Contains(s, "(health: "+HealthStatusStarting+")") || strings.Contains(s, "("+HealthStatusStarting+")") || (strings.Contains(s, HealthStatusStarting) && strings.Contains(s, "health")) {
		return HealthStatusStarting
	}

	return ""
}

// ParseExitCode extracts whether an exit code is present and its integer value
// from Docker status strings like "Exited (1) 2 minutes ago" or "Exited (0)".
func ParseExitCode(status string) (bool, int) {
	lower := strings.ToLower(status)
	idx := strings.Index(lower, "exited (")
	if idx == -1 {
		return false, 0
	}
	rest := status[idx+len("exited ("):]
	end := strings.IndexByte(rest, ')')
	if end == -1 {
		return false, 0
	}
	code, err := strconv.Atoi(strings.TrimSpace(rest[:end]))
	if err != nil {
		return false, 0
	}
	return true, code
}

// IsContainerFailed returns true and the exit code if the container is in an error state (dead or non-zero exit code).
func IsContainerFailed(state, status string) (bool, int) {
	if strings.ToLower(state) == "dead" {
		return true, -1
	}
	hasCode, code := ParseExitCode(status)
	if hasCode && code != 0 {
		return true, code
	}
	return false, 0
}
