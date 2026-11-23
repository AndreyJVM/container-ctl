package tui

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// Tests from keys_modals_test.go
// =============================================================================

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestModalLogsSearch(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.activeModal = ModalLogs
	m.logsTargetName = "test-app"
	m.logsTargetID = "c123"
	m.logsLines = []string{
		"2026-11-20 INFO starting service",
		"2026-11-20 ERROR connection failed",
		"2026-11-20 INFO healthy",
	}

	// 1. Press '/' to activate log search
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.logsFiltering {
		t.Fatalf("expected logsFiltering to be true after pressing '/'")
	}

	// 2. Type "ERROR"
	for _, r := range "ERROR" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}
	if m.logsFilterQuery != "ERROR" {
		t.Errorf("expected logsFilterQuery 'ERROR', got %q", m.logsFilterQuery)
	}

	// View should display Search prompt and matching line
	view := m.View()
	if !strings.Contains(view, "Search:") || !strings.Contains(view, "connection failed") {
		t.Errorf("expected search prompt and matching line in view, got: %s", view)
	}

	// Backspace in search
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.logsFilterQuery != "ERRO" {
		t.Errorf("expected logsFilterQuery 'ERRO' after backspace, got %q", m.logsFilterQuery)
	}

	// Re-add 'R'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	m = res.(Model)

	// 3. Press Enter to lock search
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.logsFiltering {
		t.Errorf("expected logsFiltering false after Enter")
	}
	if m.logsFilterQuery != "ERROR" {
		t.Errorf("expected query preserved after Enter, got %q", m.logsFilterQuery)
	}

	// 4. Press Esc to clear filter query
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.logsFilterQuery != "" {
		t.Errorf("expected logsFilterQuery cleared after Esc, got %q", m.logsFilterQuery)
	}
	if m.activeModal != ModalLogs {
		t.Errorf("modal should remain open after clearing filter")
	}

	// 5. Test regex search: type "error|healthy"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	for _, r := range "error|healthy" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.countFilteredLogs() != 2 {
		t.Errorf("expected countFilteredLogs=2 for regex, got %d", m.countFilteredLogs())
	}
	viewRegex := m.View()
	if !strings.Contains(viewRegex, "2 matches") || !strings.Contains(viewRegex, "[regex]") {
		t.Errorf("expected 2 matches and [regex] badge in view, got: %s", viewRegex)
	}

	// Clear filter with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)

	// 6. Press Esc again to close modal
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected modal closed after second Esc, got %s", m.activeModal)
	}
}

func TestModalInspectScrollingAndCopy(t *testing.T) {
	defer clipboard.TestLock()()
	origWriter := clipboard.GetOutputWriter()
	var buf bytes.Buffer
	clipboard.SetOutputWriter(&buf)
	defer func() { clipboard.SetOutputWriter(origWriter) }()

	m := newFastTestModel()

	// Set inspect target with enough env vars to allow scrolling
	m.inspectTarget = monitor.ContainerDetail{
		ID:     "c123456789abcdef",
		Name:   "worker-pool",
		Image:  "worker:latest",
		State:  "running",
		Status: "Up 5 hours",
		Env: []string{
			"E1=1", "E2=2", "E3=3", "E4=4", "E5=5",
			"E6=6", "E7=7", "E8=8", "E9=9", "E10=10",
			"E11=11", "E12=12", "E13=13", "E14=14", "E15=15",
		},
	}
	m.activeModal = ModalInspect
	m.inspectScroll = 0

	// Test scroll down with KeyDown
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.inspectScroll != 1 {
		t.Errorf("expected inspectScroll=1 after down, got %d", m.inspectScroll)
	}

	// Test scroll down with 'j'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = res.(Model)
	if m.inspectScroll != 2 {
		t.Errorf("expected inspectScroll=2 after 'j', got %d", m.inspectScroll)
	}

	// Test scroll up with KeyUp
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.inspectScroll != 1 {
		t.Errorf("expected inspectScroll=1 after up, got %d", m.inspectScroll)
	}

	// Test scroll up with 'k'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = res.(Model)
	if m.inspectScroll != 0 {
		t.Errorf("expected inspectScroll=0 after 'k', got %d", m.inspectScroll)
	}

	// Test page down
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = res.(Model)
	if m.inspectScroll != 10 {
		t.Errorf("expected inspectScroll=10 after pgdown, got %d", m.inspectScroll)
	}

	// Test page up
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = res.(Model)
	if m.inspectScroll != 0 {
		t.Errorf("expected inspectScroll=0 after pgup, got %d", m.inspectScroll)
	}

	// Test 'y' copy ID
	buf.Reset()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "c123456789abcdef") {
		t.Errorf("expected toast with container ID on 'y', got: %q", m.actionMessage)
	}

	// Test clipboard error handling
	clipboard.SetOutputWriter(errWriter{})
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Clipboard error") {
		t.Errorf("expected clipboard error toast, got: %q", m.actionMessage)
	}
	clipboard.SetOutputWriter(&buf)

	// Verify view rendering includes pagination info
	v := m.View()
	if !strings.Contains(v, "worker-pool") || !strings.Contains(v, "[Lines") {
		t.Errorf("expected inspect modal view with pagination, got: %s", v)
	}

	// Close inspect modal with 'q'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected inspect modal closed with 'q', got %s", m.activeModal)
	}
}

func TestModalConfirmDialogs(t *testing.T) {
	t.Parallel()

	t.Run("ModalComposeDown cancel variants and confirm", func(t *testing.T) {
		cancelKeys := []string{"esc", "q", "n", "N"}
		for _, k := range cancelKeys {
			m := newFastTestModel()
			m.activeModal = ModalComposeDown
			m.composeDownTarget = "my-stack"

			var msg tea.KeyMsg
			if k == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
			res, cmd := m.handleModalInput(msg)
			m = res.(Model)
			if m.activeModal != "" || m.composeDownTarget != "" || cmd != nil {
				t.Errorf("key %q: expected ModalComposeDown cancelled cleanly", k)
			}
		}

		// Confirm with y and Y
		for _, k := range []string{"y", "Y"} {
			m := newFastTestModel()
			m.activeModal = ModalComposeDown
			m.composeDownTarget = "my-stack"
			res, cmd := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			m = res.(Model)
			if m.activeModal != "" || cmd == nil {
				t.Errorf("key %q: expected ModalComposeDown confirmed and returned cmd", k)
			}
			if !strings.Contains(m.actionMessage, "Downing Compose project my-stack") {
				t.Errorf("expected toast for downing compose project, got %q", m.actionMessage)
			}
		}
	})

	t.Run("ModalRemoveContainer cancel variants and confirm", func(t *testing.T) {
		cancelKeys := []string{"esc", "q", "n", "N"}
		for _, k := range cancelKeys {
			m := newFastTestModel()
			m.activeModal = ModalRemoveContainer
			m.removeContainerTargetID = "c1"
			m.removeContainerTargetName = "web"

			var msg tea.KeyMsg
			if k == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
			res, cmd := m.handleModalInput(msg)
			m = res.(Model)
			if m.activeModal != "" || m.removeContainerTargetID != "" || cmd != nil {
				t.Errorf("key %q: expected ModalRemoveContainer cancelled cleanly", k)
			}
		}

		// Confirm with y and Y
		for _, k := range []string{"y", "Y"} {
			m := newFastTestModel()
			m.activeModal = ModalRemoveContainer
			m.removeContainerTargetID = "c1"
			m.removeContainerTargetName = "web"
			res, cmd := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			m = res.(Model)
			if m.activeModal != "" || cmd == nil {
				t.Errorf("key %q: expected ModalRemoveContainer confirmed and returned cmd", k)
			}
			if !strings.Contains(m.actionMessage, "Removing container web") {
				t.Errorf("expected toast for removing container, got %q", m.actionMessage)
			}
		}
	})

	t.Run("ModalRemoveImage cancel variants and confirm", func(t *testing.T) {
		cancelKeys := []string{"esc", "q", "n", "N"}
		for _, k := range cancelKeys {
			m := newFastTestModel()
			m.activeModal = ModalRemoveImage
			m.removeImageTargetID = "img1"
			m.removeImageTargetName = "nginx:latest"
			m.removeImageTargetInUse = true

			var msg tea.KeyMsg
			if k == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
			res, cmd := m.handleModalInput(msg)
			m = res.(Model)
			if m.activeModal != "" || m.removeImageTargetID != "" || m.removeImageTargetInUse || cmd != nil {
				t.Errorf("key %q: expected ModalRemoveImage cancelled cleanly", k)
			}
		}

		// Confirm with y and Y
		for _, k := range []string{"y", "Y"} {
			m := newFastTestModel()
			m.activeModal = ModalRemoveImage
			m.removeImageTargetID = "img1"
			m.removeImageTargetName = "nginx:latest"
			res, cmd := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			m = res.(Model)
			if m.activeModal != "" || cmd == nil {
				t.Errorf("key %q: expected ModalRemoveImage confirmed and returned cmd", k)
			}
			if !strings.Contains(m.actionMessage, "Removing image nginx:latest") {
				t.Errorf("expected toast for removing image, got %q", m.actionMessage)
			}
		}
	})

	t.Run("ModalRemoveNetwork cancel variants and confirm", func(t *testing.T) {
		cancelKeys := []string{"esc", "q", "n", "N"}
		for _, k := range cancelKeys {
			m := newFastTestModel()
			m.activeModal = ModalRemoveNetwork
			m.removeNetworkTargetID = "net1"
			m.removeNetworkTargetName = "app-network"
			m.removeNetworkContainersCount = 2

			var msg tea.KeyMsg
			if k == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
			res, cmd := m.handleModalInput(msg)
			m = res.(Model)
			if m.activeModal != "" || m.removeNetworkTargetID != "" || m.removeNetworkContainersCount != 0 || cmd != nil {
				t.Errorf("key %q: expected ModalRemoveNetwork cancelled cleanly", k)
			}
		}

		// Confirm with y and Y
		for _, k := range []string{"y", "Y"} {
			m := newFastTestModel()
			m.activeModal = ModalRemoveNetwork
			m.removeNetworkTargetID = "net1"
			m.removeNetworkTargetName = "app-network"
			res, cmd := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			m = res.(Model)
			if m.activeModal != "" || cmd == nil {
				t.Errorf("key %q: expected ModalRemoveNetwork confirmed and returned cmd", k)
			}
			if !strings.Contains(m.actionMessage, "Removing network app-network") {
				t.Errorf("expected toast for removing network, got %q", m.actionMessage)
			}
		}
	})

	t.Run("ModalRemoveVolume cancel variants and confirm", func(t *testing.T) {
		cancelKeys := []string{"esc", "q", "n", "N"}
		for _, k := range cancelKeys {
			m := newFastTestModel()
			m.activeModal = ModalRemoveVolume
			m.removeVolumeTargetName = "db-data"
			m.removeVolumeTargetInUse = true

			var msg tea.KeyMsg
			if k == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
			res, cmd := m.handleModalInput(msg)
			m = res.(Model)
			if m.activeModal != "" || m.removeVolumeTargetName != "" || m.removeVolumeTargetInUse || cmd != nil {
				t.Errorf("key %q: expected ModalRemoveVolume cancelled cleanly", k)
			}
		}

		// Confirm with y and Y
		for _, k := range []string{"y", "Y"} {
			m := newFastTestModel()
			m.activeModal = ModalRemoveVolume
			m.removeVolumeTargetName = "db-data"
			m.removeVolumeTargetInUse = false
			res, cmd := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			m = res.(Model)
			if m.activeModal != "" || cmd == nil {
				t.Errorf("key %q: expected ModalRemoveVolume confirmed and returned cmd", k)
			}
			if !strings.Contains(m.actionMessage, "Removing volume db-data") {
				t.Errorf("expected toast for removing volume, got %q", m.actionMessage)
			}
		}
	})
}

func TestModalPrune(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.activeModal = ModalPrune

	// Select prune option 1
	res, _ := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = res.(Model)
	if m.pruneConfirm != 1 {
		t.Errorf("expected pruneConfirm=1, got %d", m.pruneConfirm)
	}

	// Cancel with 'n'
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = res.(Model)
	if m.pruneConfirm != 0 {
		t.Errorf("expected pruneConfirm reset to 0, got %d", m.pruneConfirm)
	}

	// Select option 2 and cancel with 'esc'
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = res.(Model)
	if m.pruneConfirm != 2 {
		t.Errorf("expected pruneConfirm=2, got %d", m.pruneConfirm)
	}
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.pruneConfirm != 0 {
		t.Errorf("expected pruneConfirm reset to 0 on esc, got %d", m.pruneConfirm)
	}

	// Esc when pruneConfirm is 0 closes modal
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected prune modal closed on second esc, got %s", m.activeModal)
	}

	// Reopen, select option 3, confirm with 'y'
	m.activeModal = ModalPrune
	m.pruneConfirm = 3
	res, cmd := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = res.(Model)
	if m.activeModal != "" || cmd == nil {
		t.Errorf("expected prune executed and modal closed")
	}

	// Close with 'q'
	m.activeModal = ModalPrune
	m.pruneConfirm = 2
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = res.(Model)
	if m.activeModal != "" || m.pruneConfirm != 0 {
		t.Errorf("expected modal closed and pruneConfirm reset on 'q'")
	}
}

func TestModalLogsNavigation(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.activeModal = ModalLogs
	m.logsLines = make([]string, 50)
	for i := range m.logsLines {
		m.logsLines[i] = "log line"
	}
	m.logsScroll = 0

	// Down
	res, _ := m.handleModalInput(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.logsScroll != 1 {
		t.Errorf("expected logsScroll=1 after down, got %d", m.logsScroll)
	}

	// PgDown
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyPgDown})
	m = res.(Model)
	if m.logsScroll != 11 {
		t.Errorf("expected logsScroll=11 after pgdown, got %d", m.logsScroll)
	}

	// PgUp
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyPgUp})
	m = res.(Model)
	if m.logsScroll != 1 {
		t.Errorf("expected logsScroll=1 after pgup, got %d", m.logsScroll)
	}

	// Up
	res, _ = m.handleModalInput(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.logsScroll != 0 {
		t.Errorf("expected logsScroll=0 after up, got %d", m.logsScroll)
	}
}

func TestModalHelp(t *testing.T) {
	t.Parallel()

	for _, k := range []string{"?", "esc", "q"} {
		m := newFastTestModel()
		m.showHelp = true

		var msg tea.KeyMsg
		if k == "esc" {
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		res, _ := m.handleHelpInput(msg)
		m = res.(Model)
		if m.showHelp {
			t.Errorf("key %q: expected showHelp=false after pressing", k)
		}
	}
}

func TestInspectModalHealthBadge(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.activeModal = "inspect"
	m.inspectTarget = monitor.ContainerDetail{
		ID:     "c1",
		Name:   "app-prod",
		Image:  "my-app:v1",
		Status: "Up 2 hours",
		Health: "healthy",
	}

	view := m.View()
	if !strings.Contains(view, "Health:") || !strings.Contains(view, "[healthy]") {
		t.Errorf("expected health badge in inspect modal view, got: %s", view)
	}

	// Test unhealthy
	m.inspectTarget.Health = "unhealthy"
	view = m.View()
	if !strings.Contains(view, "[unhealthy]") {
		t.Errorf("expected [unhealthy] badge in inspect modal view, got: %s", view)
	}

	// Test starting
	m.inspectTarget.Health = "starting"
	view = m.View()
	if !strings.Contains(view, "[starting]") {
		t.Errorf("expected [starting] badge in inspect modal view, got: %s", view)
	}

	// Test no health configured
	m.inspectTarget.Health = ""
	view = m.View()
	if !strings.Contains(view, "Health:") {
		t.Errorf("expected Health line even if health is empty, got: %s", view)
	}
}

func TestInspectModalCustomLinesAndCounts(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.inspectCustomLines = []string{"header", "entry 1", "entry 2", "entry 3"}
	if count := m.countInspectLines(); count != 4 {
		t.Errorf("expected countInspectLines=4 for custom lines, got %d", count)
	}

	m.inspectCustomLines = nil
	m.inspectTarget = monitor.ContainerDetail{ID: "c1", Name: "app"}
	if count := m.countInspectLines(); count == 0 {
		t.Errorf("expected countInspectLines > 0 for ContainerDetail target")
	}

	// countFilteredLogs
	m.logsLines = []string{
		"DEBUG initialized",
		"INFO server started",
		"WARN memory high",
		"ERROR connection reset",
		"error database down",
	}
	if count := m.countFilteredLogs(); count != 5 {
		t.Errorf("expected 5 lines with empty filter, got %d", count)
	}

	m.logsFilterQuery = "ERROR"
	if count := m.countFilteredLogs(); count != 2 {
		t.Errorf("expected 2 lines matching 'ERROR' (case-insensitive), got %d", count)
	}
}

func TestPruneModalAndWorkflow(t *testing.T) {
	origDiskUsage := getDockerDiskUsage
	getDockerDiskUsage = func() (monitor.DockerDiskUsage, error) {
		return monitor.DockerDiskUsage{TotalSize: 1024 * 1024}, nil
	}
	defer func() { getDockerDiskUsage = origDiskUsage }()

	m := newFastTestModel()
	m.width = 100
	m.height = 30

	// 1. Open prune modal on tab 2 with "c"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Fatalf("expected activeModal to be 'prune', got %q", m.activeModal)
	}

	// 2. View in menu state
	view := m.View()
	if !strings.Contains(view, "CLEANUP & PRUNE DOCKER RESOURCES") {
		t.Errorf("expected prune modal header in view, got: %s", view)
	}
	if !strings.Contains(view, "Prune Stopped Containers") {
		t.Errorf("expected menu item 1 in view, got: %s", view)
	}
	if !strings.Contains(view, "[1-4]") {
		t.Errorf("expected [1-4] footer hint, got: %s", view)
	}

	// 3. Select option 1 (prune stopped containers)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = res.(Model)
	if m.pruneConfirm != 1 {
		t.Errorf("expected pruneConfirm to be 1, got %d", m.pruneConfirm)
	}
	confirmView := m.View()
	if !strings.Contains(confirmView, "WARNING: Prune Stopped Containers") {
		t.Errorf("expected confirmation prompt in view, got: %s", confirmView)
	}
	if !strings.Contains(confirmView, "[Y]") {
		t.Errorf("expected [Y] confirmation hint, got: %s", confirmView)
	}

	// 4. Cancel confirmation with "n"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = res.(Model)
	if m.pruneConfirm != 0 {
		t.Errorf("expected pruneConfirm to be 0 after 'n', got %d", m.pruneConfirm)
	}
	if m.activeModal != "prune" {
		t.Errorf("expected activeModal to remain 'prune', got %q", m.activeModal)
	}

	// 5. Select option 4 and confirm with "Y"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = res.(Model)
	if m.pruneConfirm != 4 {
		t.Errorf("expected pruneConfirm to be 4, got %d", m.pruneConfirm)
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal to close after confirmation, got %q", m.activeModal)
	}
	if m.pruneConfirm != 0 {
		t.Errorf("expected pruneConfirm to reset to 0, got %d", m.pruneConfirm)
	}

	// 6. Test opening prune modal on Tab 2 (Images) and Tab 3 (Compose)
	m.activeTab = TabImages
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Errorf("expected activeModal 'prune' on tab 2 with 'C', got %q", m.activeModal)
	}

	// 7. Close modal with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal to close on Esc, got %q", m.activeModal)
	}

	// 8. Check footer hints on Docker, Images, and Compose tabs.
	m.activeTab = TabDocker
	m.activeModal = ""
	dockerView := m.View()
	if !strings.Contains(dockerView, "[C]") || !strings.Contains(dockerView, "Prune") {
		t.Errorf("expected [C] Prune in Docker footer, got: %s", dockerView)
	}

	m.activeTab = TabImages
	imagesView := m.View()
	if !strings.Contains(imagesView, "[C]") || !strings.Contains(imagesView, "Prune") {
		t.Errorf("expected [C] Prune in Images footer, got: %s", imagesView)
	}

	m.activeTab = TabCompose
	composeView := m.View()
	if !strings.Contains(composeView, "[C]") || !strings.Contains(composeView, "Prune") {
		t.Errorf("expected [C] Prune in Compose footer, got: %s", composeView)
	}

	// 9. Check help box hint
	m.showHelp = true
	helpView := m.View()
	if !strings.Contains(helpView, "[C]") || !strings.Contains(helpView, "Prune unused Docker resources") {
		t.Errorf("expected [C] hint in help view, got: %s", helpView)
	}
}

func TestModalSwarmScaleInteraction(t *testing.T) {
	m := newFastTestModel()
	m.activeModal = ModalSwarmScale
	m.swarmScaleTargetID = "svc-web-id"
	m.swarmScaleTargetName = "web-service"
	m.swarmScaleCurrent = 3
	m.swarmScaleDesired = 3

	// 1. Verify View renders modal
	v := m.View()
	if !strings.Contains(v, "SCALE DOCKER SWARM SERVICE") || !strings.Contains(v, "web-service") {
		t.Errorf("expected swarm scale modal in View(), got: %s", v)
	}

	// 2. Increment with '+'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	m = res.(Model)
	if m.swarmScaleDesired != 4 {
		t.Errorf("expected desired=4 after '+', got %d", m.swarmScaleDesired)
	}

	// Increment with 'up'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.swarmScaleDesired != 5 {
		t.Errorf("expected desired=5 after 'up', got %d", m.swarmScaleDesired)
	}

	// 3. Decrement with '-' and 'down'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	m = res.(Model)
	if m.swarmScaleDesired != 4 {
		t.Errorf("expected desired=4 after '-', got %d", m.swarmScaleDesired)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.swarmScaleDesired != 3 {
		t.Errorf("expected desired=3 after 'down', got %d", m.swarmScaleDesired)
	}

	// Decrement below 0 is clamped to 0
	m.swarmScaleDesired = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	m = res.(Model)
	if m.swarmScaleDesired != 0 {
		t.Errorf("expected clamp at 0, got %d", m.swarmScaleDesired)
	}

	// 4. Direct digit typing: first digit replaces, next appends
	m.swarmScaleTyped = false
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = res.(Model)
	if m.swarmScaleDesired != 2 {
		t.Errorf("expected desired=2 after typing '2', got %d", m.swarmScaleDesired)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = res.(Model)
	if m.swarmScaleDesired != 25 {
		t.Errorf("expected desired=25 after typing '5', got %d", m.swarmScaleDesired)
	}

	// 5. Backspace removes last digit
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.swarmScaleDesired != 2 {
		t.Errorf("expected desired=2 after backspace, got %d", m.swarmScaleDesired)
	}

	// 6. Escape cancels modal
	mEsc := m
	res, _ = mEsc.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mEsc = res.(Model)
	if mEsc.activeModal != "" {
		t.Errorf("expected modal closed after Esc, got %s", mEsc.activeModal)
	}

	// 7. 'q' cancels modal
	mQ := m
	res, _ = mQ.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	mQ = res.(Model)
	if mQ.activeModal != "" {
		t.Errorf("expected modal closed after 'q', got %s", mQ.activeModal)
	}

	// 8. Enter applies scale and produces command
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mApplied := res.(Model)
	if mApplied.activeModal != "" {
		t.Errorf("expected modal closed after Enter, got %s", mApplied.activeModal)
	}
	if cmd == nil {
		t.Errorf("expected non-nil cmd on Enter")
	}
	if !strings.Contains(mApplied.actionMessage, "Scaling service web-service to 2 replicas") {
		t.Errorf("expected toast with scaling message, got %s", mApplied.actionMessage)
	}
}

// =============================================================================
// Tests from keys_modals_exec_test.go
// =============================================================================

func TestHandleExecModalInput(t *testing.T) {
	originalStart := startEmbeddedShell
	startEmbeddedShell = func(_ *exec.Cmd, _, _ int) (embeddedShellSession, error) {
		return &fakeEmbeddedShell{}, nil
	}
	defer func() { startEmbeddedShell = originalStart }()

	initModalModel := func() Model {
		m := newFastTestModel()
		m.dockerInfo.Containers = []monitor.ContainerItem{
			testutil.AContainer(testutil.WithID("c1"), testutil.WithName("test-box"), testutil.WithState("running", "Up")),
		}
		m.activeModal = ModalExec
		m.execTargetID = "c1"
		m.execTargetName = "test-box"
		m.execCursorIdx = 0
		m.execIsCustomInput = false
		m.execCustomCmd = ""
		return m
	}

	t.Run("navigation with wrapping", func(t *testing.T) {
		m := initModalModel()

		// up/k wraps from 0 to 6
		for _, key := range []tea.KeyMsg{
			{Type: tea.KeyRunes, Runes: []rune("k")},
			{Type: tea.KeyUp},
		} {
			m.execCursorIdx = 0
			res, _ := m.handleExecModalInput(key)
			m = res.(Model)
			if m.execCursorIdx != 6 {
				t.Errorf("key %v: expected wrap from 0 to 6, got %d", key, m.execCursorIdx)
			}
		}

		// down/j wraps from 6 to 0
		for _, key := range []tea.KeyMsg{
			{Type: tea.KeyRunes, Runes: []rune("j")},
			{Type: tea.KeyDown},
		} {
			m.execCursorIdx = 6
			res, _ := m.handleExecModalInput(key)
			m = res.(Model)
			if m.execCursorIdx != 0 {
				t.Errorf("key %v: expected wrap from 6 to 0, got %d", key, m.execCursorIdx)
			}
		}

		// step-by-step navigation
		m.execCursorIdx = 2
		res, _ := m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyDown})
		m = res.(Model)
		if m.execCursorIdx != 3 {
			t.Errorf("expected cursor 3 after down from 2, got %d", m.execCursorIdx)
		}

		res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyUp})
		m = res.(Model)
		if m.execCursorIdx != 2 {
			t.Errorf("expected cursor 2 after up from 3, got %d", m.execCursorIdx)
		}
	})

	t.Run("modal cancellation via esc and q", func(t *testing.T) {
		for _, keyStr := range []string{"esc", "q"} {
			m := initModalModel()
			var msg tea.KeyMsg
			if keyStr == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEscape}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyStr)}
			}

			res, cmd := m.handleExecModalInput(msg)
			m = res.(Model)
			if cmd != nil {
				t.Errorf("key %s: expected nil cmd on cancel", keyStr)
			}
			if m.activeModal != ModalNone {
				t.Errorf("key %s: expected activeModal to be cleared, got %q", keyStr, m.activeModal)
			}
			if m.execTargetID != "" || m.execTargetName != "" {
				t.Errorf("key %s: expected target ID/Name to be cleared, got ID=%q Name=%q", keyStr, m.execTargetID, m.execTargetName)
			}
		}
	})

	t.Run("direct shortcuts for all execution modes", func(t *testing.T) {
		tests := []struct {
			name            string
			keys            []string
			expectedCustom  bool
			expectedCmd     bool
			expectedClosed  bool
			isEmbeddedShell bool
		}{
			{
				name:            "Option 1: Embedded shell ('1', 'b', 'B')",
				keys:            []string{"1", "b", "B"},
				expectedCmd:     true,
				isEmbeddedShell: true,
			},
			{
				name:           "Option 2: Sh process ('2', 's', 'S')",
				keys:           []string{"2", "s", "S"},
				expectedCmd:    true,
				expectedClosed: true,
			},
			{
				name:           "Option 3: Zsh process ('3', 'z', 'Z')",
				keys:           []string{"3", "z", "Z"},
				expectedCmd:    true,
				expectedClosed: true,
			},
			{
				name:           "Option 4: Attach process ('4', 'a', 'A')",
				keys:           []string{"4", "a", "A"},
				expectedCmd:    true,
				expectedClosed: true,
			},
			{
				name:           "Option 5: Root shell process ('5', 'r', 'R')",
				keys:           []string{"5", "r", "R"},
				expectedCmd:    true,
				expectedClosed: true,
			},
			{
				name:           "Option 6: Custom input mode ('6', 'c', 'C')",
				keys:           []string{"6", "c", "C"},
				expectedCustom: true,
				expectedClosed: false,
			},
			{
				name:           "Option 7: Auto shell process ('7')",
				keys:           []string{"7"},
				expectedCmd:    true,
				expectedClosed: true,
			},
		}

		for _, tt := range tests {
			for _, k := range tt.keys {
				m := initModalModel()
				res, cmd := m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
				m = res.(Model)

				if tt.expectedCmd && cmd == nil {
					t.Errorf("%s (key %q): expected non-nil cmd", tt.name, k)
				}
				if tt.expectedCustom && !m.execIsCustomInput {
					t.Errorf("%s (key %q): expected execIsCustomInput=true", tt.name, k)
				}
				if tt.expectedClosed && m.activeModal != ModalNone {
					t.Errorf("%s (key %q): expected activeModal to close, got %q", tt.name, k, m.activeModal)
				}
				if tt.isEmbeddedShell && m.embeddedShell == nil {
					t.Errorf("%s (key %q): expected embedded shell session to be initialized", tt.name, k)
				}
			}
		}
	})

	t.Run("selection via enter key on each cursor index", func(t *testing.T) {
		tests := []struct {
			name            string
			cursorIdx       int
			expectedCustom  bool
			expectedCmd     bool
			expectedClosed  bool
			isEmbeddedShell bool
		}{
			{name: "Enter on index 0 (Embedded Shell)", cursorIdx: 0, expectedCmd: true, isEmbeddedShell: true},
			{name: "Enter on index 1 (Sh)", cursorIdx: 1, expectedCmd: true, expectedClosed: true},
			{name: "Enter on index 2 (Zsh)", cursorIdx: 2, expectedCmd: true, expectedClosed: true},
			{name: "Enter on index 3 (Attach)", cursorIdx: 3, expectedCmd: true, expectedClosed: true},
			{name: "Enter on index 4 (Root)", cursorIdx: 4, expectedCmd: true, expectedClosed: true},
			{name: "Enter on index 5 (Custom Mode)", cursorIdx: 5, expectedCustom: true, expectedClosed: false},
			{name: "Enter on index 6 (Auto)", cursorIdx: 6, expectedCmd: true, expectedClosed: true},
		}

		for _, tt := range tests {
			m := initModalModel()
			m.execCursorIdx = tt.cursorIdx
			res, cmd := m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyEnter})
			m = res.(Model)

			if tt.expectedCmd && cmd == nil {
				t.Errorf("%s: expected non-nil cmd", tt.name)
			}
			if tt.expectedCustom && !m.execIsCustomInput {
				t.Errorf("%s: expected execIsCustomInput=true", tt.name)
			}
			if tt.expectedClosed && m.activeModal != ModalNone {
				t.Errorf("%s: expected modal closed, got %q", tt.name, m.activeModal)
			}
			if tt.isEmbeddedShell && m.embeddedShell == nil {
				t.Errorf("%s: expected embedded shell session initialized", tt.name)
			}
		}
	})

	t.Run("custom input mode editing and launch", func(t *testing.T) {
		m := initModalModel()
		m.execIsCustomInput = true

		// Typing runes and space
		for _, r := range "ls -la /tmp" {
			var msg tea.KeyMsg
			if r == ' ' {
				msg = tea.KeyMsg{Type: tea.KeySpace}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			}
			res, _ := m.handleExecModalInput(msg)
			m = res.(Model)
		}
		if m.execCustomCmd != "ls -la /tmp" {
			t.Fatalf("expected custom cmd 'ls -la /tmp', got %q", m.execCustomCmd)
		}

		// Backspace deletes single character
		res, _ := m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyBackspace})
		m = res.(Model)
		if m.execCustomCmd != "ls -la /tm" {
			t.Errorf("expected 'ls -la /tm' after backspace, got %q", m.execCustomCmd)
		}

		// Unicode backspace safety
		m.execCustomCmd = "тест"
		res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyBackspace})
		m = res.(Model)
		if m.execCustomCmd != "тес" {
			t.Errorf("expected 'тес' after unicode backspace, got %q", m.execCustomCmd)
		}

		// Backspace on empty string does not panic
		m.execCustomCmd = ""
		res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyBackspace})
		m = res.(Model)
		if m.execCustomCmd != "" {
			t.Errorf("expected empty string after backspace on empty, got %q", m.execCustomCmd)
		}

		// Ignored non-character keys (e.g. up, down, tab)
		res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyUp})
		m = res.(Model)
		if m.execCustomCmd != "" {
			t.Errorf("expected empty string after up key in custom mode, got %q", m.execCustomCmd)
		}

		// Esc cancels custom input mode and returns to modal
		m.execCustomCmd = "some command"
		res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyEsc})
		m = res.(Model)
		if m.execIsCustomInput {
			t.Errorf("expected execIsCustomInput=false after esc")
		}
		if m.activeModal != ModalExec {
			t.Errorf("expected activeModal to stay ModalExec after esc, got %q", m.activeModal)
		}

		// Enter with empty or whitespace string cancels custom mode without launching
		m.execIsCustomInput = true
		m.execCustomCmd = "   "
		res, cmd := m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyEnter})
		m = res.(Model)
		if m.execIsCustomInput {
			t.Errorf("expected execIsCustomInput=false after enter with whitespace")
		}
		if cmd != nil {
			t.Errorf("expected nil cmd after enter with whitespace")
		}
		if m.activeModal != ModalExec {
			t.Errorf("expected modal to remain open after whitespace cancel, got %q", m.activeModal)
		}

		// Enter with non-empty command launches custom exec process and closes modal
		m.execIsCustomInput = true
		m.execCustomCmd = "top -b -n 1"
		res, cmd = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyEnter})
		m = res.(Model)
		if cmd == nil {
			t.Fatalf("expected non-nil cmd after enter with valid custom command")
		}
		if m.activeModal != ModalNone {
			t.Errorf("expected activeModal to close on custom command launch, got %q", m.activeModal)
		}
		if m.execTargetID != "" {
			t.Errorf("expected execTargetID to be cleared, got %q", m.execTargetID)
		}
	})

	t.Run("unhandled keys in menu mode", func(t *testing.T) {
		m := initModalModel()
		for _, key := range []tea.KeyMsg{
			{Type: tea.KeyRunes, Runes: []rune("x")},
			{Type: tea.KeyTab},
		} {
			res, cmd := m.handleExecModalInput(key)
			m = res.(Model)
			if cmd != nil {
				t.Errorf("key %v: expected nil cmd", key)
			}
			if m.activeModal != ModalExec {
				t.Errorf("key %v: expected modal to stay open", key)
			}
		}
	})
}

func TestExecModal_ViewRendering(t *testing.T) {
	m := newFastTestModel()
	m.activeModal = ModalExec
	m.execTargetID = "c123456"
	m.execTargetName = "production-app"
	m.execCursorIdx = 0
	m.width = 100
	m.height = 30

	// 1. Menu view rendering
	view := m.View()
	if !strings.Contains(view, "production-app") {
		t.Errorf("expected view to contain target container name 'production-app', got:\n%s", view)
	}
	if !strings.Contains(view, "c123456") {
		t.Errorf("expected view to contain target container id 'c123456', got:\n%s", view)
	}
	if !strings.Contains(view, "Embedded Bash / Sh") {
		t.Errorf("expected view to contain 'Embedded Bash / Sh', got:\n%s", view)
	}

	// 2. Custom input view rendering
	m.execIsCustomInput = true
	m.execCustomCmd = "netstat -tuln"
	customView := m.View()
	if !strings.Contains(customView, "netstat -tuln") {
		t.Errorf("expected customView to contain 'netstat -tuln', got:\n%s", customView)
	}
}

func TestExecModalWorkflow(t *testing.T) {
	restore := monitor.SetDockerCLIAvailableCheckerForTesting(func() bool { return true })
	defer restore()

	m := newFastTestModel()
	m.dockerInfo = monitor.DockerSummary{
		Status: "running",
		Containers: []monitor.ContainerItem{
			testutil.AContainer(testutil.WithID("c123"), testutil.WithName("app-test"), testutil.WithState("running", "Up")),
		},
	}
	m.cursorIdx = 0

	// 1. Open exec modal with "e"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(Model)
	if m.activeModal != ModalExec {
		t.Fatalf("expected activeModal to be ModalExec, got %q", m.activeModal)
	}
	if m.execTargetID != "c123" || m.execTargetName != "app-test" {
		t.Fatalf("expected target c123/app-test, got ID=%q Name=%q", m.execTargetID, m.execTargetName)
	}

	// 2. Navigate down and up in menu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.execCursorIdx != 1 {
		t.Errorf("expected execCursorIdx 1 after down, got %d", m.execCursorIdx)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.execCursorIdx != 0 {
		t.Errorf("expected execCursorIdx 0 after up, got %d", m.execCursorIdx)
	}

	// Wrap around up
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.execCursorIdx != 6 {
		t.Errorf("expected execCursorIdx 6 after wrap up, got %d", m.execCursorIdx)
	}

	// 3. Switch to custom command mode via 'c'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = res.(Model)
	if !m.execIsCustomInput {
		t.Fatalf("expected execIsCustomInput to be true")
	}

	// Type custom command "python3"
	for _, r := range "python3" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}
	if m.execCustomCmd != "python3" {
		t.Errorf("expected execCustomCmd 'python3', got %q", m.execCustomCmd)
	}

	// Backspace
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.execCustomCmd != "python" {
		t.Errorf("expected execCustomCmd 'python', got %q", m.execCustomCmd)
	}

	// Esc in custom command goes back to menu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = res.(Model)
	if m.execIsCustomInput {
		t.Errorf("expected execIsCustomInput false after Esc")
	}
	if m.activeModal != ModalExec {
		t.Errorf("expected activeModal still ModalExec")
	}

	// Direct shortcut "4" launches attach
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = res.(Model)
	if m.activeModal != ModalNone {
		t.Errorf("expected modal to close on launching exec, got %q", m.activeModal)
	}
	if cmd == nil {
		t.Errorf("expected non-nil tea.Cmd for launching attach")
	}
}

// =============================================================================
// Tests from keys_modals_confirm_test.go
// =============================================================================

func TestModalConfirmCancelKeys(t *testing.T) {
	t.Parallel()

	cancelKeys := []string{"esc", "q", "n", "N"}
	for _, k := range cancelKeys {
		if !isModalCancelKey(k) {
			t.Errorf("expected isModalCancelKey(%q) to be true", k)
		}
	}

	confirmKeys := []string{"y", "Y"}
	for _, k := range confirmKeys {
		if !isModalConfirmKey(k) {
			t.Errorf("expected isModalConfirmKey(%q) to be true", k)
		}
	}

	otherKeys := []string{"x", "a", "enter", "tab", "1"}
	for _, k := range otherKeys {
		if isModalCancelKey(k) {
			t.Errorf("expected isModalCancelKey(%q) to be false", k)
		}
		if isModalConfirmKey(k) {
			t.Errorf("expected isModalConfirmKey(%q) to be false", k)
		}
	}
}

func TestDirectConfirmModalHandlers(t *testing.T) {
	t.Parallel()

	t.Run("handleComposeDownModalInput cancel", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalComposeDown
		m.composeDownTarget = "my-stack"

		res, cmd := m.handleComposeDownModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd on cancel")
		}
		if updated.activeModal != ModalNone {
			t.Errorf("expected modal closed")
		}
		if updated.composeDownTarget != "" {
			t.Errorf("expected target cleared")
		}
	})

	t.Run("handleComposeDownModalInput confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalComposeDown
		m.composeDownTarget = "my-stack"
		m.dockerInfo = monitor.DockerSummary{
			Containers: []monitor.ContainerItem{{Name: "web", ComposeProject: "my-stack"}},
		}

		res, cmd := m.handleComposeDownModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		updated := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on confirm")
		}
		if updated.activeModal != ModalNone {
			t.Errorf("expected modal closed")
		}
	})

	t.Run("handleRemoveContainerModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveContainer
		m.removeContainerTargetID = "c123"
		m.removeContainerTargetName = "web-container"

		res, cmd := m.handleRemoveContainerModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeContainerTargetID != "" {
			t.Errorf("unexpected state on cancel: %+v", updated)
		}

		m.activeModal = ModalRemoveContainer
		m.removeContainerTargetID = "c123"
		m.removeContainerTargetName = "web-container"
		res, cmd = m.handleRemoveContainerModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})

	t.Run("handleRemoveImageModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveImage
		m.removeImageTargetID = "img123"
		m.removeImageTargetName = "alpine:latest"
		m.removeImageTargetInUse = true

		res, cmd := m.handleRemoveImageModalInput(tea.KeyMsg{Type: tea.KeyEsc})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeImageTargetID != "" {
			t.Errorf("unexpected state on cancel")
		}

		m.activeModal = ModalRemoveImage
		m.removeImageTargetID = "img123"
		m.removeImageTargetName = "alpine:latest"
		m.removeImageTargetInUse = true
		res, cmd = m.handleRemoveImageModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})

	t.Run("handleRemoveNetworkModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveNetwork
		m.removeNetworkTargetID = "net123"
		m.removeNetworkTargetName = "bridge-net"

		res, cmd := m.handleRemoveNetworkModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeNetworkTargetID != "" {
			t.Errorf("unexpected state on cancel")
		}

		m.activeModal = ModalRemoveNetwork
		m.removeNetworkTargetID = "net123"
		m.removeNetworkTargetName = "bridge-net"
		res, cmd = m.handleRemoveNetworkModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})

	t.Run("handleRemoveVolumeModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveVolume
		m.removeVolumeTargetName = "vol-data"
		m.removeVolumeTargetInUse = true

		res, cmd := m.handleRemoveVolumeModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeVolumeTargetName != "" {
			t.Errorf("unexpected state on cancel")
		}

		m.activeModal = ModalRemoveVolume
		m.removeVolumeTargetName = "vol-data"
		m.removeVolumeTargetInUse = true
		res, cmd = m.handleRemoveVolumeModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})
}

// =============================================================================
// Tests from keys_modals_viewer_test.go
// =============================================================================

func TestDirectViewerModalHandlers(t *testing.T) {
	t.Parallel()

	t.Run("handleHelpInput", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.showHelp = true

		res, cmd := m.handleHelpInput(tea.KeyMsg{Type: tea.KeyEsc})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd")
		}
		if updated.showHelp {
			t.Errorf("expected showHelp false after Esc")
		}

		m.showHelp = true
		res, _ = m.handleHelpInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		updated = res.(Model)
		if updated.showHelp {
			t.Errorf("expected showHelp false after '?'")
		}
	})

	t.Run("handleLogsModalInput scrolling and cancel", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalLogs
		m.logsLines = []string{
			"line 1", "line 2", "line 3", "line 4", "line 5",
			"line 6", "line 7", "line 8", "line 9", "line 10",
			"line 11", "line 12", "line 13", "line 14", "line 15",
		}
		m.logsScroll = 0

		// Down / j scrolls
		res, _ := m.handleLogsModalInput(tea.KeyMsg{Type: tea.KeyDown})
		updated := res.(Model)
		if updated.logsScroll != 1 {
			t.Errorf("expected logsScroll=1 after Down, got %d", updated.logsScroll)
		}

		// Up / k scrolls
		res, _ = updated.handleLogsModalInput(tea.KeyMsg{Type: tea.KeyUp})
		updated = res.(Model)
		if updated.logsScroll != 0 {
			t.Errorf("expected logsScroll=0 after Up, got %d", updated.logsScroll)
		}

		// PgDown
		res, _ = updated.handleLogsModalInput(tea.KeyMsg{Type: tea.KeyPgDown})
		updated = res.(Model)
		if updated.logsScroll == 0 {
			t.Errorf("expected logsScroll > 0 after PgDown")
		}

		// PgUp
		res, _ = updated.handleLogsModalInput(tea.KeyMsg{Type: tea.KeyPgUp})
		updated = res.(Model)
		if updated.logsScroll != 0 {
			t.Errorf("expected logsScroll=0 after PgUp, got %d", updated.logsScroll)
		}

		// Quit
		res, _ = updated.handleLogsModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		updated = res.(Model)
		if updated.activeModal != ModalNone {
			t.Errorf("expected modal closed on 'q'")
		}
	})

	t.Run("handleInspectModalInput scrolling and copy", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalInspect
		m.inspectCustomLines = make([]string, 30)
		for i := range m.inspectCustomLines {
			m.inspectCustomLines[i] = "detail line"
		}
		m.inspectCopyID = "test-target-id"

		// Down
		res, _ := m.handleInspectModalInput(tea.KeyMsg{Type: tea.KeyDown})
		updated := res.(Model)
		if updated.inspectScroll != 1 {
			t.Errorf("expected inspectScroll=1, got %d", updated.inspectScroll)
		}

		// Up
		res, _ = updated.handleInspectModalInput(tea.KeyMsg{Type: tea.KeyUp})
		updated = res.(Model)
		if updated.inspectScroll != 0 {
			t.Errorf("expected inspectScroll=0, got %d", updated.inspectScroll)
		}

		// Esc closes modal
		res, _ = updated.handleInspectModalInput(tea.KeyMsg{Type: tea.KeyEsc})
		updated = res.(Model)
		if updated.activeModal != ModalNone {
			t.Errorf("expected modal closed on Esc")
		}
	})

	t.Run("handlePruneModalInput options and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalPrune
		m.pruneConfirm = 0

		// Selecting option '2' (images)
		res, _ := m.handlePruneModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
		updated := res.(Model)
		if updated.pruneConfirm != 2 {
			t.Errorf("expected pruneConfirm=2, got %d", updated.pruneConfirm)
		}

		// 'n' cancels confirmation
		res, _ = updated.handlePruneModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		updated = res.(Model)
		if updated.pruneConfirm != 0 {
			t.Errorf("expected pruneConfirm reset to 0, got %d", updated.pruneConfirm)
		}

		// '3' (volumes) then 'y' triggers pruneCmd
		res, _ = updated.handlePruneModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		updated = res.(Model)
		res, cmd := updated.handlePruneModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected prune command scheduled and modal closed")
		}
		if !strings.Contains(updated.actionMessage, "Pruning") {
			t.Errorf("expected pruning toast message, got %q", updated.actionMessage)
		}
	})

	t.Run("countInspectLines and countFilteredLogs", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.inspectCustomLines = []string{"line1", "line2", "line3"}
		if m.countInspectLines() != 3 {
			t.Errorf("expected countInspectLines=3, got %d", m.countInspectLines())
		}

		m.inspectCustomLines = nil
		m.inspectTarget = monitor.ContainerDetail{Name: "c1", Image: "img1"}
		if m.countInspectLines() <= 0 {
			t.Errorf("expected countInspectLines > 0 for non-empty ContainerDetail")
		}

		m.logsLines = []string{"foo", "bar", "foobar"}
		m.logsFilterQuery = "foo"
		if m.countFilteredLogs() != 2 {
			t.Errorf("expected countFilteredLogs=2, got %d", m.countFilteredLogs())
		}
	})
}
