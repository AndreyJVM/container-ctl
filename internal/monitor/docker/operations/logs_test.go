package operations

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestGetContainerLogs_EmptyID(t *testing.T) {
	_, err := GetContainerLogs("   ", 10)
	if err == nil {
		t.Errorf("expected error for empty container ID")
	}
}

func TestDemuxDockerLogs(t *testing.T) {
	t.Run("plain text without headers", func(t *testing.T) {
		plain := "Hello world log output"
		res := DemuxDockerLogs(strings.NewReader(plain))
		if res != plain {
			t.Errorf("expected '%s', got '%s'", plain, res)
		}
	})

	t.Run("multiplexed stdout and stderr frames", func(t *testing.T) {
		var buf bytes.Buffer

		// Frame 1: stdout (streamType 1)
		stdoutMsg := "Line from stdout\n"
		var hdr1 [8]byte
		hdr1[0] = 1 // stdout
		binary.BigEndian.PutUint32(hdr1[4:8], uint32(len(stdoutMsg)))
		buf.Write(hdr1[:])
		buf.WriteString(stdoutMsg)

		// Frame 2: stderr (streamType 2)
		stderrMsg := "Line from stderr\n"
		var hdr2 [8]byte
		hdr2[0] = 2 // stderr
		binary.BigEndian.PutUint32(hdr2[4:8], uint32(len(stderrMsg)))
		buf.Write(hdr2[:])
		buf.WriteString(stderrMsg)

		expected := stdoutMsg + stderrMsg
		res := DemuxDockerLogs(&buf)
		if res != expected {
			t.Errorf("expected demuxed %q, got %q", expected, res)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if res := DemuxDockerLogs(strings.NewReader("")); res != "" {
			t.Errorf("expected empty string for empty input, got %q", res)
		}
	})

	t.Run("multiplexed stream with zero-length payload frame", func(t *testing.T) {
		var buf bytes.Buffer
		var hdr [8]byte
		hdr[0] = 1 // stdout
		binary.BigEndian.PutUint32(hdr[4:8], 0)
		buf.Write(hdr[:])

		res := DemuxDockerLogs(&buf)
		if res != "" {
			t.Errorf("expected empty string for empty frame payload, got %q", res)
		}
	})

	t.Run("plain text shorter than 8 byte header", func(t *testing.T) {
		shortText := "info"
		res := DemuxDockerLogs(strings.NewReader(shortText))
		if res != shortText {
			t.Errorf("expected %q, got %q", shortText, res)
		}
	})
}

func TestCleanLogSnippet(t *testing.T) {
	// Empty input
	if got := CleanLogSnippet("", 3); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
	if got := CleanLogSnippet("  \n\n  ", 3); got != "" {
		t.Errorf("expected empty string for whitespace, got %q", got)
	}

	// Fewer lines than max
	input := "line 1\nline 2"
	if got := CleanLogSnippet(input, 3); got != "line 1\nline 2" {
		t.Errorf("unexpected snippet: %q", got)
	}

	// More lines than max
	inputLong := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	expected := "line 3\nline 4\nline 5"
	if got := CleanLogSnippet(inputLong, 3); got != expected {
		t.Errorf("CleanLogSnippet = %q, want %q", got, expected)
	}

	// Default fallback when maxLines <= 0
	if got := CleanLogSnippet(inputLong, 0); got != expected {
		t.Errorf("CleanLogSnippet with 0 maxLines = %q, want %q", got, expected)
	}
}
