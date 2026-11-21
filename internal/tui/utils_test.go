package tui

import (
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestFormatDockerStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		status        string
		statusDisplay string
		expected      string
	}{
		// Direct status matches
		{
			name:     "monitor.DockerStatusRunning constant",
			status:   monitor.DockerStatusRunning,
			expected: "Running",
		},
		{
			name:     "monitor.DockerStatusStopped constant",
			status:   monitor.DockerStatusStopped,
			expected: "Stopped",
		},
		{
			name:     "monitor.DockerStatusNotInstalled constant",
			status:   monitor.DockerStatusNotInstalled,
			expected: "Not Installed",
		},

		// StatusDisplay matches when status is unmapped
		{
			name:          "StatusDisplay constant running",
			status:        "other",
			statusDisplay: monitor.DockerStatusDisplayRunning,
			expected:      "Running",
		},
		{
			name:          "StatusDisplay lowercase running",
			status:        "other",
			statusDisplay: "running",
			expected:      "Running",
		},
		{
			name:          "StatusDisplay constant stopped",
			status:        "other",
			statusDisplay: monitor.DockerStatusDisplayStopped,
			expected:      "Stopped",
		},
		{
			name:          "StatusDisplay lowercase stopped",
			status:        "other",
			statusDisplay: "stopped",
			expected:      "Stopped",
		},
		{
			name:          "StatusDisplay constant not installed",
			status:        "other",
			statusDisplay: monitor.DockerStatusDisplayNotInstalled,
			expected:      "Not Installed",
		},
		{
			name:          "StatusDisplay lowercase not_installed",
			status:        "other",
			statusDisplay: "not_installed",
			expected:      "Not Installed",
		},
		{
			name:          "StatusDisplay custom string",
			status:        "other",
			statusDisplay: "Degraded",
			expected:      "Degraded",
		},

		// Empty status & empty display defaults to Stopped
		{
			name:          "empty status and empty display",
			status:        "",
			statusDisplay: "",
			expected:      "Stopped",
		},

		// Empty status with valid display
		{
			name:          "empty status with display running",
			status:        "",
			statusDisplay: "running",
			expected:      "Running",
		},

		// Raw status fallback when unmapped and display is empty
		{
			name:          "raw status fallback",
			status:        "starting",
			statusDisplay: "",
			expected:      "starting",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := Model{
				dockerInfo: monitor.DockerSummary{
					StatusDisplay: tt.statusDisplay,
				},
			}
			if res := m.formatDockerStatus(tt.status); res != tt.expected {
				t.Errorf("status=%q display=%q: expected %q, got %q", tt.status, tt.statusDisplay, tt.expected, res)
			}
		})
	}
}

func TestFormatComposeStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		status        string
		statusDisplay string
		expected      string
	}{
		// Direct statuses
		{
			name:     "monitor.StatusRunning constant",
			status:   monitor.StatusRunning,
			expected: "Running",
		},
		{
			name:     "monitor.StatusPartial constant",
			status:   monitor.StatusPartial,
			expected: "Partial",
		},
		{
			name:     "monitor.StatusStopped constant",
			status:   monitor.StatusStopped,
			expected: "Stopped",
		},

		// StatusDisplay fallbacks
		{
			name:          "StatusDisplay constant running",
			status:        "unknown",
			statusDisplay: monitor.StatusDisplayRunning,
			expected:      "Running",
		},
		{
			name:          "StatusDisplay lowercase running",
			status:        "unknown",
			statusDisplay: "running",
			expected:      "Running",
		},
		{
			name:          "StatusDisplay constant partial",
			status:        "unknown",
			statusDisplay: monitor.StatusDisplayPartial,
			expected:      "Partial",
		},
		{
			name:          "StatusDisplay lowercase partial",
			status:        "unknown",
			statusDisplay: "partial",
			expected:      "Partial",
		},
		{
			name:          "StatusDisplay constant stopped",
			status:        "unknown",
			statusDisplay: monitor.StatusDisplayStopped,
			expected:      "Stopped",
		},
		{
			name:          "StatusDisplay lowercase stopped",
			status:        "unknown",
			statusDisplay: "stopped",
			expected:      "Stopped",
		},
		{
			name:          "custom display string",
			status:        "unknown",
			statusDisplay: "CustomStatus",
			expected:      "CustomStatus",
		},
		{
			name:          "empty status and empty display",
			status:        "",
			statusDisplay: "",
			expected:      "Stopped",
		},
		{
			name:          "raw status fallback",
			status:        "raw-status",
			statusDisplay: "",
			expected:      "raw-status",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := Model{}
			if res := m.formatComposeStatus(tt.status, tt.statusDisplay); res != tt.expected {
				t.Errorf("status=%q display=%q: expected %q, got %q", tt.status, tt.statusDisplay, tt.expected, res)
			}
		})
	}
}

func TestFormatSwarmStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		isActive bool
		expected string
	}{
		{name: "active cluster", isActive: true, expected: "Active"},
		{name: "inactive cluster", isActive: false, expected: "Inactive"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := Model{}
			if res := m.formatSwarmStatus(tt.isActive); res != tt.expected {
				t.Errorf("isActive=%v: expected %q, got %q", tt.isActive, tt.expected, res)
			}
		})
	}
}
