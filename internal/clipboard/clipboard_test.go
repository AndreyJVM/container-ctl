package clipboard

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/aymanbagabas/go-osc52/v2"
)

type failWriter struct{}

func (f failWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write failed")
}

func TestClipboardWrite(t *testing.T) {
	orig := GetOutputWriter()
	defer SetOutputWriter(orig)

	var buf bytes.Buffer
	SetOutputWriter(&buf)

	err := Write("docker exec -it my-app sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "\x1b]52;c;") {
		t.Errorf("expected OSC 52 clipboard prefix, got: %q", out)
	}

	// Empty text should not emit sequence
	buf.Reset()
	if err := Write(""); err != nil {
		t.Fatalf("unexpected error on empty: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer on empty write, got: %q", buf.String())
	}
}

func TestTestLock(t *testing.T) {
	unlock := TestLock()
	if unlock == nil {
		t.Errorf("expected non-nil unlock function")
	}
	unlock()
}

func TestClipboardInstance(t *testing.T) {
	var buf bytes.Buffer
	cb := New(&buf)

	if cb.GetWriter() != &buf {
		t.Errorf("expected writer &buf, got: %v", cb.GetWriter())
	}

	// Write system clipboard
	if err := cb.Write("hello world"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "\x1b]52;c;") {
		t.Errorf("expected OSC 52 clipboard sequence, got: %q", out)
	}

	// Write primary selection
	buf.Reset()
	if err := cb.WritePrimary("primary text"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out = buf.String()
	if !strings.HasPrefix(out, "\x1b]52;p;") {
		t.Errorf("expected OSC 52 primary selection sequence, got: %q", out)
	}

	// WriteAll
	buf.Reset()
	if err := cb.WriteAll("both"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, "\x1b]52;c;") || !strings.Contains(out, "\x1b]52;p;") {
		t.Errorf("expected both clipboard and primary sequences, got: %q", out)
	}

	// Empty primary write
	buf.Reset()
	if err := cb.WritePrimary(""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty buffer on empty primary write, got: %q", buf.String())
	}
}

func TestNilSafety(t *testing.T) {
	cb := New(nil)
	if cb.GetWriter() != io.Discard {
		t.Errorf("expected io.Discard for nil writer, got: %v", cb.GetWriter())
	}

	// Should not panic
	if err := cb.Write("test safe write"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cb.SetWriter(nil)
	if cb.GetWriter() != io.Discard {
		t.Errorf("expected io.Discard after SetWriter(nil)")
	}
	if err := cb.WritePrimary("test safe primary"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMaxSizeLimit(t *testing.T) {
	var buf bytes.Buffer
	cb := New(&buf)
	cb.SetMaxSize(10)

	if cb.GetMaxSize() != 10 {
		t.Errorf("expected max size 10, got: %d", cb.GetMaxSize())
	}

	// Fits in limit
	if err := cb.Write("1234567890"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Exceeds limit
	err := cb.Write("12345678901")
	if !errors.Is(err, ErrTextTooLarge) {
		t.Errorf("expected ErrTextTooLarge, got: %v", err)
	}

	// Primary exceeds limit
	err = cb.WritePrimary("12345678901")
	if !errors.Is(err, ErrTextTooLarge) {
		t.Errorf("expected ErrTextTooLarge, got: %v", err)
	}
}

func TestWriterError(t *testing.T) {
	cb := New(failWriter{})
	if err := cb.Write("fail"); err == nil {
		t.Errorf("expected error from failWriter, got nil")
	}
	if err := cb.WritePrimary("fail"); err == nil {
		t.Errorf("expected error from failWriter on primary, got nil")
	}
	if err := cb.WriteAll("fail"); err == nil {
		t.Errorf("expected error from failWriter on WriteAll, got nil")
	}
}

func TestModesAndAutoDetect(t *testing.T) {
	var buf bytes.Buffer
	cb := New(&buf)

	cb.SetMode(osc52.TmuxMode)
	if cb.GetMode() != osc52.TmuxMode {
		t.Errorf("expected TmuxMode, got: %v", cb.GetMode())
	}

	if err := cb.Write("tmux-test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "\x1bPtmux;\x1b\x1b]52;") {
		t.Errorf("expected tmux escape sequence prefix, got: %q", buf.String())
	}

	// Test AutoDetect with TMUX
	t.Setenv("TMUX", "1")
	cb.AutoDetect()
	if cb.GetMode() != osc52.TmuxMode {
		t.Errorf("expected TmuxMode via AutoDetect with TMUX=1")
	}

	// Test AutoDetect with Screen
	os.Unsetenv("TMUX")
	t.Setenv("TERM", "screen-256color")
	cb.AutoDetect()
	if cb.GetMode() != osc52.ScreenMode {
		t.Errorf("expected ScreenMode via AutoDetect with TERM=screen*")
	}

	// Test AutoDetect default
	t.Setenv("TERM", "xterm-256color")
	cb.AutoDetect()
	if cb.GetMode() != osc52.DefaultMode {
		t.Errorf("expected DefaultMode via AutoDetect")
	}
}

func TestConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	cb := New(&buf)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = cb.Write("concurrent write")
			_ = cb.WritePrimary("concurrent primary")
			_ = cb.GetWriter()
			_ = cb.GetMaxSize()
			_ = cb.GetMode()
		}(i)
	}
	wg.Wait()
}

func TestPackageLevelHelpers(t *testing.T) {
	defer TestLock()()
	orig := GetOutputWriter()
	defer SetOutputWriter(orig)

	var buf bytes.Buffer
	SetOutputWriter(&buf)

	if Default() == nil {
		t.Fatalf("expected non-nil Default()")
	}

	if err := WritePrimary("primary-pkg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "\x1b]52;p;") {
		t.Errorf("expected primary sequence in package write")
	}

	buf.Reset()
	if err := WriteAll("all-pkg"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "\x1b]52;c;") || !strings.Contains(buf.String(), "\x1b]52;p;") {
		t.Errorf("expected both sequences in package WriteAll")
	}
}
