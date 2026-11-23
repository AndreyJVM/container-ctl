package tui

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

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
