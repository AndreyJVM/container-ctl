package tui

import (
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestFormatDockerStatus(t *testing.T) {
	m := Model{}

	// Direct status matches
	if res := m.formatDockerStatus(monitor.DockerStatusRunning); res != "Running" {
		t.Errorf("expected Running, got %s", res)
	}
	if res := m.formatDockerStatus(monitor.DockerStatusStopped); res != "Stopped" {
		t.Errorf("expected Stopped, got %s", res)
	}
	if res := m.formatDockerStatus(monitor.DockerStatusNotInstalled); res != "Not Installed" {
		t.Errorf("expected Not Installed, got %s", res)
	}

	// StatusDisplay matches when status is other
	m.dockerInfo.StatusDisplay = monitor.DockerStatusDisplayRunning
	if res := m.formatDockerStatus("other"); res != "Running" {
		t.Errorf("expected Running from display, got %s", res)
	}
	m.dockerInfo.StatusDisplay = monitor.DockerStatusDisplayStopped
	if res := m.formatDockerStatus("other"); res != "Stopped" {
		t.Errorf("expected Stopped from display, got %s", res)
	}
	m.dockerInfo.StatusDisplay = monitor.DockerStatusDisplayNotInstalled
	if res := m.formatDockerStatus("other"); res != "Not Installed" {
		t.Errorf("expected Not Installed from display, got %s", res)
	}

	// Custom display
	m.dockerInfo.StatusDisplay = "Degraded"
	if res := m.formatDockerStatus("other"); res != "Degraded" {
		t.Errorf("expected Degraded, got %s", res)
	}

	// Empty status & empty display -> Stopped
	m.dockerInfo.StatusDisplay = ""
	if res := m.formatDockerStatus(""); res != "Stopped" {
		t.Errorf("expected Stopped for empty status, got %s", res)
	}

	// Fallback to raw status
	if res := m.formatDockerStatus("starting"); res != "starting" {
		t.Errorf("expected starting, got %s", res)
	}
}

func TestFormatComposeStatus(t *testing.T) {
	m := Model{}

	// Direct statuses
	if res := m.formatComposeStatus(monitor.StatusRunning, ""); res != "Running" {
		t.Errorf("expected Running, got %s", res)
	}
	if res := m.formatComposeStatus(monitor.StatusPartial, ""); res != "Partial" {
		t.Errorf("expected Partial, got %s", res)
	}
	if res := m.formatComposeStatus(monitor.StatusStopped, ""); res != "Stopped" {
		t.Errorf("expected Stopped, got %s", res)
	}

	// StatusDisplay fallbacks
	if res := m.formatComposeStatus("unknown", monitor.StatusDisplayRunning); res != "Running" {
		t.Errorf("expected Running, got %s", res)
	}
	if res := m.formatComposeStatus("unknown", monitor.StatusDisplayPartial); res != "Partial" {
		t.Errorf("expected Partial, got %s", res)
	}
	if res := m.formatComposeStatus("unknown", monitor.StatusDisplayStopped); res != "Stopped" {
		t.Errorf("expected Stopped, got %s", res)
	}
	if res := m.formatComposeStatus("unknown", "CustomStatus"); res != "CustomStatus" {
		t.Errorf("expected CustomStatus, got %s", res)
	}
	if res := m.formatComposeStatus("", ""); res != "Stopped" {
		t.Errorf("expected Stopped, got %s", res)
	}
	if res := m.formatComposeStatus("raw-status", ""); res != "raw-status" {
		t.Errorf("expected raw-status, got %s", res)
	}
}

func TestFormatSwarmStatus(t *testing.T) {
	m := Model{}
	if res := m.formatSwarmStatus(true); res != "Active" {
		t.Errorf("expected Active, got %s", res)
	}
	if res := m.formatSwarmStatus(false); res != "Inactive" {
		t.Errorf("expected Inactive, got %s", res)
	}
}
