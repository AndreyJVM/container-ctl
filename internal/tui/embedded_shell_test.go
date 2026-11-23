package tui

import (
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type fakeEmbeddedShell struct {
	writes       [][]byte
	feed         []byte
	resizeWidth  int
	resizeHeight int
	closeCount   int
	render       string
	feedErr      error
	writeErr     error
	resizeErr    error
	readData     []byte
	readErr      error
	waitErr      error
}

func (s *fakeEmbeddedShell) Read() ([]byte, error) {
	if s.readErr != nil || len(s.readData) > 0 {
		return s.readData, s.readErr
	}
	return nil, io.EOF
}

func (s *fakeEmbeddedShell) Write(data []byte) error {
	s.writes = append(s.writes, append([]byte(nil), data...))
	return s.writeErr
}

func (s *fakeEmbeddedShell) Feed(data []byte) error {
	s.feed = append(s.feed, data...)
	return s.feedErr
}

func (s *fakeEmbeddedShell) Render() string { return s.render }

func (s *fakeEmbeddedShell) Resize(width, height int) error {
	s.resizeWidth, s.resizeHeight = width, height
	return s.resizeErr
}

func (s *fakeEmbeddedShell) Wait() error { return s.waitErr }

func (s *fakeEmbeddedShell) Close() error {
	s.closeCount++
	return nil
}

func TestEmbeddedShellLifecycleAndFocus(t *testing.T) {
	originalStart := startEmbeddedShell
	defer func() { startEmbeddedShell = originalStart }()

	fake := &fakeEmbeddedShell{render: "root@api:/app#"}
	var startedCommand *exec.Cmd
	startEmbeddedShell = func(cmd *exec.Cmd, _, _ int) (embeddedShellSession, error) {
		startedCommand = cmd
		return fake, nil
	}

	m := newFastTestModel()
	m.width, m.height = 140, 40
	m.activeModal = ModalExec
	m.execTargetID = "container-id"
	m.execTargetName = "api"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("expected shell read and wait commands")
	}
	if m.embeddedShell != fake || !m.shellFocused {
		t.Fatalf("expected an active focused embedded shell")
	}
	if startedCommand == nil || !strings.Contains(strings.Join(startedCommand.Args, " "), "TERM=xterm-256color") {
		t.Fatalf("expected an xterm-compatible Docker exec command, got %#v", startedCommand)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("pwd")})
	m = updated.(Model)
	if len(fake.writes) != 1 || string(fake.writes[0]) != "pwd" {
		t.Fatalf("expected keyboard input to reach the shell, got %#v", fake.writes)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	m = updated.(Model)
	if m.shellFocused {
		t.Fatal("expected Ctrl+] to return focus to container-ctl")
	}
	if len(fake.writes) != 1 {
		t.Fatal("expected Ctrl+] not to be forwarded to the shell")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = updated.(Model)
	if !m.shellFocused {
		t.Fatal("expected E to focus the existing shell")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	m = updated.(Model)
	if m.embeddedShell != nil || fake.closeCount != 1 {
		t.Fatalf("expected Ctrl+X to close the unfocused shell once")
	}
}

func TestEmbeddedShell_UnfocusedDoesNotInterceptKeysInFilterOrModal(t *testing.T) {
	t.Parallel()

	fake := &fakeEmbeddedShell{}
	m := newFastTestModel()
	m.embeddedShell = fake
	m.shellFocused = false
	m.isFiltering = true

	// User types 'e' while filtering containers
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	if m.shellFocused {
		t.Fatal("expected shell not to steal focus when isFiltering is true")
	}
	if m.filterQuery != "e" {
		t.Fatalf("expected filterQuery to be 'e', got %q", m.filterQuery)
	}

	// User types 'e' inside modal (e.g. exec custom command input)
	m.isFiltering = false
	m.activeModal = ModalExec
	m.execIsCustomInput = true
	m.execCustomCmd = "echo"

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	if m.shellFocused {
		t.Fatal("expected shell not to steal focus when inside modal")
	}
	if m.execCustomCmd != "echoe" {
		t.Fatalf("expected execCustomCmd to receive 'e', got %q", m.execCustomCmd)
	}
}

func TestEmbeddedShellStartFailureDoesNotLaunchFallback(t *testing.T) {
	originalStart := startEmbeddedShell
	defer func() { startEmbeddedShell = originalStart }()

	startEmbeddedShell = func(_ *exec.Cmd, _, _ int) (embeddedShellSession, error) {
		return nil, errors.New("PTY unavailable")
	}
	m := newFastTestModel()
	m.activeModal = ModalExec
	m.execTargetID = "container-id"
	m.execTargetName = "api"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("expected no external fallback command after PTY failure")
	}
	if m.embeddedShell != nil || !strings.Contains(m.actionMessage, "PTY unavailable") {
		t.Fatalf("expected a visible PTY error without an active session, message=%q", m.actionMessage)
	}
}

func TestShellKeyBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  tea.KeyMsg
		want string
	}{
		{name: "runes", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ls")}, want: "ls"},
		{name: "enter", key: tea.KeyMsg{Type: tea.KeyEnter}, want: "\r"},
		{name: "ctrl c", key: tea.KeyMsg{Type: tea.KeyCtrlC}, want: "\x03"},
		{name: "ctrl d", key: tea.KeyMsg{Type: tea.KeyCtrlD}, want: "\x04"},
		{name: "backspace", key: tea.KeyMsg{Type: tea.KeyBackspace}, want: "\x7f"},
		{name: "tab", key: tea.KeyMsg{Type: tea.KeyTab}, want: "\t"},
		{name: "esc", key: tea.KeyMsg{Type: tea.KeyEsc}, want: "\x1b"},
		{name: "up", key: tea.KeyMsg{Type: tea.KeyUp}, want: "\x1b[A"},
		{name: "down", key: tea.KeyMsg{Type: tea.KeyDown}, want: "\x1b[B"},
		{name: "right", key: tea.KeyMsg{Type: tea.KeyRight}, want: "\x1b[C"},
		{name: "left", key: tea.KeyMsg{Type: tea.KeyLeft}, want: "\x1b[D"},
		{name: "shift tab", key: tea.KeyMsg{Type: tea.KeyShiftTab}, want: "\x1b[Z"},
		{name: "alt key", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true}, want: "\x1bb"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := string(shellKeyBytes(tt.key)); got != tt.want {
				t.Fatalf("shellKeyBytes(%s) = %q, want %q", tt.key.String(), got, tt.want)
			}
		})
	}
}

func TestEmbeddedShellOutputAndExitMessagesIgnoreStaleSessions(t *testing.T) {
	t.Parallel()

	active := &fakeEmbeddedShell{}
	stale := &fakeEmbeddedShell{}
	m := newFastTestModel()
	m.embeddedShell = active
	m.shellContainerName = "api"

	updated, cmd := m.Update(embeddedShellOutputMsg{session: stale, data: []byte("stale")})
	m = updated.(Model)
	if cmd != nil || len(active.feed) != 0 {
		t.Fatal("expected stale shell output to be ignored")
	}

	updated, cmd = m.Update(embeddedShellOutputMsg{session: active, data: []byte("ready")})
	m = updated.(Model)
	if cmd == nil || string(active.feed) != "ready" {
		t.Fatalf("expected active output to be rendered and reading to continue")
	}

	updated, _ = m.Update(embeddedShellExitedMsg{session: stale, err: errors.New("stale")})
	m = updated.(Model)
	if m.embeddedShell != active {
		t.Fatal("expected stale exit message not to close the active shell")
	}

	updated, _ = m.Update(embeddedShellExitedMsg{session: active, err: nil})
	m = updated.(Model)
	if m.embeddedShell != nil || active.closeCount != 1 {
		t.Fatal("expected active shell exit to close and clear the session")
	}
}

func TestEmbeddedShell_WindowSizeMsgAndResize(t *testing.T) {
	t.Parallel()

	t.Run("successful resize propagates dimensions", func(t *testing.T) {
		t.Parallel()
		fake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = fake
		m.shellContainerName = "api"

		res, cmd := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd on window resize")
		}
		expectedW, expectedH := embeddedShellSize(160, 50)
		if fake.resizeWidth != expectedW || fake.resizeHeight != expectedH {
			t.Errorf("expected shell resized to (%d, %d), got (%d, %d)",
				expectedW, expectedH, fake.resizeWidth, fake.resizeHeight)
		}
		if updated.width != 160 || updated.height != 50 {
			t.Errorf("expected model dimensions updated to 160x50, got %dx%d", updated.width, updated.height)
		}
	})

	t.Run("resize failure closes shell with toast error", func(t *testing.T) {
		t.Parallel()
		fake := &fakeEmbeddedShell{resizeErr: errors.New("PTY resize failed")}
		m := newFastTestModel()
		m.embeddedShell = fake
		m.shellContainerName = "api"

		res, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		updated := res.(Model)

		if updated.embeddedShell != nil {
			t.Errorf("expected embedded shell to be closed on resize error")
		}
		if !strings.Contains(updated.actionMessage, "Shell error in api: PTY resize failed") {
			t.Errorf("expected error toast, got: %q", updated.actionMessage)
		}
	})
}

func TestEmbeddedShell_FeedErrorHandling(t *testing.T) {
	t.Parallel()

	fake := &fakeEmbeddedShell{feedErr: errors.New("terminal feed failed")}
	m := newFastTestModel()
	m.embeddedShell = fake
	m.shellContainerName = "web"

	res, cmd := m.Update(embeddedShellOutputMsg{session: fake, data: []byte("broken vt sequence")})
	updated := res.(Model)

	if cmd != nil {
		t.Errorf("expected nil cmd when feed fails")
	}
	if updated.embeddedShell != nil {
		t.Errorf("expected shell closed after feed error")
	}
	if !strings.Contains(updated.actionMessage, "Shell error in web: terminal feed failed") {
		t.Errorf("expected toast error message, got: %q", updated.actionMessage)
	}
}

func TestEmbeddedShell_ReadCmdAndErrorHandling(t *testing.T) {
	t.Parallel()

	fake := &fakeEmbeddedShell{readData: []byte("output line\n"), readErr: nil}
	cmd := readEmbeddedShellCmd(fake)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from readEmbeddedShellCmd")
	}
	msg := cmd()
	outMsg, ok := msg.(embeddedShellOutputMsg)
	if !ok {
		t.Fatalf("expected embeddedShellOutputMsg, got %T", msg)
	}
	if string(outMsg.data) != "output line\n" || outMsg.session != fake {
		t.Errorf("unexpected outMsg: %+v", outMsg)
	}

	// Output msg with err stops reading loop
	m := newFastTestModel()
	m.embeddedShell = fake
	res, nextCmd := m.Update(embeddedShellOutputMsg{session: fake, data: nil, err: io.EOF})
	_ = res.(Model)
	if nextCmd != nil {
		t.Errorf("expected read loop to stop (cmd == nil) when msg.err is non-nil")
	}
}

func TestEmbeddedShell_WaitCmdAndExitCodeVariants(t *testing.T) {
	t.Parallel()

	fake := &fakeEmbeddedShell{}
	cmd := waitEmbeddedShellCmd(fake)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from waitEmbeddedShellCmd")
	}
	msg := cmd()
	exitMsg, ok := msg.(embeddedShellExitedMsg)
	if !ok {
		t.Fatalf("expected embeddedShellExitedMsg, got %T", msg)
	}
	if exitMsg.session != fake {
		t.Errorf("unexpected session in exitMsg")
	}

	t.Run("exit code 0 (clean exit)", func(t *testing.T) {
		t.Parallel()
		subFake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = subFake
		m.shellContainerName = "api"

		res, nextCmd := m.Update(embeddedShellExitedMsg{session: subFake, err: nil})
		updated := res.(Model)

		if updated.embeddedShell != nil {
			t.Errorf("expected shell closed")
		}
		if !strings.Contains(updated.actionMessage, "Shell session in api ended") {
			t.Errorf("expected clean exit toast, got %q", updated.actionMessage)
		}
		if nextCmd == nil {
			t.Errorf("expected fetchDataCmd to refresh dashboard")
		}
	})

	t.Run("exit code 130 (SIGINT / Ctrl+C)", func(t *testing.T) {
		t.Parallel()
		exit130 := exec.Command("bash", "-c", "exit 130").Run()
		if exit130 == nil {
			t.Skip("unable to produce exit code 130")
		}

		subFake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = subFake
		m.shellContainerName = "api"

		res, _ := m.Update(embeddedShellExitedMsg{session: subFake, err: exit130})
		updated := res.(Model)

		if !strings.Contains(updated.actionMessage, "Shell session in api ended") ||
			strings.Contains(updated.actionMessage, "code") {
			t.Errorf("expected clean exit toast without code for 130, got: %q", updated.actionMessage)
		}
	})

	t.Run("exit code 1 (general error exit)", func(t *testing.T) {
		t.Parallel()
		exit1 := exec.Command("bash", "-c", "exit 1").Run()
		if exit1 == nil {
			t.Skip("unable to produce exit code 1")
		}

		subFake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = subFake
		m.shellContainerName = "api"

		res, _ := m.Update(embeddedShellExitedMsg{session: subFake, err: exit1})
		updated := res.(Model)

		if !strings.Contains(updated.actionMessage, "Shell session in api ended (code 1)") {
			t.Errorf("expected code 1 toast, got: %q", updated.actionMessage)
		}
	})

	t.Run("generic non-ExitError error", func(t *testing.T) {
		t.Parallel()
		subFake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = subFake
		m.shellContainerName = "api"

		res, _ := m.Update(embeddedShellExitedMsg{session: subFake, err: errors.New("PTY read error")})
		updated := res.(Model)

		if !strings.Contains(updated.actionMessage, "Shell error in api: PTY read error") {
			t.Errorf("expected error toast, got: %q", updated.actionMessage)
		}
	})
}

func TestEmbeddedShell_KeyHandlingEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("write error closes shell with toast", func(t *testing.T) {
		t.Parallel()
		fake := &fakeEmbeddedShell{writeErr: errors.New("PTY write broken")}
		m := newFastTestModel()
		m.embeddedShell = fake
		m.shellFocused = true
		m.shellContainerName = "web"

		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		updated := res.(Model)

		if updated.embeddedShell != nil {
			t.Errorf("expected shell closed after write error")
		}
		if !strings.Contains(updated.actionMessage, "Shell error in web: PTY write broken") {
			t.Errorf("expected write error toast, got %q", updated.actionMessage)
		}
	})

	t.Run("unhandled key does not write to shell", func(t *testing.T) {
		t.Parallel()
		fake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = fake
		m.shellFocused = true

		// KeyF12 is unmapped in shellKeyBytes
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF12})
		_ = res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd for unhandled key")
		}
		if len(fake.writes) != 0 {
			t.Errorf("expected no writes for unhandled key, got: %v", fake.writes)
		}
	})

	t.Run("capital 'E' focuses shell when unfocused", func(t *testing.T) {
		t.Parallel()
		fake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = fake
		m.shellFocused = false

		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		updated := res.(Model)
		if !updated.shellFocused {
			t.Errorf("expected shell focused after pressing 'E'")
		}
	})

	t.Run("Ctrl+X closes shell when focused", func(t *testing.T) {
		t.Parallel()
		fake := &fakeEmbeddedShell{}
		m := newFastTestModel()
		m.embeddedShell = fake
		m.shellFocused = true
		m.shellContainerName = "alpine"

		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
		updated := res.(Model)
		if updated.embeddedShell != nil {
			t.Errorf("expected embedded shell to be nil after Ctrl+X")
		}
		if updated.shellFocused {
			t.Errorf("expected shellFocused to be false after Ctrl+X")
		}
		if cmd == nil {
			t.Errorf("expected fetchDataCmd after closing shell")
		}
		if !strings.Contains(updated.actionMessage, "Shell session in alpine closed") {
			t.Errorf("expected close toast message, got: %q", updated.actionMessage)
		}
	})
}

func TestViewRendersEmbeddedShellAndFocusHelp(t *testing.T) {
	t.Parallel()

	t.Run("wide layout with focused shell", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width, m.height = 140, 40
		m.embeddedShell = &fakeEmbeddedShell{render: "root@api:/app# pwd"}
		m.shellContainerName = "api"
		m.shellFocused = true

		view := m.View()
		for _, want := range []string{
			"TERMINAL: api",
			"root@api:/app# pwd",
			"Ctrl+]",
			"Return to container-ctl",
			"Ctrl+D",
			"Exit shell",
		} {
			if !strings.Contains(view, want) {
				t.Fatalf("expected embedded shell view to contain %q, got:\n%s", want, view)
			}
		}
	})

	t.Run("narrow layout renders terminal panel", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width, m.height = 90, 30
		m.embeddedShell = &fakeEmbeddedShell{render: "compact terminal"}
		m.shellContainerName = "db"
		m.shellFocused = true

		view := m.View()
		if !strings.Contains(view, "compact terminal") {
			t.Errorf("expected compact terminal output in narrow view")
		}
	})

	t.Run("unfocused shell renders focus hint in footer", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width, m.height = 120, 35
		m.embeddedShell = &fakeEmbeddedShell{render: "idle shell"}
		m.shellContainerName = "web"
		m.shellFocused = false

		view := m.View()
		if !strings.Contains(view, "Focus terminal") {
			t.Errorf("expected 'Focus terminal' hint in footer when unfocused")
		}
	})
}
