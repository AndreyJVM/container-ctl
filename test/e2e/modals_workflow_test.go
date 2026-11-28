package e2e

import (
	"fmt"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/tui"
)

// TestE2E_PruneModalWorkflow tests full navigation, selection, confirmation,
// cancellation, and execution of Docker system cleanup & prune modal.
func TestE2E_PruneModalWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Open Prune Modal with 'c'
	m, _ = sendKey(m, "c")
	assertModal(t, m, tui.ModalPrune)
	assertContains(t, m.View(), "CLEANUP & PRUNE DOCKER RESOURCES")

	// 2. Select option 1 (Containers) -> enters confirmation step
	m, _ = sendKey(m, "1")
	assertContainsAny(t, m.View(), "Confirm Pruning", "[Y] Confirm")

	// 3. Cancel confirmation with 'n' -> returns to main prune menu
	m, _ = sendKey(m, "n")
	menuView := m.View()
	assertContains(t, menuView, "[1]")
	assertContains(t, menuView, "Prune Stopped Containers")

	// 4. Select option 4 (All / System prune) -> enters full prune confirmation
	m, _ = sendKey(m, "4")
	assertContains(t, m.View(), "[Y] Confirm")

	// 5. Cancel with Esc -> returns to menu
	m, _ = sendKey(m, "esc")
	assertModal(t, m, tui.ModalPrune)

	// 6. Select option 2 (Images) and confirm with 'y' -> executes prune command and closes modal
	m, _ = sendKey(m, "2")
	m, cmd := sendKey(m, "y")
	assertModalClosed(t, m)
	if cmd == nil {
		t.Errorf("expected pruneCmd to be dispatched upon confirmation")
	}

	// 7. Verify Esc closes the prune modal when on root menu
	m, _ = sendKey(m, "c")
	assertModal(t, m, tui.ModalPrune)
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}

// TestE2E_ExecModalWorkflow tests cursor cycling, custom command text entry,
// editing with backspace, cancelling custom input, and dismissing the exec modal.
func TestE2E_ExecModalWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Open Exec modal with 'e' on running container (web-gateway)
	m, _ = sendKey(m, "e")
	assertModal(t, m, tui.ModalExec)

	view := m.View()
	assertContains(t, view, "EXECUTE INTERACTIVE SHELL")
	assertContains(t, view, "web-gateway")

	// 2. Cycle menu items with 'j' and 'k'
	m, _ = sendKey(m, "j")
	m, _ = sendKey(m, "j")
	m, _ = sendKey(m, "k")

	// 3. Switch to custom command input mode with 'c'
	m, _ = sendKey(m, "c")
	assertContains(t, m.View(), "Enter custom command:")

	// 4. Type custom command and test backspace
	m = sendTypeString(m, "cat /etc/hosts_temp")
	assertContains(t, m.View(), "hosts_temp")

	// Delete "_temp" with 5 backspaces
	for i := 0; i < 5; i++ {
		m, _ = sendKey(m, "backspace")
	}
	deletedView := m.View()
	assertNotContains(t, deletedView, "hosts_temp")
	assertContains(t, deletedView, "/etc/hosts")

	// 5. Esc cancels custom input mode and returns to exec menu
	m, _ = sendKey(m, "esc")
	menuView := m.View()
	assertNotContains(t, menuView, "Enter custom command:")
	assertContains(t, menuView, "EXECUTE INTERACTIVE SHELL")

	// 6. Esc closes the exec modal
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}

// TestE2E_LogsModalInteractiveWorkflow tests log viewing, scrolling,
// in-modal searching (/), filtering results, and clean dismissal.
func TestE2E_LogsModalInteractiveWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Prepare 60 realistic log lines
	var logLines []string
	for i := 1; i <= 60; i++ {
		if i%10 == 0 {
			logLines = append(logLines, fmt.Sprintf("2026-11-28T02:00:%02d.000Z [ERROR] Failed database query attempt #%d", i, i))
		} else if i%5 == 0 {
			logLines = append(logLines, fmt.Sprintf("2026-11-28T02:00:%02d.000Z [WARN] High memory consumption detected #%d", i, i))
		} else {
			logLines = append(logLines, fmt.Sprintf("2026-11-28T02:00:%02d.000Z [INFO] HTTP GET /api/v1/health HTTP/1.1 200 OK #%d", i, i))
		}
	}

	m.OpenModal(tui.ModalLogs)
	m.SetLogsForTesting("web-gateway", "c101", logLines)

	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalLogs {
		t.Fatalf("expected ModalLogs to be active")
	}

	// 1. Initial view renders logs
	assertContainsAny(t, m.View(), "LOGS", "web-gateway")

	// 2. Scrolling down and up
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "pgdown")
	m, _ = sendKey(m, "up")
	m, _ = sendKey(m, "pgup")

	// 3. Search and filter logs with '/'
	m = sendFilterAndConfirm(m, "ERROR")

	filteredView := m.View()
	assertContains(t, filteredView, "[ERROR]")
	assertNotContains(t, filteredView, "[INFO]")

	// 4. Esc clears filter
	m, _ = sendKey(m, "esc")
	assertContains(t, m.View(), "[INFO]")

	// 5. Esc closes modal
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}

// TestE2E_ComposeModalWorkflow tests Compose project actions:
// opening Compose Down modal with 'd', cancelling with Esc, confirming with 'y',
// and invoking start ('u') / stop ('s') actions.
func TestE2E_ComposeModalWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Switch to Compose tab (3)
	m, _ = sendKey(m, "3")
	assertTab(t, m, tui.TabCompose)
	assertContains(t, m.View(), "microservices-stack")

	// 1. Open Compose Down modal with 'd'
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalComposeDown)
	assertContains(t, m.View(), "microservices-stack")

	// 2. Cancel with Esc
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// 3. Open again and confirm with 'y'
	m, _ = sendKey(m, "d")
	m, downCmd := sendKey(m, "y")
	assertModalClosed(t, m)
	if downCmd == nil {
		t.Errorf("expected downComposeCmd dispatched on confirm")
	}

	// 4. Test Stop project with 's'
	m, stopCmd := sendKey(m, "s")
	if stopCmd == nil {
		t.Errorf("expected stopComposeCmd dispatched on 's'")
	}

	// 5. Test Start project with 'u'
	m, startCmd := sendKey(m, "u")
	if startCmd == nil {
		t.Errorf("expected startComposeCmd dispatched on 'u'")
	}

	// 6. Test Copy project name with 'y'
	m, _ = sendKey(m, "y")
}
