package modals

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
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

func TestRenderInspect(t *testing.T) {
	detail := monitor.ContainerDetail{
		ID:              "c123456789",
		Name:            "web-service",
		Image:           "nginx:alpine",
		State:           "running",
		Status:          "Up 3 hours",
		Health:          "healthy",
		CPUPerc:         4.5,
		MemUsageMB:      120.0,
		MemPerc:         3.2,
		Created:         "2026-11-06",
		IP:              "172.17.0.2",
		Gateway:         "172.17.0.1",
		Command:         "nginx -g daemon off;",
		Mounts:          []string{"/host/data -> /data (ro)"},
		NetRxBytes:      5 * 1024 * 1024,
		NetTxBytes:      10 * 1024 * 1024,
		BlockReadBytes:  2 * 1024 * 1024,
		BlockWriteBytes: 4 * 1024 * 1024,
		RestartPolicy:   "unless-stopped",
	}

	view := RenderInspect(detail)
	if !strings.Contains(view, "web-service") || !strings.Contains(view, "[healthy]") {
		t.Errorf("expected container name and healthy badge, got: %s", view)
	}
	if !strings.Contains(view, "172.17.0.2") || !strings.Contains(view, "/host/data") {
		t.Errorf("expected IP address and mounts in inspect view, got: %s", view)
	}
	if !strings.Contains(view, "Net I/O:") || !strings.Contains(view, "5.0 MB / 10.0 MB") {
		t.Errorf("expected Net I/O in inspect view, got: %s", view)
	}
	if !strings.Contains(view, "Block I/O:") || !strings.Contains(view, "2.0 MB / 4.0 MB") {
		t.Errorf("expected Block I/O in inspect view, got: %s", view)
	}
	if !strings.Contains(view, "unless-stopped") || !strings.Contains(view, "Restart Policy:") {
		t.Errorf("expected Restart Policy in inspect view, got: %s", view)
	}

	// Environment rendering with masked secrets
	detailWithEnv := detail
	detailWithEnv.Env = []string{
		"APP_ENV=production",
		"DB_PASSWORD=supersecret",
		"API_KEY=abcdef123456",
		"PORT=8080",
	}
	envView := RenderInspect(detailWithEnv)
	if !strings.Contains(envView, "APP_ENV=production") || !strings.Contains(envView, "PORT=8080") {
		t.Errorf("expected regular env vars rendered, got: %s", envView)
	}
	if !strings.Contains(envView, "DB_PASSWORD=******") || strings.Contains(envView, "supersecret") {
		t.Errorf("expected DB_PASSWORD to be masked, got: %s", envView)
	}
	if !strings.Contains(envView, "API_KEY=******") || strings.Contains(envView, "abcdef123456") {
		t.Errorf("expected API_KEY to be masked, got: %s", envView)
	}
}

func TestMaskEnvVar(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PORT=8080", "PORT=8080"},
		{"DB_PASS=mysecret", "DB_PASS=******"},
		{"AUTH_TOKEN=jwt1234", "AUTH_TOKEN=******"},
		{"AWS_SECRET_ACCESS_KEY=xyz", "AWS_SECRET_ACCESS_KEY=******"},
		{"CREDENTIALS_FILE=/tmp/cred", "CREDENTIALS_FILE=******"},
		{"NO_EQUALS", "NO_EQUALS"},
	}

	for _, tt := range tests {
		actual := MaskEnvVar(tt.input)
		if actual != tt.expected {
			t.Errorf("MaskEnvVar(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
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

func TestRenderPrune(t *testing.T) {
	// Error view
	errView := RenderPrune(monitor.DockerDiskUsage{}, "daemon error", 0)
	if !strings.Contains(errView, "daemon error") {
		t.Errorf("expected error message in prune view, got: %s", errView)
	}

	// Menu view
	usage := monitor.DockerDiskUsage{
		Images:     monitor.DockerDiskItem{TotalCount: 5, ActiveCount: 2, TotalSize: 1024 * 1024 * 500, Reclaimable: 1024 * 1024 * 200},
		Containers: monitor.DockerDiskItem{TotalCount: 3, ActiveCount: 1, TotalSize: 1024 * 1024 * 50, Reclaimable: 1024 * 1024 * 30},
		TotalSize:  1024 * 1024 * 550,
		TotalRecl:  1024 * 1024 * 230,
	}
	menuView := RenderPrune(usage, "", 0)
	if !strings.Contains(menuView, "CLEANUP & PRUNE DOCKER RESOURCES") || !strings.Contains(menuView, "[1]") {
		t.Errorf("expected prune menu items, got: %s", menuView)
	}

	// Confirmation step view
	confirmView := RenderPrune(usage, "", 1)
	if !strings.Contains(confirmView, "WARNING:") || !strings.Contains(confirmView, "[Y]") {
		t.Errorf("expected prune confirmation view, got: %s", confirmView)
	}
}

func TestGetPruneActionName(t *testing.T) {
	if act1 := getPruneActionName(1); act1 != "Prune Stopped Containers" {
		t.Errorf("expected Prune Stopped Containers, got %q", act1)
	}
	if act2 := getPruneActionName(2); act2 != "Prune Dangling & Unused Images" {
		t.Errorf("expected Prune Dangling & Unused Images, got %q", act2)
	}
	if act3 := getPruneActionName(3); act3 != "Prune Unused Volumes" {
		t.Errorf("expected Prune Unused Volumes, got %q", act3)
	}
	if act4 := getPruneActionName(4); act4 != "Prune System (All Unused Resources)" {
		t.Errorf("expected Prune System (All Unused Resources), got %q", act4)
	}
	if actOther := getPruneActionName(99); actOther != "" {
		t.Errorf("expected empty string for unknown step, got %q", actOther)
	}
}

func TestRenderComposeDown(t *testing.T) {
	view := RenderComposeDown("my-stack")
	if !strings.Contains(view, "my-stack") || !strings.Contains(view, "CONFIRM DOCKER COMPOSE DOWN") {
		t.Errorf("expected project name and title in compose down modal, got: %s", view)
	}
}

func TestRenderRemoveContainer(t *testing.T) {
	view := RenderRemoveContainer("my-db", "c98765")
	if !strings.Contains(view, "my-db") || !strings.Contains(view, "c98765") || !strings.Contains(view, "CONFIRM CONTAINER REMOVAL") {
		t.Errorf("expected container name, id and title in remove modal, got: %s", view)
	}
}

func TestRenderRemoveImage(t *testing.T) {
	// Normal unused image
	view := RenderRemoveImage("nginx:alpine", "img12345", false)
	if !strings.Contains(view, "nginx:alpine") || !strings.Contains(view, "img12345") || !strings.Contains(view, "CONFIRM IMAGE REMOVAL") {
		t.Errorf("expected image name, id and title in remove image modal, got: %s", view)
	}

	// In use image with warning
	viewInUse := RenderRemoveImage("redis:latest", "img67890", true)
	if !strings.Contains(viewInUse, "Warning: Image is currently in use by containers") {
		t.Errorf("expected in-use warning in modal, got: %s", viewInUse)
	}
}

func TestRenderHelp(t *testing.T) {
	view := RenderHelp(2 * time.Second)
	if !strings.Contains(view, "HELP & KEYBOARD SHORTCUTS") || !strings.Contains(view, "[K]") || !strings.Contains(view, "[x]") {
		t.Errorf("expected help title and new shortcut keys [K] and [x], got: %s", view)
	}
	if !strings.Contains(view, "NAVIGATION") || !strings.Contains(view, "CONTAINER ACTIONS") {
		t.Errorf("expected section titles in help view, got: %s", view)
	}
}

func TestRenderExec(t *testing.T) {
	// 1. Menu view
	menuView := RenderExec(ExecModalContext{
		ContainerName: "my-web",
		ContainerID:   "c999",
		CursorIdx:     0,
		IsCustomInput: false,
	})

	if !strings.Contains(menuView, "my-web") || !strings.Contains(menuView, "c999") {
		t.Errorf("expected container name and id in exec modal, got: %s", menuView)
	}
	if !strings.Contains(menuView, "Embedded Bash / Sh") || !strings.Contains(menuView, "Docker Attach") {
		t.Errorf("expected modes in exec modal, got: %s", menuView)
	}

	// 2. Custom input view
	customView := RenderExec(ExecModalContext{
		ContainerName: "my-web",
		ContainerID:   "c999",
		IsCustomInput: true,
		CustomCmd:     "htop",
	})

	if !strings.Contains(customView, "htop") {
		t.Errorf("expected custom command in custom input view, got: %s", customView)
	}
}

func TestRenderExecOffersEmbeddedAndExternalShells(t *testing.T) {
	view := RenderExec(ExecModalContext{
		ContainerName: "api",
		ContainerID:   "c123",
	})

	if !strings.Contains(view, "Embedded Bash / Sh") {
		t.Fatalf("expected the recommended embedded shell option, got: %s", view)
	}
	if !strings.Contains(view, "External Bash / Sh") {
		t.Fatalf("expected an external shell fallback, got: %s", view)
	}
}

func TestRenderInspect_Scrolling(t *testing.T) {
	detail := monitor.ContainerDetail{
		ID:     "c1234567890abcdef",
		Name:   "huge-app",
		Image:  "my-app:v1",
		State:  "running",
		Status: "Up 10 hours",
		Health: "unhealthy",
		Mounts: []string{"/m1", "/m2", "/m3"},
		Env: []string{
			"VAR1=val1", "VAR2=val2", "VAR3=val3", "VAR4=val4",
			"VAR5=val5", "VAR6=val6", "VAR7=val7", "VAR8=val8",
			"VAR9=val9", "VAR10=val10", "VAR11=val11", "VAR12=val12",
		},
	}

	lines := BuildInspectLines(detail)
	if len(lines) <= DefaultMaxVisibleInspect {
		t.Fatalf("expected more than %d lines, got %d", DefaultMaxVisibleInspect, len(lines))
	}

	// Render page 1 (scroll = 0)
	v0 := RenderInspect(detail, 0)
	if !strings.Contains(v0, "[Lines 1-18 of") || !strings.Contains(v0, "[y] Copy ID") {
		t.Errorf("expected pagination footer in v0, got: %s", v0)
	}

	// Render scrolled
	vScroll := RenderInspect(detail, 5)
	if !strings.Contains(vScroll, "[Lines 6-") {
		t.Errorf("expected scrolled lines indicator, got: %s", vScroll)
	}
}

func TestBuildInspectLines_Variants(t *testing.T) {
	// Test starting status
	startingDetail := monitor.ContainerDetail{
		ID:     "starting-c",
		Name:   "starter",
		Health: "starting",
	}
	lines := BuildInspectLines(startingDetail)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "[starting]") {
		t.Errorf("expected [starting] health badge, got: %s", joined)
	}
}
