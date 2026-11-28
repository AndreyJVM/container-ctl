package e2e

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

// TestE2E_EmbeddedShellFullLifecycle verifies opening an in-app embedded shell session,
// keyboard focus toggling, input forwarding, clean closure via Ctrl+X, process exit handling,
// and PTY launch failure handling.
func TestE2E_EmbeddedShellFullLifecycle(t *testing.T) {
	fake := &e2eFakeShellSession{}
	cleanupShell := tui.SetStartEmbeddedShellForTesting(func(cmd *exec.Cmd, width, height int) (tui.EmbeddedShellSession, error) {
		return fake, nil
	})
	defer cleanupShell()

	m := sendWindowSize(createComprehensiveTestModel(), 140, 40)

	// 1. Cursor starts on running container "web-gateway" (c101). Press 'e' to open Exec modal.
	m, _ = sendKey(m, "e")
	assertModal(t, m, tui.ModalExec)
	assertContains(t, m.View(), "EXECUTE INTERACTIVE SHELL")

	// 2. Index 0 is "Embedded Shell (in-app)". Press Enter to launch.
	m, cmd := sendKey(m, "enter")
	assertModalClosed(t, m)
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd upon launching embedded shell")
	}

	if !m.IsShellActive() {
		t.Fatal("expected embedded shell to be active")
	}
	if !m.IsShellFocused() {
		t.Fatal("expected embedded shell to be focused upon launch")
	}
	if m.ShellContainerName() != "web-gateway" {
		t.Fatalf("expected shell container name 'web-gateway', got %q", m.ShellContainerName())
	}

	view := m.View()
	assertContains(t, view, "TERMINAL: web-gateway")
	assertContains(t, view, "root@web-gateway:/app#")
	assertContains(t, view, "Ctrl+]")
	assertContains(t, view, "Return to container-ctl")

	// 3. Typing keystrokes while focused should forward input directly to shell
	m = sendTypeString(m, "whoami")
	if len(fake.writes) != 6 {
		t.Fatalf("expected 6 writes forwarded to shell, got %d", len(fake.writes))
	}

	// 4. Ctrl+] unfocuses shell and returns control to container-ctl dashboard
	m, _ = sendKey(m, "ctrl+]")
	if m.IsShellFocused() {
		t.Fatal("expected shell to be unfocused after Ctrl+]")
	}
	if !m.IsShellActive() {
		t.Fatal("expected shell to remain active while unfocused")
	}
	unfocusedView := m.View()
	assertContains(t, unfocusedView, "Focus terminal")

	// 5. While unfocused, typing dashboard hotkeys (e.g., '2' to switch tab) does not send to shell
	writesBefore := len(fake.writes)
	m, _ = sendKey(m, "2")
	assertTab(t, m, tui.TabImages)
	if len(fake.writes) != writesBefore {
		t.Errorf("expected no additional writes to shell while unfocused, got %d", len(fake.writes)-writesBefore)
	}

	// 6. Pressing 'e' while unfocused re-focuses the existing shell
	m, _ = sendKey(m, "e")
	if !m.IsShellFocused() {
		t.Fatal("expected 'e' to re-focus active shell")
	}

	// 7. Ctrl+X closes shell cleanly
	m, closeCmd := sendKey(m, "ctrl+x")
	if m.IsShellActive() {
		t.Fatal("expected shell to be inactive after Ctrl+X")
	}
	if m.IsShellFocused() {
		t.Fatal("expected shell focus to be cleared after Ctrl+X")
	}
	if fake.closeCount != 1 {
		t.Fatalf("expected shell Close() called exactly once, got %d", fake.closeCount)
	}
	if closeCmd == nil {
		t.Error("expected fetchDataCmd dispatched on shell close")
	}
	assertToastContains(t, m, "Shell session in web-gateway closed")

	// 8. Test external process exit message handling (e.g. exit code 0)
	// Launch shell again
	m, _ = sendKey(m, "1") // return to containers tab
	m, _ = sendKey(m, "e")
	m, _ = sendKey(m, "enter")
	if !m.IsShellActive() {
		t.Fatal("expected shell active after second launch")
	}

	// Simulate exit message from session
	updated, refreshCmd := m.Update(tui.NewEmbeddedShellExitedMsg(fake, nil))
	m = updated.(tui.Model)
	if m.IsShellActive() {
		t.Fatal("expected shell closed after exit message")
	}
	assertToastContains(t, m, "Shell session in web-gateway ended")
	if refreshCmd == nil {
		t.Error("expected fetchDataCmd dispatched after shell exited")
	}

	// 9. Test shell startup error (e.g. PTY error)
	cleanupFail := tui.SetStartEmbeddedShellForTesting(func(_ *exec.Cmd, _, _ int) (tui.EmbeddedShellSession, error) {
		return nil, errors.New("PTY fork failed")
	})
	defer cleanupFail()

	m, _ = sendKey(m, "e")
	m, failCmd := sendKey(m, "enter")
	if failCmd != nil {
		t.Error("expected nil cmd when shell fails to start")
	}
	if m.IsShellActive() {
		t.Fatal("expected shell not to be active after startup failure")
	}
	assertToastContains(t, m, "Shell error in web-gateway: PTY fork failed")
}

// TestE2E_ComposeServiceLogsWorkflow verifies opening aggregate Compose project logs
// with 'l' on TabCompose, viewing multi-service interleaved logs, searching within logs,
// clearing search filter, and closing logs viewer.
func TestE2E_ComposeServiceLogsWorkflow(t *testing.T) {
	// Mock compose runner returning multi-service log entries
	cleanupRunner := monitor.SetComposeCommandRunnerForTesting(func(ctx context.Context, args ...string) ([]byte, error) {
		mockLogs := strings.Join([]string{
			"gateway | 2026-11-28 10:00:00 [INFO] Gateway started listening on :80",
			"api     | 2026-11-28 10:00:01 [INFO] API database connected successfully",
			"gateway | 2026-11-28 10:00:05 [WARN] Upstream backend latency exceeded 100ms",
			"api     | 2026-11-28 10:00:10 [ERROR] Connection timeout to cache cluster",
			"gateway | 2026-11-28 10:00:15 [INFO] Health check probe HTTP/1.1 200 OK",
		}, "\n")
		return []byte(mockLogs), nil
	})
	defer cleanupRunner()

	m := sendWindowSize(createComprehensiveTestModel(), 140, 40)

	// 1. Switch to TabCompose (3)
	m, _ = sendKey(m, "3")
	assertTab(t, m, tui.TabCompose)
	assertContains(t, m.View(), "microservices-stack")

	// 2. Press 'l' to open Compose project logs
	m, _ = sendKey(m, "l")
	assertModal(t, m, tui.ModalLogs)

	logsView := m.View()
	assertContains(t, logsView, "LOGS")
	assertContains(t, logsView, "Compose: microservices-stack")
	assertContains(t, logsView, "gateway |")
	assertContains(t, logsView, "api     |")
	assertContains(t, logsView, "Gateway started listening on :80")
	assertContains(t, logsView, "Connection timeout to cache cluster")

	// 3. Search and filter logs with '/'
	m = sendFilterAndConfirm(m, "ERROR")
	filteredView := m.View()
	assertContains(t, filteredView, "[ERROR]")
	assertContains(t, filteredView, "Connection timeout")
	assertNotContains(t, filteredView, "Gateway started listening")

	// 4. Esc clears filter
	m, _ = sendKey(m, "esc")
	clearedView := m.View()
	assertContains(t, clearedView, "Gateway started listening")
	assertContains(t, clearedView, "Connection timeout")

	// 5. Esc closes logs modal
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}

// TestE2E_SwarmReplicaScalingModalDeepWorkflow tests interactive Swarm service scaling:
// modal numeric input, digit replacement, digit shifting, bounds enforcement,
// incrementing/decrementing, ignoring invalid characters, confirming scaling action,
// and rejecting scaling on global services.
func TestE2E_SwarmReplicaScalingModalDeepWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Switch to Swarm tab (6)
	m, _ = sendKey(m, "6")
	assertTab(t, m, tui.TabSwarm)
	assertContains(t, m.View(), "web-proxy")

	// 2. Press 's' on replicated service -> opens ModalSwarmScale
	m, _ = sendKey(m, "s")
	assertModal(t, m, tui.ModalSwarmScale)

	view := m.View()
	assertContains(t, view, "SCALE DOCKER SWARM SERVICE")
	assertContains(t, view, "web-proxy")
	assertContains(t, view, "Current Replicas: 2")

	// 3. Type single digit '5': replaces initial replica count (2 -> 5)
	m, _ = sendKey(m, "5")
	assertContains(t, m.View(), "[  5 ]")

	// 4. Type second digit '0': shifts left (5 -> 50)
	m, _ = sendKey(m, "0")
	assertContains(t, m.View(), "[ 50 ]")

	// 5. Type invalid characters: letters and punctuation should be ignored
	m, _ = sendKey(m, "x")
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "@")
	assertContains(t, m.View(), "[ 50 ]")

	// 6. Backspace: integer division by 10 (50 -> 5)
	m, _ = sendKey(m, "backspace")
	assertContains(t, m.View(), "[  5 ]")

	// 7. Increment with '+' / 'k' / 'up'
	m, _ = sendKey(m, "+")
	assertContains(t, m.View(), "[  6 ]")
	m, _ = sendKey(m, "k")
	assertContains(t, m.View(), "[  7 ]")

	// 8. Decrement with '-' / 'j' / 'down'
	m, _ = sendKey(m, "-")
	assertContains(t, m.View(), "[  6 ]")
	m, _ = sendKey(m, "j")
	assertContains(t, m.View(), "[  5 ]")

	// 9. Confirm with Enter -> triggers scaling command and displays toast
	m, scaleCmd := sendKey(m, "enter")
	assertModalClosed(t, m)
	if scaleCmd == nil {
		t.Error("expected scaleSwarmServiceCmd dispatched upon Enter")
	}
	assertToastContains(t, m, "Scaling service web-proxy to 5 replicas...")

	// 10. Test global service rejection: add a global service and verify 's' is rejected
	mWithGlobal := m
	globalData := actions.DataFetchedMsg{
		Swarm: monitor.SwarmSummary{
			IsActive:      true,
			StatusDisplay: "Active",
			IsManager:     true,
			NodesCount:    3,
			ManagersCount: 1,
			Services: []monitor.SwarmService{
				{
					ID:              "srv-agent",
					Name:            "monitoring-agent",
					Image:           "prom/node-exporter:latest",
					Mode:            "global",
					Replicas:        "3/3",
					RunningReplicas: 3,
					TargetReplicas:  3,
				},
			},
		},
		Timestamp: "12:05:00",
	}
	updated, _ := mWithGlobal.Update(globalData)
	mWithGlobal = updated.(tui.Model)

	// Attempting to scale global service
	mWithGlobal, _ = sendKey(mWithGlobal, "s")
	assertModalClosed(t, mWithGlobal)
	assertToastContains(t, mWithGlobal, "Cannot scale global Swarm service")
}
