package modals

import (
	"strings"
	"testing"
)

func TestRenderLogs(t *testing.T) {
	// Empty logs
	emptyView := RenderLogs(nil, 0, "test-container", "c12345", 100)
	if !strings.Contains(emptyView, "test-container") || !strings.Contains(emptyView, "No logs output available") {
		t.Errorf("expected container name and no logs message, got: %s", emptyView)
	}

	// Logs with scrolling
	lines := []string{"log line 1", "log line 2", "log line 3", "log line 4"}
	view := RenderLogs(lines, 1, "test-container", "c12345", 80)
	if !strings.Contains(view, "log line 2") {
		t.Errorf("expected 'log line 2' in view, got: %s", view)
	}
	if !strings.Contains(view, "4 lines") && !strings.Contains(view, "4") {
		t.Errorf("expected total lines count in view, got: %s", view)
	}
}

func TestCalculateLogRange(t *testing.T) {
	// Zero lines
	s, e := calculateLogRange(0, 0, 18)
	if s != 0 || e != 0 {
		t.Errorf("expected 0, 0 for empty lines, got %d, %d", s, e)
	}

	// Negative scroll clamped to 0
	s, e = calculateLogRange(50, -5, 18)
	if s != 0 || e != 18 {
		t.Errorf("expected 0, 18 for negative scroll, got %d, %d", s, e)
	}

	// Scroll beyond line count clamped to max
	s, e = calculateLogRange(10, 50, 18)
	if s != 9 || e != 10 {
		t.Errorf("expected 9, 10 for excessive scroll, got %d, %d", s, e)
	}

	// Normal middle range
	s, e = calculateLogRange(100, 20, 18)
	if s != 20 || e != 38 {
		t.Errorf("expected 20, 38, got %d, %d", s, e)
	}
}

func TestGetLogGutterStyle(t *testing.T) {
	errStyle := getLogGutterStyle("2026-11-07 ERROR: connection refused")
	if errStyle.GetForeground() == nil {
		t.Errorf("expected colored style for ERROR line")
	}

	warnStyle := getLogGutterStyle("2026-11-07 [WARN] slow response detected")
	if warnStyle.GetForeground() == nil {
		t.Errorf("expected colored style for WARN line")
	}

	infoStyle := getLogGutterStyle("2026-11-07 info: server listening on port 8080")
	if infoStyle.GetForeground() == nil {
		t.Errorf("expected styled output for normal line")
	}
}

func TestRenderLogs_EdgeCases(t *testing.T) {
	lines := []string{
		"INFO started",
		"WARN slow query",
		"ERROR db down",
	}

	// Negative scroll should not panic and render safely
	negView := RenderLogs(lines, -10, "app", "id1", 60)
	if !strings.Contains(negView, "app") || !strings.Contains(negView, "INFO started") {
		t.Errorf("expected safe render with negative scroll, got: %s", negView)
	}

	// Excessive scroll should not panic
	bigScrollView := RenderLogs(lines, 100, "app", "id1", 60)
	if !strings.Contains(bigScrollView, "ERROR db down") {
		t.Errorf("expected last line on excessive scroll, got: %s", bigScrollView)
	}
}

func TestRenderLogs_Filter(t *testing.T) {
	lines := []string{
		"2026-11-20 INFO starting server on port 8080",
		"2026-11-20 ERROR failed to connect to database",
		"2026-11-20 WARN retrying in 5 seconds",
		"2026-11-20 INFO heartbeat ok",
	}

	// Filter query for ERROR
	filteredView := RenderLogs(lines, 0, "api-srv", "c999", 80, "ERROR", "false")
	if !strings.Contains(filteredView, "failed to connect to database") {
		t.Errorf("expected error line in filtered logs, got: %s", filteredView)
	}
	if strings.Contains(filteredView, "starting server") {
		t.Errorf("did not expect info line when filtered by ERROR, got: %s", filteredView)
	}
	if !strings.Contains(filteredView, "1 matches") {
		t.Errorf("expected match count in filtered logs view, got: %s", filteredView)
	}

	// Active search input
	activeSearchView := RenderLogs(lines, 0, "api-srv", "c999", 80, "WARN", "true")
	if !strings.Contains(activeSearchView, "Search:") || !strings.Contains(activeSearchView, "WARN") {
		t.Errorf("expected active search bar in view, got: %s", activeSearchView)
	}

	// No matching filter query
	noMatchView := RenderLogs(lines, 0, "api-srv", "c999", 80, "FATAL", "false")
	if !strings.Contains(noMatchView, "No log lines match filter") {
		t.Errorf("expected no log lines match message, got: %s", noMatchView)
	}
}
