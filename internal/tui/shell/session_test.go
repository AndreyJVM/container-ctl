package shell

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

type fakePty struct {
	bytes.Buffer
	readData    []byte
	readErr     error
	startErr    error
	started     bool
	closedCount int
	resizeErr   error
	width       int
	height      int
}

func (p *fakePty) Read(b []byte) (int, error) {
	if len(p.readData) > 0 {
		n := copy(b, p.readData)
		p.readData = p.readData[n:]
		return n, nil
	}
	if p.readErr != nil {
		return 0, p.readErr
	}
	return 0, io.EOF
}

func (p *fakePty) Close() error {
	p.closedCount++
	return nil
}

func (p *fakePty) Resize(width, height int) error {
	p.width, p.height = width, height
	return p.resizeErr
}

func (p *fakePty) Start(_ *exec.Cmd) error {
	p.started = true
	return p.startErr
}

type fakeTerminal struct {
	output      bytes.Buffer
	writeErr    error
	closedCount int
	width       int
	height      int
}

func (t *fakeTerminal) Read(_ []byte) (int, error) { return 0, io.EOF }

func (t *fakeTerminal) Write(data []byte) (int, error) {
	if t.writeErr != nil {
		return 0, t.writeErr
	}
	return t.output.Write(data)
}

func (t *fakeTerminal) Close() error {
	t.closedCount++
	return nil
}

func (t *fakeTerminal) Render() string { return t.output.String() }

func (t *fakeTerminal) Resize(width, height int) {
	t.width, t.height = width, height
}

func TestStart_ValidationAndResourceCleanup(t *testing.T) {
	t.Parallel()

	t.Run("rejects nil command before opening PTY", func(t *testing.T) {
		t.Parallel()
		ptyCreated := false
		_, err := startWithFactories(nil, 80, 20,
			func(_, _ int) (ptyBackend, error) {
				ptyCreated = true
				return &fakePty{}, nil
			},
			func(_, _ int) terminalBackend { return &fakeTerminal{} },
		)

		if err == nil {
			t.Fatal("expected nil command to be rejected")
		}
		if ptyCreated {
			t.Fatal("expected no PTY allocation for a nil command")
		}
	})

	t.Run("fails when PTY creation fails", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("cannot create pty device")
		terminalCreated := false

		_, err := startWithFactories(exec.Command("ignored"), 80, 20,
			func(_, _ int) (ptyBackend, error) { return nil, expectedErr },
			func(_, _ int) terminalBackend {
				terminalCreated = true
				return &fakeTerminal{}
			},
		)

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected PTY creation error %v, got: %v", expectedErr, err)
		}
		if terminalCreated {
			t.Fatal("expected virtual terminal factory not called when PTY creation fails")
		}
	})

	t.Run("fails when terminal creation returns nil and cleans up PTY", func(t *testing.T) {
		t.Parallel()
		pty := &fakePty{}
		_, err := startWithFactories(exec.Command("ignored"), 80, 20,
			func(_, _ int) (ptyBackend, error) { return pty, nil },
			func(_, _ int) terminalBackend { return nil },
		)

		if err == nil || !strings.Contains(err.Error(), "empty terminal") {
			t.Fatalf("expected empty terminal error, got: %v", err)
		}
		if pty.closedCount != 1 {
			t.Fatalf("expected allocated PTY to be closed when terminal is empty, got closedCount=%d", pty.closedCount)
		}
	})

	t.Run("start failure closes both allocated PTY and terminal", func(t *testing.T) {
		t.Parallel()
		fakeErr := errors.New("exec binary not found in path")
		pty := &fakePty{startErr: fakeErr}
		terminal := &fakeTerminal{}

		_, err := startWithFactories(exec.Command("ignored"), 80, 20,
			func(_, _ int) (ptyBackend, error) { return pty, nil },
			func(_, _ int) terminalBackend { return terminal },
		)

		if !errors.Is(err, fakeErr) {
			t.Fatalf("expected start error %v, got %v", fakeErr, err)
		}
		if pty.closedCount != 1 || terminal.closedCount != 1 {
			t.Fatalf("expected allocated resources to close once, PTY=%d terminal=%d", pty.closedCount, terminal.closedCount)
		}
	})
}

func TestSession_Read(t *testing.T) {
	t.Parallel()

	t.Run("reads available output from PTY", func(t *testing.T) {
		t.Parallel()
		pty := &fakePty{readData: []byte("bash-5.2$ ")}
		session, err := startWithFactories(exec.Command("ignored"), 80, 20,
			func(_, _ int) (ptyBackend, error) { return pty, nil },
			func(_, _ int) terminalBackend { return &fakeTerminal{} },
		)
		if err != nil {
			t.Fatalf("startWithFactories failed: %v", err)
		}

		data, err := session.Read()
		if err != nil {
			t.Fatalf("expected clean read, got: %v", err)
		}
		if string(data) != "bash-5.2$ " {
			t.Errorf("expected %q, got %q", "bash-5.2$ ", string(data))
		}
	})

	t.Run("propagates read error when no bytes available", func(t *testing.T) {
		t.Parallel()
		readErr := errors.New("pty I/O error")
		pty := &fakePty{readErr: readErr}
		session, err := startWithFactories(exec.Command("ignored"), 80, 20,
			func(_, _ int) (ptyBackend, error) { return pty, nil },
			func(_, _ int) terminalBackend { return &fakeTerminal{} },
		)
		if err != nil {
			t.Fatalf("startWithFactories failed: %v", err)
		}

		data, err := session.Read()
		if !errors.Is(err, readErr) {
			t.Fatalf("expected read error %v, got: %v", readErr, err)
		}
		if data != nil {
			t.Errorf("expected nil data on error, got: %v", data)
		}
	})
}

func TestSession_InputOutputResizeAndNormalize(t *testing.T) {
	t.Parallel()

	pty := &fakePty{}
	terminal := &fakeTerminal{}
	session, err := startWithFactories(exec.Command("ignored"), 80, 20,
		func(_, _ int) (ptyBackend, error) { return pty, nil },
		func(_, _ int) terminalBackend { return terminal },
	)
	if err != nil {
		t.Fatalf("startWithFactories() returned an error: %v", err)
	}

	// 1. Write sends input to PTY
	if err := session.Write([]byte("ls -la\r")); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if got := pty.String(); got != "ls -la\r" {
		t.Errorf("expected PTY input 'ls -la\\r', got: %q", got)
	}

	// 2. Feed applies output to Virtual Terminal
	if err := session.Feed([]byte("total 32\r\n")); err != nil {
		t.Fatalf("Feed() error: %v", err)
	}
	if got := session.Render(); got != "total 32\r\n" {
		t.Errorf("expected virtual terminal render 'total 32\\r\\n', got: %q", got)
	}

	// 3. Resize updates PTY and Virtual Terminal dimensions
	if err := session.Resize(100, 30); err != nil {
		t.Fatalf("Resize() error: %v", err)
	}
	if pty.width != 100 || pty.height != 30 || terminal.width != 100 || terminal.height != 30 {
		t.Errorf("expected both PTY and terminal to resize to 100x30, got PTY=%dx%d terminal=%dx%d",
			pty.width, pty.height, terminal.width, terminal.height)
	}

	// 4. Resize normalizes negative/zero dimensions to at least 1x1
	if err := session.Resize(0, -10); err != nil {
		t.Fatalf("Resize() error: %v", err)
	}
	if pty.width != 1 || pty.height != 1 || terminal.width != 1 || terminal.height != 1 {
		t.Errorf("expected normalized dimensions 1x1, got PTY=%dx%d terminal=%dx%d",
			pty.width, pty.height, terminal.width, terminal.height)
	}

	// 5. Resize propagates PTY error
	pty.resizeErr = errors.New("pty resize syscall failed")
	if err := session.Resize(120, 40); err == nil || !strings.Contains(err.Error(), "pty resize syscall failed") {
		t.Errorf("expected wrapped pty resize error, got: %v", err)
	}
}

func TestSession_ClosedOperationsAndIdempotence(t *testing.T) {
	t.Parallel()

	pty := &fakePty{}
	terminal := &fakeTerminal{}
	session, err := startWithFactories(exec.Command("ignored"), 80, 20,
		func(_, _ int) (ptyBackend, error) { return pty, nil },
		func(_, _ int) terminalBackend { return terminal },
	)
	if err != nil {
		t.Fatalf("startWithFactories failed: %v", err)
	}

	// Initial Close
	if err := session.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}
	// Second Close (idempotent)
	if err := session.Close(); err != nil {
		t.Fatalf("second Close() error: %v", err)
	}
	if pty.closedCount != 1 || terminal.closedCount != 1 {
		t.Fatalf("expected resources closed only once, PTY=%d terminal=%d", pty.closedCount, terminal.closedCount)
	}

	// Operations after Close must return io.ErrClosedPipe
	if err := session.Write([]byte("test")); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("expected io.ErrClosedPipe from Write after Close, got: %v", err)
	}
	if err := session.Feed([]byte("test")); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("expected io.ErrClosedPipe from Feed after Close, got: %v", err)
	}
	if err := session.Resize(80, 24); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("expected io.ErrClosedPipe from Resize after Close, got: %v", err)
	}
}

func TestSession_LiveProcessExecutionAndTermination(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh is not available on host system")
	}

	t.Run("runs command, reads output, feeds emulator and waits for exit", func(t *testing.T) {
		session, err := Start(exec.Command(sh, "-c", "printf 'pty-ready-marker\\r\\n'"), 40, 8)
		if err != nil {
			t.Fatalf("Start() error: %v", err)
		}
		defer session.Close()

		data, readErr := session.Read()
		if len(data) == 0 {
			t.Fatalf("expected PTY output, read error: %v", readErr)
		}
		if err := session.Feed(data); err != nil {
			t.Fatalf("Feed() error: %v", err)
		}
		if got := session.Render(); !strings.Contains(got, "pty-ready-marker") {
			t.Fatalf("expected rendered PTY output to contain 'pty-ready-marker', got %q", got)
		}
		if err := session.Wait(); err != nil {
			t.Fatalf("Wait() error: %v", err)
		}
	})

	t.Run("close safely terminates active running child process", func(t *testing.T) {
		// Run long-running process
		cmd := exec.Command(sh, "-c", "sleep 30")
		session, err := Start(cmd, 40, 8)
		if err != nil {
			t.Fatalf("Start() error: %v", err)
		}

		// Close should kill the process without error
		if err := session.Close(); err != nil {
			t.Fatalf("Close() error while killing active process: %v", err)
		}
		// Second close should be safe
		if err := session.Close(); err != nil {
			t.Fatalf("second Close() error: %v", err)
		}
	})
}

func TestSession_TerminalQueries(t *testing.T) {
	t.Parallel()

	t.Run("responds to cursor position report CPR \\x1b[6n", func(t *testing.T) {
		t.Parallel()
		fakePty := &fakePty{readData: []byte("/ # \x1b[6n")}
		session := &Session{pty: fakePty, terminal: &fakeTerminal{}}

		data, err := session.Read()
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		if !bytes.Contains(data, []byte("\x1b[6n")) {
			t.Fatalf("expected data to contain escape sequence, got: %q", string(data))
		}
		written := fakePty.String()
		if !strings.Contains(written, "\x1b[1;1R") {
			t.Errorf("expected PTY response '\\x1b[1;1R' for CPR, got written: %q", written)
		}
	})

	t.Run("responds to operating status DSR \\x1b[5n", func(t *testing.T) {
		t.Parallel()
		fakePty := &fakePty{readData: []byte("\x1b[5n")}
		session := &Session{pty: fakePty, terminal: &fakeTerminal{}}

		_, err := session.Read()
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		written := fakePty.String()
		if !strings.Contains(written, "\x1b[0n") {
			t.Errorf("expected PTY response '\\x1b[0n' for DSR, got written: %q", written)
		}
	})

	t.Run("responds to device attributes DA \\x1b[0c", func(t *testing.T) {
		t.Parallel()
		fakePty := &fakePty{readData: []byte("\x1b[0c")}
		session := &Session{pty: fakePty, terminal: &fakeTerminal{}}

		_, err := session.Read()
		if err != nil {
			t.Fatalf("unexpected read error: %v", err)
		}
		written := fakePty.String()
		if !strings.Contains(written, "\x1b[?1;2c") {
			t.Errorf("expected PTY response '\\x1b[?1;2c' for DA, got written: %q", written)
		}
	})
}
