package tui

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func newFastComposeTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabCompose
	m.compose = []monitor.ComposeProject{
		{
			Name:            "proj-alpha",
			Status:          "running",
			StatusDisplay:   "Running",
			TotalServices:   2,
			RunningServices: 2,
			Services: []monitor.ComposeService{
				{Name: "web", State: "running"},
				{Name: "api", State: "running"},
			},
		},
		{
			Name:            "proj-beta",
			Status:          "stopped",
			StatusDisplay:   "Stopped",
			TotalServices:   1,
			RunningServices: 0,
			Services: []monitor.ComposeService{
				{Name: "worker", State: "exited"},
			},
		},
	}
	m.composeCursorIdx = 0
	return m
}

func TestComposeTabNavigationAndActions(t *testing.T) {
	t.Parallel()

	t.Run("cursor navigation with wrapping", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()

		// Down moves to second project
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = res.(Model)
		if m.composeCursorIdx != 1 {
			t.Errorf("expected composeCursorIdx=1, got %d", m.composeCursorIdx)
		}

		// Down wraps back to 0
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = res.(Model)
		if m.composeCursorIdx != 0 {
			t.Errorf("expected composeCursorIdx=0 after wrap, got %d", m.composeCursorIdx)
		}

		// Up wraps to 1
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = res.(Model)
		if m.composeCursorIdx != 1 {
			t.Errorf("expected composeCursorIdx=1 after up wrap, got %d", m.composeCursorIdx)
		}

		// Key 'k' moves to 0
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = res.(Model)
		if m.composeCursorIdx != 0 {
			t.Errorf("expected composeCursorIdx=0 after 'k', got %d", m.composeCursorIdx)
		}

		// Key 'j' moves to 1
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = res.(Model)
		if m.composeCursorIdx != 1 {
			t.Errorf("expected composeCursorIdx=1 after 'j', got %d", m.composeCursorIdx)
		}
	})

	t.Run("view rendering contains footer hints and selection marker", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()
		view := m.View()

		for _, hint := range []string{"Logs", "Restart", "Stop", "Start", "Down"} {
			if !strings.Contains(view, hint) {
				t.Errorf("expected footer view to contain hint %q, got: %s", hint, view)
			}
		}
		if !strings.Contains(view, "► ") {
			t.Errorf("expected cursor marker '► ' in compose tab view")
		}
	})

	t.Run("action hotkeys start stop restart logs", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()
		m.composeCursorIdx = 1 // target proj-beta

		// Test 'p' restarts compose project
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
		mRestart := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on 'p'")
		}
		if !strings.Contains(mRestart.actionMessage, "Restarting Compose project proj-beta") {
			t.Errorf("expected restarting toast, got: %q", mRestart.actionMessage)
		}

		// Test 's' stops compose project
		res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		mStop := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on 's'")
		}
		if !strings.Contains(mStop.actionMessage, "Stopping Compose project proj-beta") {
			t.Errorf("expected stopping toast, got: %q", mStop.actionMessage)
		}

		// Test 'u' starts compose project
		res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
		mStart := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on 'u'")
		}
		if !strings.Contains(mStart.actionMessage, "Starting Compose project proj-beta") {
			t.Errorf("expected starting toast, got: %q", mStart.actionMessage)
		}

		// Test 'l' opens logs modal or triggers toast feedback
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		mLogs := res.(Model)
		if mLogs.activeModal == ModalLogs {
			if !strings.Contains(mLogs.logsTargetName, "proj-beta") {
				t.Errorf("expected logsTargetName to contain 'proj-beta', got %q", mLogs.logsTargetName)
			}
		} else if !strings.Contains(mLogs.actionMessage, "Logs error") {
			t.Errorf("expected logs modal or error toast, got message=%q", mLogs.actionMessage)
		}
	})

	t.Run("clipboard copy project name with 'y'", func(t *testing.T) {
		origWriter := clipboard.OutputWriter
		var buf bytes.Buffer
		clipboard.OutputWriter = &buf
		defer func() { clipboard.OutputWriter = origWriter }()

		m := newFastComposeTestModel()
		m.composeCursorIdx = 0

		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd from clipboard copy")
		}
		if !strings.Contains(updated.actionMessage, "Copied to clipboard: proj-alpha") {
			t.Errorf("expected copy toast, got: %q", updated.actionMessage)
		}
		expectedB64 := base64.StdEncoding.EncodeToString([]byte("proj-alpha"))
		if !strings.Contains(buf.String(), expectedB64) {
			t.Errorf("expected clipboard output to contain base64 %q, got: %q", expectedB64, buf.String())
		}
	})

	t.Run("compose down modal workflow confirmation and cancel", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()
		m.composeCursorIdx = 1 // proj-beta

		// Open modal with 'd'
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
		m = res.(Model)
		if m.activeModal != ModalComposeDown {
			t.Fatalf("expected activeModal 'compose_down', got %q", m.activeModal)
		}
		if m.composeDownTarget != "proj-beta" {
			t.Errorf("expected composeDownTarget 'proj-beta', got %q", m.composeDownTarget)
		}

		modalView := m.View()
		if !strings.Contains(modalView, "CONFIRM DOCKER COMPOSE DOWN") || !strings.Contains(modalView, "proj-beta") {
			t.Errorf("expected compose down modal content, got: %s", modalView)
		}

		// Cancel modal with 'n'
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
		mCancel := res.(Model)
		if mCancel.activeModal != "" || mCancel.composeDownTarget != "" {
			t.Errorf("expected modal closed and target cleared after 'n', got modal=%q", mCancel.activeModal)
		}

		// Cancel modal with 'esc'
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		mCancelEsc := res.(Model)
		if mCancelEsc.activeModal != "" {
			t.Errorf("expected modal closed after 'esc'")
		}

		// Cancel modal with 'q'
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		mCancelQ := res.(Model)
		if mCancelQ.activeModal != "" {
			t.Errorf("expected modal closed after 'q'")
		}

		// Re-open and confirm with 'y'
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		mConfirm := res.(Model)
		if mConfirm.activeModal != "" || mConfirm.composeDownTarget != "" {
			t.Errorf("expected activeModal cleared after 'y'")
		}
		if cmd == nil {
			t.Errorf("expected non-nil tea.Cmd for downComposeCmd")
		}
		if !strings.Contains(mConfirm.actionMessage, "Downing Compose project proj-beta") {
			t.Errorf("expected downing toast, got: %q", mConfirm.actionMessage)
		}
	})
}

func TestComposeNavigationKeys(t *testing.T) {
	t.Parallel()

	m := newFastComposeTestModel()
	m.compose = append(m.compose,
		monitor.ComposeProject{Name: "proj-gamma"},
		monitor.ComposeProject{Name: "proj-delta"},
		monitor.ComposeProject{Name: "proj-epsilon"},
		monitor.ComposeProject{Name: "proj-zeta"},
	) // Total 6 projects

	m.composeCursorIdx = 0

	// End / 'G' moves to last element
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = res.(Model)
	if m.composeCursorIdx != 5 {
		t.Errorf("expected composeCursorIdx=5 after 'G', got %d", m.composeCursorIdx)
	}

	// Home / 'g' moves to first element
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = res.(Model)
	if m.composeCursorIdx != 0 {
		t.Errorf("expected composeCursorIdx=0 after 'g', got %d", m.composeCursorIdx)
	}

	// PageDown moves cursor forward by 5
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = res.(Model)
	if m.composeCursorIdx != 5 {
		t.Errorf("expected composeCursorIdx=5 after PgDown, got %d", m.composeCursorIdx)
	}

	// PageUp moves cursor back by 5
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = res.(Model)
	if m.composeCursorIdx != 0 {
		t.Errorf("expected composeCursorIdx=0 after PgUp, got %d", m.composeCursorIdx)
	}
}

func TestComposeActions_EmptyListSafety(t *testing.T) {
	t.Parallel()

	m := newFastComposeTestModel()
	m.compose = nil
	m.composeCursorIdx = 0

	// Pressing actions on empty list does not panic and returns nil cmd
	actionKeys := []string{"s", "S", "u", "U", "p", "P", "d", "D", "l", "L", "y", "Y"}
	for _, k := range actionKeys {
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd when key %q pressed on empty list", k)
		}
		if updated.activeModal != "" {
			t.Errorf("expected no modal opened for key %q on empty list", k)
		}
	}
}
