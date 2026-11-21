package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModalLogsSearch(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.16.0")
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
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, 1, "v0.4.1")

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
	m := NewModel(2*time.Second, 1, "v1.0.0")

	// 1. ModalComposeDown cancel
	m.activeModal = ModalComposeDown
	m.composeDownTarget = "my-stack"
	res, cmd := m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = res.(Model)
	if m.activeModal != "" || m.composeDownTarget != "" || cmd != nil {
		t.Errorf("expected ModalComposeDown cancelled cleanly")
	}

	// ModalComposeDown confirm
	m.activeModal = ModalComposeDown
	m.composeDownTarget = "my-stack"
	res, cmd = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = res.(Model)
	if m.activeModal != "" || cmd == nil {
		t.Errorf("expected ModalComposeDown confirmed and returned cmd")
	}

	// 2. ModalRemoveContainer cancel
	m.activeModal = ModalRemoveContainer
	m.removeContainerTargetID = "c1"
	m.removeContainerTargetName = "web"
	res, cmd = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = res.(Model)
	if m.activeModal != "" || m.removeContainerTargetID != "" || cmd != nil {
		t.Errorf("expected ModalRemoveContainer cancelled cleanly")
	}

	// ModalRemoveContainer confirm
	m.activeModal = ModalRemoveContainer
	m.removeContainerTargetID = "c1"
	m.removeContainerTargetName = "web"
	res, cmd = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = res.(Model)
	if m.activeModal != "" || cmd == nil {
		t.Errorf("expected ModalRemoveContainer confirmed and returned cmd")
	}

	// 3. ModalRemoveImage cancel
	m.activeModal = ModalRemoveImage
	m.removeImageTargetID = "img1"
	m.removeImageTargetName = "nginx:latest"
	res, cmd = m.handleModalInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != "" || m.removeImageTargetID != "" || cmd != nil {
		t.Errorf("expected ModalRemoveImage cancelled cleanly")
	}

	// ModalRemoveImage confirm
	m.activeModal = ModalRemoveImage
	m.removeImageTargetID = "img1"
	m.removeImageTargetName = "nginx:latest"
	res, cmd = m.handleModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = res.(Model)
	if m.activeModal != "" || cmd == nil {
		t.Errorf("expected ModalRemoveImage confirmed and returned cmd")
	}
}

func TestModalPrune(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v1.0.0")
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
}

func TestModalLogsNavigation(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v1.0.0")
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
	m := NewModel(2*time.Second, 1, "v1.0.0")
	m.showHelp = true

	res, _ := m.handleHelpInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = res.(Model)
	if m.showHelp {
		t.Errorf("expected showHelp=false after '?'")
	}

	m.showHelp = true
	res, _ = m.handleHelpInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.showHelp {
		t.Errorf("expected showHelp=false after 'esc'")
	}
}
