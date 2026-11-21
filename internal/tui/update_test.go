package tui

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdate_MessageTypes(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v1.0.0")

	// 1. tickMsg produces a batch command
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Errorf("expected batch cmd on tickMsg, got nil")
	}

	// 2. WindowSizeMsg updates width and height
	res, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = res.(Model)
	if m.width != 120 || m.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", m.width, m.height)
	}

	// 3. actionFinishedMsg with error
	res, _ = m.Update(actionFinishedMsg{err: errors.New("action failed")})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "action failed") {
		t.Errorf("expected toast with error, got: %s", m.actionMessage)
	}

	// actionFinishedMsg with message and refresh
	res, cmd = m.Update(actionFinishedMsg{message: "container started", refresh: true})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "container started") {
		t.Errorf("expected toast 'container started', got: %s", m.actionMessage)
	}
	if cmd == nil {
		t.Errorf("expected fetchDataCmd on refresh: true")
	}

	// 4. actions.FinishedMsg with Err
	res, _ = m.Update(actions.FinishedMsg{Err: errors.New("actions package error")})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "actions package error") {
		t.Errorf("expected toast with error, got: %s", m.actionMessage)
	}

	// actions.FinishedMsg with Message and Refresh
	res, cmd = m.Update(actions.FinishedMsg{Message: "project stopped", Refresh: true})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "project stopped") {
		t.Errorf("expected toast 'project stopped', got: %s", m.actionMessage)
	}
	if cmd == nil {
		t.Errorf("expected fetchDataCmd on Refresh: true")
	}

	// 5. dataFetchedMsg applies fresh container and system info
	res, _ = m.Update(dataFetchedMsg{
		dockerInfo: monitor.DockerSummary{
			Status: "running",
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c1"), testutil.WithName("api")),
			},
		},
		images: []monitor.DockerImage{
			{ID: "img1", Repository: "api-image"},
		},
		compose: []monitor.ComposeProject{
			{Name: "app-stack"},
		},
		networks: []monitor.DockerNetwork{
			{ID: "n1", Name: "app-net"},
		},
		volumes: []monitor.DockerVolume{
			{Name: "app-vol"},
		},
		swarm: monitor.SwarmSummary{
			IsActive: true,
		},
		timestamp: "12:00:00",
	})
	m = res.(Model)
	if len(m.dockerInfo.Containers) != 1 || m.dockerInfo.Containers[0].Name != "api" {
		t.Errorf("expected 1 container 'api' after dataFetchedMsg")
	}
	if len(m.images) != 1 || len(m.compose) != 1 || len(m.networks) != 1 || len(m.volumes) != 1 {
		t.Errorf("expected images, compose, networks, volumes populated")
	}

	// 6. actions.DataFetchedMsg
	res, _ = m.Update(actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Status: "running",
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c2"), testutil.WithName("worker")),
			},
		},
		Images: []monitor.DockerImage{
			{ID: "img2", Repository: "worker-image"},
		},
		Timestamp: "12:01:00",
	})
	m = res.(Model)
	if len(m.dockerInfo.Containers) != 1 || m.dockerInfo.Containers[0].Name != "worker" {
		t.Errorf("expected 1 container 'worker' after actions.DataFetchedMsg")
	}

	// 7. Unknown msg returns model unchanged
	res, _ = m.Update("unknown string msg")
	if res == nil {
		t.Errorf("expected non-nil model on unknown message")
	}
}

func TestUpdate_ExecFinished(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v1.0.0")

	// Nil error
	res, _ := m.Update(execFinishedMsg{containerName: "box", err: nil})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Shell session in box ended") {
		t.Errorf("expected success toast, got: %s", m.actionMessage)
	}

	// Generic error
	res, _ = m.Update(execFinishedMsg{containerName: "box", err: errors.New("boom")})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Shell error in box: boom") {
		t.Errorf("expected error toast, got: %s", m.actionMessage)
	}

	// ExitError with code 130 (SIGINT)
	exit130 := exec.Command("bash", "-c", "exit 130").Run()
	if exit130 != nil {
		res, _ = m.Update(execFinishedMsg{containerName: "box", err: exit130})
		m = res.(Model)
		if !strings.Contains(m.actionMessage, "Shell session in box ended") {
			t.Errorf("expected clean exit toast for code 130, got: %s", m.actionMessage)
		}
	}

	// ExitError with code 1
	exit1 := exec.Command("bash", "-c", "exit 1").Run()
	if exit1 != nil {
		res, _ = m.Update(execFinishedMsg{containerName: "box", err: exit1})
		m = res.(Model)
		if !strings.Contains(m.actionMessage, "ended (code 1)") {
			t.Errorf("expected code 1 toast, got: %s", m.actionMessage)
		}
	}
}
