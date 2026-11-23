package tui

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

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
