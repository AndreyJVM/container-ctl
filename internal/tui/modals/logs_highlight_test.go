package modals

import (
	"regexp"
	"strings"
	"testing"
)

func TestCompileLogMatcher(t *testing.T) {
	// Empty query
	matcher, re, isRegex := CompileLogMatcher("")
	if !matcher("any string") || re != nil || isRegex {
		t.Errorf("expected empty query to match everything and have nil regex")
	}

	// Valid regex
	matcher, re, isRegex = CompileLogMatcher(`err(or)?:\s*\d+`)
	if !isRegex || re == nil {
		t.Fatalf("expected valid regex compilation")
	}
	if !matcher("Error: 500 internal server error") {
		t.Errorf("expected regex match")
	}
	if matcher("Notice: ok") {
		t.Errorf("did not expect match")
	}

	// Invalid regex syntax -> graceful fallback to literal match
	matcher, re, isRegex = CompileLogMatcher(`[incomplete-regex`)
	if isRegex || re != nil {
		t.Errorf("expected isRegex=false for invalid regex")
	}
	if !matcher("something with [incomplete-regex inside") {
		t.Errorf("expected literal fallback match")
	}
	if matcher("no match here") {
		t.Errorf("did not expect match on different text")
	}
}

func TestHighlightLogLine_Syntax(t *testing.T) {
	// Empty line
	if out := HighlightLogLine("", nil, ""); out != "" {
		t.Errorf("expected empty string, got: %q", out)
	}

	// Line with timestamp, log level, and HTTP method
	line := "2026-11-22T14:35:34Z ERROR GET /api/v1/users failed"
	out := HighlightLogLine(line, nil, "")
	if !strings.Contains(out, "2026-11-22T14:35:34Z") {
		t.Errorf("expected timestamp preserved in output")
	}
	if !strings.Contains(out, "ERROR") || !strings.Contains(out, "GET") {
		t.Errorf("expected keywords preserved in output")
	}

	// WARN and INFO and DEBUG tokens
	warnLine := HighlightLogLine("WARN database connection high latency", nil, "")
	if !strings.Contains(warnLine, "WARN") {
		t.Errorf("expected WARN token preserved")
	}

	infoLine := HighlightLogLine("INFO server listening on :8080", nil, "")
	if !strings.Contains(infoLine, "INFO") {
		t.Errorf("expected INFO token preserved")
	}

	debugLine := HighlightLogLine("DEBUG payload dump: test", nil, "")
	if !strings.Contains(debugLine, "DEBUG") {
		t.Errorf("expected DEBUG token preserved")
	}
}

func TestHighlightLogLine_SearchHighlight(t *testing.T) {
	line := "Connection timeout after 30s to backend-service (timeout)"

	// Regex match highlighting
	re := regexp.MustCompile(`(?i)timeout`)
	out := HighlightLogLine(line, re, "timeout")
	if !strings.Contains(out, "timeout") {
		t.Errorf("expected highlighted output to contain 'timeout'")
	}

	// Literal match highlighting (when regex is nil)
	outLit := HighlightLogLine(line, nil, "backend")
	if !strings.Contains(outLit, "backend") {
		t.Errorf("expected literal highlighted output to contain 'backend'")
	}

	// Edge case: empty query
	outPlain := HighlightLogLine(line, nil, "")
	if !strings.Contains(outPlain, "Connection") {
		t.Errorf("expected plain output without query to preserve text")
	}
}
