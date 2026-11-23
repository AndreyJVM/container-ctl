package clipboard

import (
	"bytes"
	"strings"
	"testing"
)

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
