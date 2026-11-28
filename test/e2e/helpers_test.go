package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

// sendKey sends a key string to the model via Update.
func sendKey(m tui.Model, key string) (tui.Model, tea.Cmd) {
	var keyMsg tea.KeyMsg
	switch key {
	case "up":
		keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		keyMsg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		keyMsg = tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		keyMsg = tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		keyMsg = tea.KeyMsg{Type: tea.KeyShiftTab}
	case "backspace":
		keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
	case "space", " ":
		keyMsg = tea.KeyMsg{Type: tea.KeySpace}
	case "pgup":
		keyMsg = tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		keyMsg = tea.KeyMsg{Type: tea.KeyPgDown}
	case "ctrl+]", "ctrl+close_bracket":
		keyMsg = tea.KeyMsg{Type: tea.KeyCtrlCloseBracket}
	case "ctrl+x":
		keyMsg = tea.KeyMsg{Type: tea.KeyCtrlX}
	case "ctrl+c":
		keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+d":
		keyMsg = tea.KeyMsg{Type: tea.KeyCtrlD}
	default:
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	updated, cmd := m.Update(keyMsg)
	return updated.(tui.Model), cmd
}

// sendKeySequence sends multiple keys sequentially to the model via Update.
func sendKeySequence(m tui.Model, keys ...string) (tui.Model, tea.Cmd) {
	var lastCmd tea.Cmd
	for _, k := range keys {
		m, lastCmd = sendKey(m, k)
	}
	return m, lastCmd
}

// sendTypeString types characters one by one into the model.
func sendTypeString(m tui.Model, text string) tui.Model {
	for _, r := range text {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(tui.Model)
	}
	return m
}

// sendFilter enters filter mode with '/' and types the query string.
func sendFilter(m tui.Model, query string) tui.Model {
	m, _ = sendKey(m, "/")
	return sendTypeString(m, query)
}

// sendFilterAndConfirm enters filter mode, types the query, and commits with Enter.
func sendFilterAndConfirm(m tui.Model, query string) tui.Model {
	m = sendFilter(m, query)
	m, _ = sendKey(m, "enter")
	return m
}

// createComprehensiveTestModel builds an initialized model with realistic multi-container and multi-subsystem data.
func createComprehensiveTestModel() tui.Model {
	m := tui.NewModelWithOptions(
		2*time.Second,
		int(tui.TabDocker),
		"v0.22.6",
		tui.WithShowSwarm(true),
	)

	data := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Status:            "running",
			StatusDisplay:     "Running",
			RunningContainers: 2,
			StoppedContainers: 1,
			ImagesCount:       3,
			Version:           "27.4.0",
			Containers: []monitor.ContainerItem{
				testutil.AContainer(
					testutil.WithID("c101"),
					testutil.WithName("web-gateway"),
					testutil.WithImage("nginx:alpine"),
					testutil.WithState("running", "Up 4 hours"),
					testutil.WithMetrics(12.5, 48.0, 512.0),
				),
				testutil.AContainer(
					testutil.WithID("c102"),
					testutil.WithName("api-backend"),
					testutil.WithImage("golang:1.24"),
					testutil.WithState("running", "Up 2 hours"),
					testutil.WithMetrics(25.0, 72.0, 1024.0),
				),
				testutil.AContainer(
					testutil.WithID("c103"),
					testutil.WithName("redis-cache"),
					testutil.WithImage("redis:7-alpine"),
					testutil.WithState("exited", "Exited (0) 30m ago"),
					testutil.WithMetrics(0.0, 0.0, 0.0),
				),
			},
		},
		Images: []monitor.DockerImage{
			testutil.AnImage(testutil.WithImageID("img1", "sha256:img1"), testutil.WithRepoTag("nginx", "alpine")),
			testutil.AnImage(testutil.WithImageID("img2", "sha256:img2"), testutil.WithRepoTag("golang", "1.24")),
			testutil.AnImage(testutil.WithImageID("img3", "sha256:img3"), testutil.WithRepoTag("redis", "7-alpine")),
		},
		Compose: []monitor.ComposeProject{
			{
				Name:            "microservices-stack",
				Status:          "running",
				StatusDisplay:   "Running",
				TotalServices:   2,
				RunningServices: 2,
				Services: []monitor.ComposeService{
					{Name: "gateway", ContainerName: "web-gateway", State: "running", Status: "Up"},
					{Name: "api", ContainerName: "api-backend", State: "running", Status: "Up"},
				},
			},
		},
		Networks: []monitor.DockerNetwork{
			{ID: "net1", Name: "frontend-bridge", Driver: "bridge", Containers: 2},
			{ID: "net2", Name: "backend-internal", Driver: "bridge", Containers: 1},
		},
		Volumes: []monitor.DockerVolume{
			{Name: "pg_data", Driver: "local", RefCount: 1},
			{Name: "redis_cache", Driver: "local", RefCount: 0},
		},
		Swarm: monitor.SwarmSummary{
			IsActive:      true,
			StatusDisplay: "Active",
			IsManager:     true,
			NodesCount:    3,
			ManagersCount: 1,
			Services: []monitor.SwarmService{
				{
					ID:              "srv-web",
					Name:            "web-proxy",
					Image:           "nginx:alpine",
					Mode:            "replicated",
					Replicas:        "2/2",
					RunningReplicas: 2,
					TargetReplicas:  2,
				},
			},
		},
		Timestamp: "12:00:00",
	}

	updated, _ := m.Update(data)
	return updated.(tui.Model)
}

// assertContains fails if actual does not contain expected substring.
func assertContains(t *testing.T, actual, expected string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Errorf("expected string to contain %q, but got:\n%s", expected, actual)
	}
}

// assertNotContains fails if actual contains unexpected substring.
func assertNotContains(t *testing.T, actual, unexpected string) {
	t.Helper()
	if strings.Contains(actual, unexpected) {
		t.Errorf("expected string NOT to contain %q, but got:\n%s", unexpected, actual)
	}
}

// assertModal fails if model's active modal does not match expected.
func assertModal(t *testing.T, m tui.Model, expected tui.ModalType) {
	t.Helper()
	if !m.IsModalOpen() {
		t.Fatalf("expected modal %q to be open, but no modal is open", expected)
	}
	if m.ActiveModal() != expected {
		t.Fatalf("expected active modal %q, got %q", expected, m.ActiveModal())
	}
}

// assertModalClosed fails if any modal is still open.
func assertModalClosed(t *testing.T, m tui.Model) {
	t.Helper()
	if m.IsModalOpen() {
		t.Fatalf("expected no modal to be open, but modal %q is active", m.ActiveModal())
	}
}

// assertTab fails if model's active tab does not match expected.
func assertTab(t *testing.T, m tui.Model, expected tui.TabID) {
	t.Helper()
	if m.ActiveTab() != expected {
		t.Fatalf("expected active tab %q, got %q", expected, m.ActiveTab())
	}
}

// assertCursor fails if model's active cursor index does not match expected.
func assertCursor(t *testing.T, m tui.Model, expected int) {
	t.Helper()
	if m.ActiveCursor() != expected {
		t.Fatalf("expected active cursor %d, got %d", expected, m.ActiveCursor())
	}
}

// assertHelpOpen fails if the Help overlay is not open.
func assertHelpOpen(t *testing.T, m tui.Model) {
	t.Helper()
	if !m.IsHelpOpen() {
		t.Fatalf("expected Help overlay to be open")
	}
}

// assertHelpClosed fails if the Help overlay is open.
func assertHelpClosed(t *testing.T, m tui.Model) {
	t.Helper()
	if m.IsHelpOpen() {
		t.Fatalf("expected Help overlay to be closed")
	}
}

// assertSelectedContainersCount fails if selected containers count does not match expected.
func assertSelectedContainersCount(t *testing.T, m tui.Model, expected int) {
	t.Helper()
	if m.SelectedContainersCount() != expected {
		t.Fatalf("expected %d selected containers, got %d", expected, m.SelectedContainersCount())
	}
}

// assertSelectedImagesCount fails if selected images count does not match expected.
func assertSelectedImagesCount(t *testing.T, m tui.Model, expected int) {
	t.Helper()
	if m.SelectedImagesCount() != expected {
		t.Fatalf("expected %d selected images, got %d", expected, m.SelectedImagesCount())
	}
}

// assertSelectedNetworksCount fails if selected networks count does not match expected.
func assertSelectedNetworksCount(t *testing.T, m tui.Model, expected int) {
	t.Helper()
	if m.SelectedNetworksCount() != expected {
		t.Fatalf("expected %d selected networks, got %d", expected, m.SelectedNetworksCount())
	}
}

// assertSelectedVolumesCount fails if selected volumes count does not match expected.
func assertSelectedVolumesCount(t *testing.T, m tui.Model, expected int) {
	t.Helper()
	if m.SelectedVolumesCount() != expected {
		t.Fatalf("expected %d selected volumes, got %d", expected, m.SelectedVolumesCount())
	}
}
