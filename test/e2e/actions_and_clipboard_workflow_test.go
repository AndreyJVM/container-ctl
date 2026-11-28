package e2e

import (
	"errors"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

func TestE2E_ContainerBatchAndSingleLifecycleActions(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()
	assertTab(t, m, tui.TabDocker)

	// --- 1. Single Container Start / Stop / Restart / Kill ---
	// Cursor is at 0: web-gateway (running)
	assertCursor(t, m, 0)
	m, stopCmd := sendKey(m, "s")
	if stopCmd == nil {
		t.Fatal("expected stop command dispatched for running container")
	}
	assertContains(t, m.ToastMessage(), "Stopping container web-gateway...")

	// Move cursor to index 2: redis-cache (exited)
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "down")
	assertCursor(t, m, 2)

	// Press 's' on stopped container -> starts it
	m, startCmd := sendKey(m, "s")
	if startCmd == nil {
		t.Fatal("expected start command dispatched for stopped container")
	}
	assertContains(t, m.ToastMessage(), "Starting container redis-cache...")

	// Press 'p' on stopped container -> restarts it
	m, restartCmd := sendKey(m, "p")
	if restartCmd == nil {
		t.Fatal("expected restart command dispatched")
	}
	assertContains(t, m.ToastMessage(), "Restarting redis-cache...")

	// Press 'K' on stopped container -> cannot kill stopped container
	m, killCmdStopped := sendKey(m, "K")
	if killCmdStopped != nil {
		t.Errorf("expected no kill command dispatched for stopped container")
	}
	assertContains(t, m.ToastMessage(), "Cannot kill: container redis-cache is not running")

	// Move cursor back to index 0: web-gateway (running)
	m, _ = sendKey(m, "g")
	assertCursor(t, m, 0)
	m, killCmdRunning := sendKey(m, "K")
	if killCmdRunning == nil {
		t.Fatal("expected kill command dispatched for running container")
	}
	assertContains(t, m.ToastMessage(), "Force killing web-gateway...")

	// --- 2. Multi-Selection: All Running Containers (c101 & c102) ---
	// Select c101 with space (auto-advances cursor to index 1: c102)
	m, _ = sendKey(m, "space")
	assertCursor(t, m, 1)

	// Select c102 (running) with space (auto-advances cursor to index 2)
	m, _ = sendKey(m, "space")
	assertSelectedContainersCount(t, m, 2)

	// Both are running -> 's' should batch stop
	m, batchStopCmd := sendKey(m, "s")
	if batchStopCmd == nil {
		t.Fatal("expected batch stop command dispatched")
	}
	assertContains(t, m.ToastMessage(), "Stopping 2 selected container(s)...")

	// 'p' with batch selection -> batch restart
	m, batchRestartCmd := sendKey(m, "p")
	if batchRestartCmd == nil {
		t.Fatal("expected batch restart command dispatched")
	}
	assertContains(t, m.ToastMessage(), "Restarting 2 selected container(s)...")

	// 'K' with batch selection -> batch kill
	m, batchKillCmd := sendKey(m, "K")
	if batchKillCmd == nil {
		t.Fatal("expected batch kill command dispatched")
	}
	assertContains(t, m.ToastMessage(), "Force killing 2 selected container(s)...")

	// Clear selection with Esc
	m, _ = sendKey(m, "esc")
	assertSelectedContainersCount(t, m, 0)

	// --- 3. Multi-Selection: Mixed States (c101 running + c103 exited) ---
	// Select c101 at index 0 (cursor advances to 1)
	m, _ = sendKey(m, "g")
	assertCursor(t, m, 0)
	m, _ = sendKey(m, "space")
	assertCursor(t, m, 1)

	// Move cursor down to index 2 (c103: redis-cache, exited) and select
	m, _ = sendKey(m, "down")
	assertCursor(t, m, 2)
	m, _ = sendKey(m, "space")
	assertSelectedContainersCount(t, m, 2)

	// Mixed states -> 's' should batch START
	m, batchMixedStartCmd := sendKey(m, "s")
	if batchMixedStartCmd == nil {
		t.Fatal("expected batch start command dispatched for mixed state selection")
	}
	assertContains(t, m.ToastMessage(), "Starting 2 selected container(s)...")

	// Open Batch Remove Modal with 'd'
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalRemoveContainer)
	assertContains(t, m.View(), "CONFIRM REMOVAL OF 2 CONTAINERS")

	// Confirm batch remove with 'y'
	m, batchRemoveCmd := sendKey(m, "y")
	assertModalClosed(t, m)
	if batchRemoveCmd == nil {
		t.Fatal("expected batch remove command dispatched")
	}
	assertSelectedContainersCount(t, m, 0)
	assertContains(t, m.ToastMessage(), "Removing 2 selected container(s)...")
}

func TestE2E_ClipboardCopyAcrossAllTabs(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Tab 1: Containers - copies "docker exec -it web-gateway sh"
	assertTab(t, m, tui.TabDocker)
	m, _ = sendKey(m, "y")
	assertContains(t, m.ToastMessage(), "web-gateway")
	assertContains(t, m.ToastMessage(), "exec -it")

	// Tab 2: Images - copies image repository:tag
	m, _ = sendKey(m, "2")
	assertTab(t, m, tui.TabImages)
	m, _ = sendKey(m, "y")
	assertContains(t, m.ToastMessage(), "nginx:alpine")

	// Tab 3: Compose - copies compose project name
	m, _ = sendKey(m, "3")
	assertTab(t, m, tui.TabCompose)
	m, _ = sendKey(m, "y")
	assertContains(t, m.ToastMessage(), "microservices-stack")

	// Tab 4: Networks - copies network name
	m, _ = sendKey(m, "4")
	assertTab(t, m, tui.TabNetworks)
	m, _ = sendKey(m, "y")
	assertContains(t, m.ToastMessage(), "frontend-bridge")

	// Tab 5: Volumes - copies volume name
	m, _ = sendKey(m, "5")
	assertTab(t, m, tui.TabVolumes)
	m, _ = sendKey(m, "y")
	assertContains(t, m.ToastMessage(), "pg_data")

	// Tab 6: Swarm - copies swarm service name
	m, _ = sendKey(m, "6")
	assertTab(t, m, tui.TabSwarm)
	m, _ = sendKey(m, "y")
	assertContains(t, m.ToastMessage(), "web-proxy")
}

func TestE2E_ActionFinishedMsgAndErrorFeedback(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Action finishes with an error
	errMsg := actions.FinishedMsg{
		Err:     errors.New("daemon error: resource is locked by another process"),
		Refresh: false,
	}
	updated, refreshCmd := m.Update(errMsg)
	m = updated.(tui.Model)
	if refreshCmd != nil {
		t.Errorf("expected no refresh cmd when Refresh=false")
	}
	assertContains(t, m.ToastMessage(), "resource is locked by another process")
	assertContains(t, m.View(), "resource is locked by another process")

	// 2. Action finishes successfully with message and refresh=true
	successMsg := actions.FinishedMsg{
		Message: "Container restarted successfully",
		Refresh: true,
	}
	updated, refreshCmd = m.Update(successMsg)
	m = updated.(tui.Model)
	if refreshCmd == nil {
		t.Errorf("expected refresh command when Refresh=true")
	}
	assertContains(t, m.ToastMessage(), "Container restarted successfully")
	assertContains(t, m.View(), "Container restarted successfully")
}

func TestE2E_WindowResizeDuringActiveModals(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. ModalLogs resize safety
	m.OpenModal(tui.ModalLogs)
	m.SetLogsForTesting("web-gateway", "c101", []string{
		"line 1: service initialization",
		"line 2: connected to database",
		"line 3: listening on :8080",
	})
	assertModal(t, m, tui.ModalLogs)

	// Shrink window
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 15})
	m = updated.(tui.Model)
	assertModal(t, m, tui.ModalLogs)
	shrinkView := m.View()
	if strings.TrimSpace(shrinkView) == "" {
		t.Fatal("expected non-empty view after shrink resize")
	}

	// Enlarge window
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	m = updated.(tui.Model)
	assertModal(t, m, tui.ModalLogs)
	enlargeView := m.View()
	assertContains(t, enlargeView, "LOGS")
	assertContains(t, enlargeView, "web-gateway")

	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// 2. ModalInspect resize safety
	m.OpenModal(tui.ModalInspect)
	m.SetInspectForTesting(monitor.ContainerDetail{
		ID:    "c101",
		Name:  "web-gateway",
		Image: "nginx:alpine",
	})
	assertModal(t, m, tui.ModalInspect)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 70, Height: 20})
	m = updated.(tui.Model)
	assertModal(t, m, tui.ModalInspect)
	inspectView := m.View()
	assertContains(t, inspectView, "CONTAINER INSPECTION")
	assertContains(t, inspectView, "web-gateway")

	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// 3. ModalPrune resize safety
	m, _ = sendKey(m, "c")
	assertModal(t, m, tui.ModalPrune)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 25})
	m = updated.(tui.Model)
	assertModal(t, m, tui.ModalPrune)
	pruneView := m.View()
	assertContains(t, pruneView, "CLEANUP & PRUNE DOCKER RESOURCES")

	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}
