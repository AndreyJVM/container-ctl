package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestViewModel_RenderingTabsAndModals(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.6.1")

	// Docker tab and persistent status.
	m.activeTab = 1
	m.dockerInfo = monitor.DockerSummary{
		Status:        "running",
		StatusDisplay: "Running",
	}
	vDocker := m.View()
	if !strings.Contains(vDocker, "DOCKER CONTAINERS") {
		t.Errorf("expected 'DOCKER CONTAINERS' in View(), got: %s", vDocker)
	}
	if !strings.Contains(vDocker, "Docker: Running") {
		t.Errorf("expected 'Docker: Running', got: %s", vDocker)
	}

	// Images tab.
	m.activeTab = TabImages
	m.images = []monitor.DockerImage{
		{
			ID:         "img1",
			Repository: "demo-image",
			Tag:        "latest",
			InUse:      true,
		},
	}
	vImages := m.View()
	if !strings.Contains(vImages, "DOCKER IMAGES") {
		t.Errorf("expected 'DOCKER IMAGES' in View(), got: %s", vImages)
	}

	// Compose tab.
	m.activeTab = TabCompose
	m.compose = []monitor.ComposeProject{
		{
			Name:            "demo-stack",
			Status:          "running",
			StatusDisplay:   "Running",
			TotalServices:   1,
			RunningServices: 1,
		},
	}
	vCompose := m.View()
	if !strings.Contains(vCompose, "DOCKER COMPOSE PROJECTS") {
		t.Errorf("expected 'DOCKER COMPOSE PROJECTS' in View(), got: %s", vCompose)
	}
	if !strings.Contains(vCompose, "[Running]") {
		t.Errorf("expected '[Running]' status in Compose tab, got: %s", vCompose)
	}

	// Swarm tab.
	m.activeTab = TabSwarm
	m.swarm = monitor.SwarmSummary{
		IsActive:      false,
		StatusDisplay: "Inactive",
	}
	vSwarm := m.View()
	if !strings.Contains(vSwarm, "DOCKER SWARM CLUSTER") {
		t.Errorf("expected 'DOCKER SWARM CLUSTER' in View(), got: %s", vSwarm)
	}
	if !strings.Contains(vSwarm, "Status: Inactive") {
		t.Errorf("expected 'Status: Inactive', got: %s", vSwarm)
	}

	// Help modal.
	m.showHelp = true
	vHelp := m.View()
	if !strings.Contains(vHelp, "HOTKEYS:") {
		t.Errorf("expected 'HOTKEYS:' in View(), got: %s", vHelp)
	}
	m.showHelp = false

	// Compose down modal.
	m.composeDownTarget = "my-stack"
	m.activeModal = "compose_down"
	vDown := m.View()
	if !strings.Contains(vDown, "CONFIRM DOCKER COMPOSE DOWN") {
		t.Errorf("expected 'CONFIRM DOCKER COMPOSE DOWN' in View(), got: %s", vDown)
	}
	if !strings.Contains(vDown, "Confirm") {
		t.Errorf("expected 'Confirm' in View(), got: %s", vDown)
	}
}
