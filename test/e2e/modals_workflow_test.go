package e2e

import (
	"fmt"
	"strings"
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
	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalPrune {
		t.Fatalf("expected ModalPrune to open upon pressing 'c'")
	}

	view := m.View()
	if !strings.Contains(view, "CLEANUP & PRUNE DOCKER RESOURCES") {
		t.Errorf("expected prune modal header in view, got:\n%s", view)
	}

	// 2. Select option 1 (Containers) -> enters confirmation step
	m, _ = sendKey(m, "1")
	confirmView := m.View()
	if !strings.Contains(confirmView, "Confirm Pruning") && !strings.Contains(confirmView, "[Y] Confirm") {
		t.Errorf("expected container prune confirmation prompt, got:\n%s", confirmView)
	}

	// 3. Cancel confirmation with 'n' -> returns to main prune menu
	m, _ = sendKey(m, "n")
	menuView := m.View()
	if !strings.Contains(menuView, "[1]") || !strings.Contains(menuView, "Prune Stopped Containers") {
		t.Errorf("expected return to prune action menu on 'n', got:\n%s", menuView)
	}

	// 4. Select option 4 (All / System prune) -> enters full prune confirmation
	m, _ = sendKey(m, "4")
	confirmAllView := m.View()
	if !strings.Contains(confirmAllView, "[Y] Confirm") {
		t.Errorf("expected confirmation prompt for system prune, got:\n%s", confirmAllView)
	}

	// 5. Cancel with Esc -> returns to menu
	m, _ = sendKey(m, "esc")
	if !m.IsModalOpen() {
		t.Fatalf("expected modal still open in menu after cancelling step with Esc")
	}

	// 6. Select option 2 (Images) and confirm with 'y' -> executes prune command and closes modal
	m, _ = sendKey(m, "2")
	m, cmd := sendKey(m, "y")
	if m.IsModalOpen() {
		t.Errorf("expected prune modal to close upon confirmation 'y'")
	}
	if cmd == nil {
		t.Errorf("expected pruneCmd to be dispatched upon confirmation")
	}

	// 7. Verify Esc closes the prune modal when on root menu
	m, _ = sendKey(m, "c")
	if !m.IsModalOpen() {
		t.Fatalf("expected prune modal open")
	}
	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected prune modal to close on Esc from root menu")
	}
}

// TestE2E_ExecModalWorkflow tests cursor cycling, custom command text entry,
// editing with backspace, cancelling custom input, and dismissing the exec modal.
func TestE2E_ExecModalWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Open Exec modal with 'e' on running container (web-gateway)
	m, _ = sendKey(m, "e")
	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalExec {
		t.Fatalf("expected ModalExec to open on running container with 'e'")
	}

	view := m.View()
	if !strings.Contains(view, "EXECUTE INTERACTIVE SHELL") {
		t.Errorf("expected exec modal header, got:\n%s", view)
	}
	if !strings.Contains(view, "web-gateway") {
		t.Errorf("expected container name in exec modal, got:\n%s", view)
	}

	// 2. Cycle menu items with 'j' and 'k'
	m, _ = sendKey(m, "j")
	m, _ = sendKey(m, "j")
	m, _ = sendKey(m, "k")

	// 3. Switch to custom command input mode with 'c'
	m, _ = sendKey(m, "c")
	customInputView := m.View()
	if !strings.Contains(customInputView, "Enter custom command:") {
		t.Errorf("expected custom command input prompt on 'c', got:\n%s", customInputView)
	}

	// 4. Type custom command and test backspace
	m = sendTypeString(m, "cat /etc/hosts_temp")
	typedView := m.View()
	if !strings.Contains(typedView, "hosts_temp") {
		t.Errorf("expected typed command in view, got:\n%s", typedView)
	}

	// Delete "_temp" with 5 backspaces
	for i := 0; i < 5; i++ {
		m, _ = sendKey(m, "backspace")
	}
	deletedView := m.View()
	if strings.Contains(deletedView, "hosts_temp") || !strings.Contains(deletedView, "/etc/hosts") {
		t.Errorf("expected '_temp' deleted, got:\n%s", deletedView)
	}

	// 5. Esc cancels custom input mode and returns to exec menu
	m, _ = sendKey(m, "esc")
	menuView := m.View()
	if strings.Contains(menuView, "Enter custom command:") {
		t.Errorf("expected custom input mode dismissed on Esc, got:\n%s", menuView)
	}
	if !strings.Contains(menuView, "EXECUTE INTERACTIVE SHELL") {
		t.Errorf("expected exec menu preserved, got:\n%s", menuView)
	}

	// 6. Esc closes the exec modal
	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected exec modal closed on Esc")
	}
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
	initialView := m.View()
	if !strings.Contains(initialView, "LOGS") && !strings.Contains(initialView, "web-gateway") {
		t.Errorf("expected logs header for web-gateway, got:\n%s", initialView)
	}

	// 2. Scrolling down and up
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "pgdown")
	m, _ = sendKey(m, "up")
	m, _ = sendKey(m, "pgup")

	// 3. Search and filter logs with '/'
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "ERROR")
	m, _ = sendKey(m, "enter") // commit filter

	filteredView := m.View()
	if !strings.Contains(filteredView, "[ERROR]") {
		t.Errorf("expected filtered logs to contain [ERROR], got:\n%s", filteredView)
	}
	if strings.Contains(filteredView, "[INFO]") {
		t.Errorf("expected [INFO] logs to be filtered out, got:\n%s", filteredView)
	}

	// 4. Esc clears filter
	m, _ = sendKey(m, "esc")
	clearedView := m.View()
	if !strings.Contains(clearedView, "[INFO]") {
		t.Errorf("expected [INFO] logs restored after Esc cleared filter, got:\n%s", clearedView)
	}

	// 5. Esc closes modal
	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected logs modal to close on Esc")
	}
}

// TestE2E_ComposeModalWorkflow tests Compose project actions:
// opening Compose Down modal with 'd', cancelling with Esc, confirming with 'y',
// and invoking start ('u') / stop ('s') actions.
func TestE2E_ComposeModalWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Switch to Compose tab (3)
	m, _ = sendKey(m, "3")
	if m.ActiveTab() != tui.TabCompose {
		t.Fatalf("expected TabCompose, got: %s", m.ActiveTab())
	}

	composeView := m.View()
	if !strings.Contains(composeView, "microservices-stack") {
		t.Fatalf("expected 'microservices-stack' in Compose view, got:\n%s", composeView)
	}

	// 1. Open Compose Down modal with 'd'
	m, _ = sendKey(m, "d")
	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalComposeDown {
		t.Fatalf("expected ModalComposeDown on 'd' key")
	}

	downModalView := m.View()
	if !strings.Contains(downModalView, "microservices-stack") {
		t.Errorf("expected project name in compose down modal, got:\n%s", downModalView)
	}

	// 2. Cancel with Esc
	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected compose down modal closed on Esc")
	}

	// 3. Open again and confirm with 'y'
	m, _ = sendKey(m, "d")
	m, downCmd := sendKey(m, "y")
	if m.IsModalOpen() {
		t.Errorf("expected modal closed on 'y'")
	}
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
