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
	t.Parallel()

	t.Run("tickMsg reschedules tick and fetch commands", func(t *testing.T) {
		m := newFastTestModel()
		_, cmd := m.Update(tickMsg(time.Now()))
		if cmd == nil {
			t.Fatalf("expected batch cmd on tickMsg, got nil")
		}
	})

	t.Run("WindowSizeMsg updates dimensions and resizes embedded shell", func(t *testing.T) {
		m := newFastTestModel()
		res, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		m = res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd on window resize")
		}
		if m.width != 120 || m.height != 40 {
			t.Errorf("expected 120x40, got %dx%d", m.width, m.height)
		}

		// When embedded shell is active
		fakeShell := &fakeEmbeddedShell{}
		m.embeddedShell = fakeShell
		m.shellContainerName = "box"
		res, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		m = res.(Model)
		if fakeShell.resizeWidth == 0 || fakeShell.resizeHeight == 0 {
			t.Errorf("expected embedded shell Resize to be called")
		}

		// When resize fails, displays toast and closes shell
		fakeShell.resizeErr = errors.New("pty resize failed")
		res, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 25})
		m = res.(Model)
		if !strings.Contains(m.actionMessage, "pty resize failed") {
			t.Errorf("expected toast with resize error, got %q", m.actionMessage)
		}
		if m.embeddedShell != nil {
			t.Errorf("expected embedded shell closed on resize error")
		}
	})

	t.Run("actionFinishedMsg toast and refresh behavior", func(t *testing.T) {
		m := newFastTestModel()

		// Error toast
		res, cmd := m.Update(actionFinishedMsg{err: errors.New("action failed")})
		m = res.(Model)
		if !strings.Contains(m.actionMessage, "action failed") {
			t.Errorf("expected toast with error, got: %s", m.actionMessage)
		}
		if cmd != nil {
			t.Errorf("expected nil cmd when refresh is false")
		}

		// Success toast with refresh
		res, cmd = m.Update(actionFinishedMsg{message: "container started", refresh: true})
		m = res.(Model)
		if !strings.Contains(m.actionMessage, "container started") {
			t.Errorf("expected toast 'container started', got: %s", m.actionMessage)
		}
		if cmd == nil {
			t.Errorf("expected fetchDataCmd on refresh: true")
		}
	})

	t.Run("actions.FinishedMsg toast and refresh behavior", func(t *testing.T) {
		m := newFastTestModel()

		// Error toast
		res, cmd := m.Update(actions.FinishedMsg{Err: errors.New("actions package error")})
		m = res.(Model)
		if !strings.Contains(m.actionMessage, "actions package error") {
			t.Errorf("expected toast with error, got: %s", m.actionMessage)
		}
		if cmd != nil {
			t.Errorf("expected nil cmd when Refresh is false")
		}

		// Success toast with refresh
		res, cmd = m.Update(actions.FinishedMsg{Message: "project stopped", Refresh: true})
		m = res.(Model)
		if !strings.Contains(m.actionMessage, "project stopped") {
			t.Errorf("expected toast 'project stopped', got: %s", m.actionMessage)
		}
		if cmd == nil {
			t.Errorf("expected fetchDataCmd on Refresh: true")
		}
	})

	t.Run("dataFetchedMsg updates model state", func(t *testing.T) {
		m := newFastTestModel()
		res, _ := m.Update(dataFetchedMsg{
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
	})

	t.Run("actions.DataFetchedMsg updates model state", func(t *testing.T) {
		m := newFastTestModel()
		res, _ := m.Update(actions.DataFetchedMsg{
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
	})

	t.Run("unfocused embedded shell hotkeys", func(t *testing.T) {
		m := newFastTestModel()
		m.embeddedShell = &fakeEmbeddedShell{}
		m.shellFocused = false

		// Pressing 'e' focuses embedded shell
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		m = res.(Model)
		if !m.shellFocused {
			t.Errorf("expected shellFocused=true after pressing 'e'")
		}

		// Pressing ctrl+x closes embedded shell
		m.shellFocused = false
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
		m = res.(Model)
		if m.embeddedShell != nil {
			t.Errorf("expected embeddedShell closed after ctrl+x")
		}
	})

	t.Run("modal, filter, and help key delegation", func(t *testing.T) {
		// Active modal intercepts keys
		m := newFastTestModel()
		m.activeModal = ModalComposeDown
		m.composeDownTarget = "stack-x"
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		m = res.(Model)
		if m.activeModal != ModalNone {
			t.Errorf("expected activeModal to close on 'q', got %s", m.activeModal)
		}

		// Help modal intercepts keys
		m = newFastTestModel()
		m.showHelp = true
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		m = res.(Model)
		if m.showHelp {
			t.Errorf("expected showHelp=false after '?'")
		}

		// Search and filter mode intercepts keys
		m = newFastTestModel()
		m.isFiltering = true
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		m = res.(Model)
		if m.filterQuery != "a" {
			t.Errorf("expected filterQuery 'a', got %q", m.filterQuery)
		}
	})

	t.Run("unknown message returns model unchanged", func(t *testing.T) {
		m := newFastTestModel()
		res, cmd := m.Update("unknown string msg")
		if res == nil {
			t.Errorf("expected non-nil model on unknown message")
		}
		if cmd != nil {
			t.Errorf("expected nil cmd on unknown message")
		}
	})
}

func TestUpdate_ExecFinished(t *testing.T) {
	origFetch := fetchDockerData
	fetchDockerData = func() dataFetchedMsg {
		return dataFetchedMsg{
			timestamp: "12:00:00",
		}
	}
	defer func() { fetchDockerData = origFetch }()

	m := newFastTestModel()

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
