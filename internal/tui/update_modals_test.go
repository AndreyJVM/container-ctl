package tui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

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

	// 5. Press Esc again to close modal
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected modal closed after second Esc, got %s", m.activeModal)
	}
}

func TestModalInspectScrollingAndCopy(t *testing.T) {
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
