package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/testutil"
)

func TestRenderDockerView(t *testing.T) {
	// Empty
	emptyView := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running"},
	})
	if !strings.Contains(emptyView, "No running containers found.") {
		t.Errorf("expected no containers running in docker view, got: %s", emptyView)
	}

	// Containers table with details panel using fixture builder
	containers := []monitor.ContainerItem{
		testutil.AContainer(
			testutil.WithID("c12345678901"),
			testutil.WithName("api-server"),
			testutil.WithImage("golang:alpine"),
			testutil.WithMetrics(12.5, 250.0, 1024.0),
			testutil.WithPorts("8080->8080/tcp"),
			testutil.WithMounts("/vol -> /app (rw)"),
			testutil.WithCompose("my-app", "api"),
			testutil.WithNetworkIO(1024*1024, 2048*1024),
			testutil.WithBlockIO(512*1024, 1536*1024),
		),
	}

	view := RenderDocker(DockerContext{
		DockerInfo:           monitor.DockerSummary{Status: "running", Containers: containers, ImagesCount: 5, Version: "27.1.0"},
		Containers:           containers,
		TotalContainersCount: 1,
		CursorIdx:            0,
		Width:                100,
	})

	testutil.AssertContainsAll(t, view,
		"api-server",
		"[healthy]",
		"8080->8080/tcp",
		"/vol -> /app (rw)",
		"my-app",
		"Net I/O:",
		"1.0 MB / 2.0 MB",
		"Block I/O:",
		"512.0 KB / 1.5 MB",
	)
}

func TestFormatContainerStatus(t *testing.T) {
	tests := []struct {
		state     string
		health    string
		rawStatus string
		expected  string
	}{
		{"running", "healthy", "Up 2 hours (healthy)", "● [healthy]"},
		{"running", "unhealthy", "Up 2 hours (unhealthy)", "▲ [unhealthy]"},
		{"running", "starting", "Up 2 minutes (health: starting)", "⏳ [starting]"},
		{"running", "", "Up 5 hours", "● running"},
		{"paused", "", "Paused", "⏸ paused"},
		{"restarting", "", "Restarting (1) 2s ago", "↻ restarting"},
		{"dead", "", "Dead", "✖ dead"},
		{"exited", "", "Exited (1) 5 seconds ago", "✖ failed (1)"},
		{"exited", "", "Exited (137) 10 seconds ago", "✖ failed (137)"},
		{"exited", "", "Exited (0) 10 minutes ago", "○ exited (0)"},
		{"exited", "", "", "○ exited"},
		{"", "", "", "○ stopped"},
	}

	for _, tc := range tests {
		var text string
		if tc.rawStatus != "" {
			text, _ = formatContainerStatus(tc.state, tc.health, tc.rawStatus)
		} else {
			text, _ = formatContainerStatus(tc.state, tc.health)
		}
		if text != tc.expected {
			t.Errorf("formatContainerStatus(%q, %q, %q) = %q, expected %q", tc.state, tc.health, tc.rawStatus, text, tc.expected)
		}
	}
}

func TestFormatContainerMetrics(t *testing.T) {
	// Stopped container
	cpu, mem := formatContainerMetrics(monitor.ContainerItem{State: "exited"})
	if cpu != "-" || mem != "-" {
		t.Errorf("expected - and - for stopped container, got %s and %s", cpu, mem)
	}

	// Running with limit
	cpu, mem = formatContainerMetrics(monitor.ContainerItem{
		State:      "running",
		CPUPerc:    15.5,
		MemUsageMB: 120.0,
		MemLimitMB: 512.0,
		MemPerc:    23.4,
	})
	if cpu != "15.5%" || !strings.Contains(mem, "120.0M (23.4%)") {
		t.Errorf("unexpected metrics format: %s, %s", cpu, mem)
	}

	// Running without limit (MemLimitMB == 0)
	_, memNoLimit := formatContainerMetrics(monitor.ContainerItem{
		State:      "running",
		MemUsageMB: 64.0,
		MemLimitMB: 0,
	})
	if memNoLimit != "64.0M" {
		t.Errorf("expected 64.0M, got: %s", memNoLimit)
	}
}

func TestStyleCPU(t *testing.T) {
	// Stopped container returns DimStyle
	stopped := styleCPU(99.0, "exited", "  99.0%")
	if !strings.Contains(stopped, "99.0%") {
		t.Errorf("expected formatted string in output, got: %s", stopped)
	}

	// Running >= 80
	high := styleCPU(85.0, "running", "  85.0%")
	if !strings.Contains(high, "85.0%") {
		t.Errorf("expected formatted string in output, got: %s", high)
	}

	// Running >= 50
	mid := styleCPU(60.0, "running", "  60.0%")
	if !strings.Contains(mid, "60.0%") {
		t.Errorf("expected formatted string in output, got: %s", mid)
	}

	// Running normal
	low := styleCPU(20.0, "running", "  20.0%")
	if !strings.Contains(low, "20.0%") {
		t.Errorf("expected formatted string in output, got: %s", low)
	}
}

func TestRenderDockerView_EdgeCases(t *testing.T) {
	// 1. Daemon down
	downView := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "stopped"},
	})
	if !strings.Contains(downView, "Docker") {
		t.Errorf("expected daemon down notice, got: %s", downView)
	}

	// 2. Filter active with results and sorting badge
	containers := []monitor.ContainerItem{
		{ID: "c1", Name: "app", State: "running", Ports: "80:80"},
	}
	filteredView := RenderDocker(DockerContext{
		DockerInfo:           monitor.DockerSummary{Status: "running", Containers: containers},
		Containers:           containers,
		TotalContainersCount: 1,
		CursorIdx:            -1, // Edge case: invalid negative cursor should not panic
		FilterQuery:          "app",
		SortMode:             1,
		SortNames:            []string{"Default", "CPU"},
	})
	if !strings.Contains(filteredView, "Filter:") || !strings.Contains(filteredView, "Sort:") {
		t.Errorf("expected filter and sort badges, got: %s", filteredView)
	}

	// 3. Filter mode active (IsFiltering == true)
	filteringView := RenderDocker(DockerContext{
		DockerInfo:  monitor.DockerSummary{Status: "running", Containers: containers},
		Containers:  containers,
		IsFiltering: true,
		FilterQuery: "ap",
	})
	if !strings.Contains(filteringView, "[Search: ap") {
		t.Errorf("expected search prompt in filtering view, got: %s", filteringView)
	}

	// 4. No containers matched filter
	noMatchView := RenderDocker(DockerContext{
		DockerInfo:  monitor.DockerSummary{Status: "running", Containers: containers},
		Containers:  nil,
		FilterQuery: "xyz",
	})
	if !strings.Contains(noMatchView, "No containers found matching") {
		t.Errorf("expected no match notice, got: %s", noMatchView)
	}

	// 5. Cursor out of bounds should not panic
	outOfBoundsView := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: containers},
		Containers: containers,
		CursorIdx:  999,
	})
	if !strings.Contains(outOfBoundsView, "app") {
		t.Errorf("expected container row in view, got: %s", outOfBoundsView)
	}
}

func TestFormatPortDisplay(t *testing.T) {
	// Empty input
	if got := FormatPortDisplay(""); got != nil {
		t.Errorf("expected nil for empty string, got: %v", got)
	}
	if got := FormatPortDisplay("   "); got != nil {
		t.Errorf("expected nil for whitespace string, got: %v", got)
	}

	// Single exposed port
	if got := FormatPortDisplay("80/tcp"); len(got) != 1 || got[0] != "80/tcp" {
		t.Errorf("expected [80/tcp], got: %v", got)
	}

	// Arrow conversion
	if got := FormatPortDisplay("8080->80/tcp"); len(got) != 1 || got[0] != "8080 → 80/tcp" {
		t.Errorf("expected arrow converted, got: %v", got)
	}

	// IPv6 deduplication
	ipv46 := "0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp, 0.0.0.0:443->443/tcp"
	got := FormatPortDisplay(ipv46)
	if len(got) != 2 {
		t.Fatalf("expected 2 deduped ports, got %d: %v", len(got), got)
	}
	if got[0] != "0.0.0.0:8080 → 8080/tcp" || got[1] != "0.0.0.0:443 → 443/tcp" {
		t.Errorf("unexpected deduped ports: %v", got)
	}
}

func TestRenderContainerDetailsCard_Extended(t *testing.T) {
	// 1. Container with complete extended fields (IP, Network, RestartPolicy, Created)
	cFull := testutil.AContainer(
		testutil.WithID("c123456789ab"),
		testutil.WithName("prod-api"),
		testutil.WithImage("api:v2"),
		testutil.WithState("running", "Up 4 hours"),
		testutil.WithPorts("0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp"),
		testutil.WithNetwork("10.0.1.25", "backend-net"),
		testutil.WithRestartPolicy("always"),
		testutil.WithCreated("2026-11-07 12:00:00"),
	)

	viewFull := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cFull}},
		Containers: []monitor.ContainerItem{cFull},
		CursorIdx:  0,
		Width:      100,
	})

	testutil.AssertContainsAll(t, viewFull,
		"10.0.1.25",
		"backend-net",
		"always",
		"2026-11-07 12:00:00",
		"0.0.0.0:8080 → 8080/tcp",
	)

	// 2. Container with on-failure restart policy and no IP
	cOnFailure := testutil.AContainer(
		testutil.WithID("c98765432100"),
		testutil.WithName("worker-job"),
		testutil.WithImage("job:latest"),
		testutil.WithState("stopped", "Exited (1)"),
		testutil.WithRestartPolicy("on-failure"),
		testutil.WithNetwork("", ""),
	)

	viewOnFail := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cOnFailure}},
		Containers: []monitor.ContainerItem{cOnFailure},
		CursorIdx:  0,
		Width:      100,
	})

	testutil.AssertContainsAll(t, viewOnFail,
		"on-failure",
		"none",
	)

	// 3. Container with ErrorSnippet
	cError := testutil.AContainer(
		testutil.WithID("cfail123"),
		testutil.WithName("crashed-db"),
		testutil.WithImage("postgres:15"),
		testutil.WithState("exited", "Exited (1) 2 minutes ago"),
	)
	cError.ErrorSnippet = "FATAL: password authentication failed\nLOG: database system is shut down"

	viewError := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cError}},
		Containers: []monitor.ContainerItem{cError},
		CursorIdx:  0,
		Width:      100,
	})

	testutil.AssertContainsAll(t, viewError,
		"Last Error:",
		"password authentication failed",
	)
}

func TestExtractPortURLs(t *testing.T) {
	tests := []struct {
		ports    string
		expected []string
	}{
		{"", nil},
		{"   ", nil},
		{"80/tcp", nil},
		{"0.0.0.0:8080->80/tcp", []string{"http://localhost:8080"}},
		{"0.0.0.0:8080->80/tcp, [::]:8080->80/tcp", []string{"http://localhost:8080"}},
		{"127.0.0.1:3000->3000/tcp, 0.0.0.0:8080->80/tcp", []string{"http://localhost:3000", "http://localhost:8080"}},
		{"0.0.0.0:443->443/tcp", []string{"https://localhost"}},
		{"0.0.0.0:80->80/tcp", []string{"http://localhost"}},
	}

	for _, tt := range tests {
		actual := ExtractPortURLs(tt.ports)
		if len(actual) != len(tt.expected) {
			t.Fatalf("ExtractPortURLs(%q) len = %d, expected %d: %v vs %v", tt.ports, len(actual), len(tt.expected), actual, tt.expected)
		}
		for i := range actual {
			if actual[i] != tt.expected[i] {
				t.Errorf("ExtractPortURLs(%q)[%d] = %q, expected %q", tt.ports, i, actual[i], tt.expected[i])
			}
		}
	}
}
