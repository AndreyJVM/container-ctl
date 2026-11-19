package monitor_test

import (
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestFacadeExports(t *testing.T) {
	// Verify that key types are accessible through the monitor facade
	var (
		_ monitor.ContainerItem
		_ monitor.ContainerDetail
		_ monitor.DockerSummary
		_ monitor.DockerDiskUsage
		_ monitor.ComposeProject
		_ monitor.SwarmSummary
		_ monitor.SystemInfo
	)

	// Verify constants
	if monitor.StatusRunning != "running" {
		t.Errorf("expected StatusRunning == 'running', got %s", monitor.StatusRunning)
	}

	// Verify health status parser
	if monitor.ParseHealthStatus("healthy") != monitor.HealthStatusHealthy {
		t.Errorf("expected HealthStatusHealthy")
	}

	// Verify format bytes
	formatted := monitor.FormatBytes(1024 * 1024)
	if formatted != "1.0 MB" {
		t.Errorf("expected '1.0 MB', got %s", formatted)
	}

	// Verify system info calls safely
	sysInfo := monitor.GetSystemInfo()
	if sysInfo.Hostname == "" {
		t.Errorf("expected non-empty hostname from facade GetSystemInfo")
	}
}
