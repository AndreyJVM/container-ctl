package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
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

	// Running with limit (< 1024MB)
	cpu, mem = formatContainerMetrics(monitor.ContainerItem{
		State:      "running",
		CPUPerc:    15.5,
		MemUsageMB: 120.0,
		MemLimitMB: 512.0,
		MemPerc:    23.4,
	})
	if cpu != "15.5%" || mem != "120.0M" {
		t.Errorf("unexpected metrics format: %s, %s", cpu, mem)
	}

	// Running with large memory (>= 1024MB)
	_, memGigabytes := formatContainerMetrics(monitor.ContainerItem{
		State:      "running",
		MemUsageMB: 2048.0,
	})
	if memGigabytes != "2.00G" {
		t.Errorf("expected 2.00G, got: %s", memGigabytes)
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

func TestStyleMem(t *testing.T) {
	// Stopped container returns DimStyle
	stopped := styleMem(100.0, 50.0, "exited", "  100.0M")
	if !strings.Contains(stopped, "100.0M") {
		t.Errorf("expected formatted string, got: %s", stopped)
	}

	// Running >= 85
	high := styleMem(900.0, 90.0, "running", "  900.0M")
	if !strings.Contains(high, "900.0M") {
		t.Errorf("expected formatted string, got: %s", high)
	}

	// Running >= 70
	warn := styleMem(750.0, 75.0, "running", "  750.0M")
	if !strings.Contains(warn, "750.0M") {
		t.Errorf("expected formatted string, got: %s", warn)
	}

	// Running normal
	normal := styleMem(200.0, 20.0, "running", "  200.0M")
	if !strings.Contains(normal, "200.0M") {
		t.Errorf("expected formatted string, got: %s", normal)
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

func TestCleanTablePorts(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"", "-"},
		{"   ", "-"},
		{"80/tcp", "80/tcp"},
		{"0.0.0.0:8080->8080/tcp", "8080->8080/tcp"},
		{"0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp", "8080->8080/tcp"},
		{"0.0.0.0:8080->8080/tcp, :::8080->8080/tcp", "8080->8080/tcp"},
		{"0.0.0.0:80->80/tcp, 0.0.0.0:443->443/tcp", "80->80/tcp, 443->443/tcp"},
		{"127.0.0.1:5432->5432/tcp", "127.0.0.1:5432->5432/tcp"},
	}

	for _, c := range cases {
		got := CleanTablePorts(c.input)
		if got != c.expected {
			t.Errorf("CleanTablePorts(%q) = %q, expected %q", c.input, got, c.expected)
		}
	}
}

func TestComputeContainerTableLayout(t *testing.T) {
	widths := []int{76, 80, 96, 120, 160}
	for _, w := range widths {
		layout := computeContainerTableLayout(w)
		total := 2 + layout.idWidth + 2 + layout.nameWidth + 2 + layout.imageWidth + 2 + layout.statusWidth + 2 + layout.cpuWidth + 2 + layout.memWidth + 2 + layout.portsWidth
		if total != layout.contentWidth {
			t.Errorf("width %d: expected sum of columns (%d) to equal contentWidth (%d)", w, total, layout.contentWidth)
		}
		if layout.portsWidth <= 0 {
			t.Errorf("width %d: portsWidth should be positive, got %d", w, layout.portsWidth)
		}
	}
}

func TestRenderDockerView_WideWithGauges(t *testing.T) {
	containers := []monitor.ContainerItem{
		testutil.AContainer(
			testutil.WithID("c12345678901"),
			testutil.WithName("api-service"),
			testutil.WithImage("golang:alpine"),
			testutil.WithMetrics(45.5, 300.0, 1024.0),
			testutil.WithPorts("8080->8080/tcp"),
		),
		testutil.AContainer(
			testutil.WithID("cstopped0001"),
			testutil.WithName("worker-idle"),
			testutil.WithImage("alpine:latest"),
			testutil.WithState("exited", "Exited (0)"),
		),
	}

	// 1. Wide view (width 140, layout.showGauges == true)
	viewWide := RenderDocker(DockerContext{
		DockerInfo:           monitor.DockerSummary{Status: "running", Containers: containers},
		Containers:           containers,
		TotalContainersCount: 2,
		CursorIdx:            0,
		Width:                140,
	})

	testutil.AssertContainsAll(t, viewWide,
		"api-service",
		"worker-idle",
		"45.5%",
		"300.0M",
		"CPU Usage:",
		"Memory Usage:",
	)

	// 2. Cursor on stopped container
	viewStoppedSelected := RenderDocker(DockerContext{
		DockerInfo:           monitor.DockerSummary{Status: "running", Containers: containers},
		Containers:           containers,
		TotalContainersCount: 2,
		CursorIdx:            1,
		Width:                140,
	})

	testutil.AssertContainsAll(t, viewStoppedSelected,
		"- (stopped)",
		"CPU Usage:",
		"Memory Usage:",
	)
}

func TestRenderDockerView_HideDetails(t *testing.T) {
	containers := []monitor.ContainerItem{
		testutil.AContainer(
			testutil.WithID("c12345678901"),
			testutil.WithName("api-server"),
			testutil.WithImage("golang:alpine"),
		),
	}

	// 1. Details visible (HideDetails == false)
	viewWithDetails := RenderDocker(DockerContext{
		DockerInfo:           monitor.DockerSummary{Status: "running", Containers: containers},
		Containers:           containers,
		TotalContainersCount: 1,
		CursorIdx:            0,
		Width:                100,
		HideDetails:          false,
	})
	if !strings.Contains(viewWithDetails, "Quick Actions:") {
		t.Errorf("expected details card with Quick Actions when HideDetails is false")
	}

	// 2. Details hidden (HideDetails == true)
	viewWithoutDetails := RenderDocker(DockerContext{
		DockerInfo:           monitor.DockerSummary{Status: "running", Containers: containers},
		Containers:           containers,
		TotalContainersCount: 1,
		CursorIdx:            0,
		Width:                100,
		HideDetails:          true,
	})
	if strings.Contains(viewWithoutDetails, "Quick Actions:") {
		t.Errorf("expected details card to be hidden when HideDetails is true")
	}
	if !strings.Contains(viewWithoutDetails, "api-server") {
		t.Errorf("expected container row to still be rendered when HideDetails is true")
	}
}
