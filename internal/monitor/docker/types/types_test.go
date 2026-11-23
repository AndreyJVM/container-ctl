package types

import (
	"testing"
	"time"
)

func TestDockerConstants(t *testing.T) {
	if DockerStatusRunning != "running" {
		t.Errorf("unexpected DockerStatusRunning: %s", DockerStatusRunning)
	}
	if DockerStatusStopped != "stopped" {
		t.Errorf("unexpected DockerStatusStopped: %s", DockerStatusStopped)
	}
	if DockerStatusNotInstalled != "not_installed" {
		t.Errorf("unexpected DockerStatusNotInstalled: %s", DockerStatusNotInstalled)
	}

	if DockerStatusDisplayRunning != "Running" {
		t.Errorf("unexpected DockerStatusDisplayRunning: %s", DockerStatusDisplayRunning)
	}
	if DockerStatusDisplayStopped != "Stopped" {
		t.Errorf("unexpected DockerStatusDisplayStopped: %s", DockerStatusDisplayStopped)
	}
	if DockerStatusDisplayNotInstalled != "Not installed" {
		t.Errorf("unexpected DockerStatusDisplayNotInstalled: %s", DockerStatusDisplayNotInstalled)
	}

	if HealthStatusHealthy != "healthy" || HealthStatusUnhealthy != "unhealthy" || HealthStatusStarting != "starting" {
		t.Errorf("unexpected health status constants")
	}

	if ComposeProjectLabel != "com.docker.compose.project" {
		t.Errorf("unexpected ComposeProjectLabel: %s", ComposeProjectLabel)
	}
	if ComposeServiceLabel != "com.docker.compose.service" {
		t.Errorf("unexpected ComposeServiceLabel: %s", ComposeServiceLabel)
	}

	if MaxConcurrentStatsFetch != 10 {
		t.Errorf("unexpected MaxConcurrentStatsFetch: %d", MaxConcurrentStatsFetch)
	}
	if DefaultDockerTimeout != 5*time.Second {
		t.Errorf("unexpected DefaultDockerTimeout: %v", DefaultDockerTimeout)
	}
	if ContainerTimeFormat != "2006-01-02 15:04:05" {
		t.Errorf("unexpected ContainerTimeFormat: %s", ContainerTimeFormat)
	}
}

func TestDockerTypes_Instantiation(t *testing.T) {
	item := ContainerItem{
		ID:             "cid-123",
		Name:           "test-container",
		Image:          "alpine:latest",
		State:          "running",
		CPUPerc:        12.5,
		MemUsageMB:     64.0,
		MemLimitMB:     512.0,
		MemPerc:        12.5,
		NetRxBytes:     1024,
		NetTxBytes:     2048,
		ComposeProject: "proj",
		ComposeService: "web",
		Labels:         map[string]string{"foo": "bar"},
	}
	if item.ID != "cid-123" || item.Labels["foo"] != "bar" {
		t.Errorf("unexpected ContainerItem: %+v", item)
	}

	detail := ContainerDetail{
		ID:         "cid-123",
		Name:       "test-container",
		Env:        []string{"KEY=VAL"},
		Mounts:     []string{"/data"},
		Gateway:    "172.17.0.1",
		MACAddress: "02:42:ac:11:00:02",
	}
	if detail.Name != "test-container" || len(detail.Env) != 1 {
		t.Errorf("unexpected ContainerDetail: %+v", detail)
	}

	summary := DockerSummary{
		Status:            DockerStatusRunning,
		StatusDisplay:     DockerStatusDisplayRunning,
		RunningContainers: 1,
		TotalContainers:   1,
		Containers:        []ContainerItem{item},
	}
	if summary.Status != DockerStatusRunning || len(summary.Containers) != 1 {
		t.Errorf("unexpected DockerSummary: %+v", summary)
	}
}
