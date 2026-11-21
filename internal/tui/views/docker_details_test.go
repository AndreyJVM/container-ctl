package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
)

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
		{"0.0.0.0:53->53/udp", nil},
		{"0.0.0.0:8080->8080/tcp, 0.0.0.0:53->53/udp", []string{"http://localhost:8080"}},
		{"0.0.0.0:99999->99999/tcp", nil},
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

func TestQuickActionsInDetailsCard(t *testing.T) {
	// Running container with mapped web port
	cRunning := testutil.AContainer(
		testutil.WithID("c123"),
		testutil.WithName("web-app"),
		testutil.WithState("running", "Up 1 hour"),
		testutil.WithPorts("0.0.0.0:8080->8080/tcp"),
	)
	viewRunning := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cRunning}},
		Containers: []monitor.ContainerItem{cRunning},
		CursorIdx:  0,
		Width:      100,
	})
	testutil.AssertContainsAll(t, viewRunning, "Quick Actions:", "[S]", "Stop", "[P]", "Restart", "[w]", "Web", "[E]", "Shell")

	// Stopped container without web port
	cStopped := testutil.AContainer(
		testutil.WithID("c456"),
		testutil.WithName("batch-job"),
		testutil.WithState("exited", "Exited (0)"),
	)
	viewStopped := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cStopped}},
		Containers: []monitor.ContainerItem{cStopped},
		CursorIdx:  0,
		Width:      100,
	})
	testutil.AssertContainsAll(t, viewStopped, "Quick Actions:", "[S]", "Start", "[x]", "Remove", "[L]", "Logs")
}

func TestContainerDetailsCard_StreamlinedMetricsAndCapping(t *testing.T) {
	// 1. Capping ports and mounts when exceeding 3 items
	cMany := testutil.AContainer(
		testutil.WithID("c999"),
		testutil.WithName("multi-service"),
		testutil.WithState("running", "Up 3 hours"),
		testutil.WithPorts("80->80/tcp, 443->443/tcp, 8080->8080/tcp, 8443->8443/tcp, 9000->9000/tcp"),
		testutil.WithMounts("/v1 -> /a", "/v2 -> /b", "/v3 -> /c", "/v4 -> /d", "/v5 -> /e"),
		testutil.WithNetworkIO(1024, 2048),
		testutil.WithBlockIO(512, 1024),
	)
	viewMany := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cMany}},
		Containers: []monitor.ContainerItem{cMany},
		CursorIdx:  0,
		Width:      100,
	})
	testutil.AssertContainsAll(t, viewMany,
		"... (+2 more)",
		"(↓ rx / ↑ tx)",
		"(read / write)",
	)

	// 2. Stopped container with 0 bytes has no Net I/O or Block I/O clutter
	cZeroStopped := testutil.AContainer(
		testutil.WithID("c000"),
		testutil.WithName("clean-stopped"),
		testutil.WithState("exited", "Exited (0)"),
		testutil.WithNetworkIO(0, 0),
		testutil.WithBlockIO(0, 0),
	)
	viewZeroStopped := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cZeroStopped}},
		Containers: []monitor.ContainerItem{cZeroStopped},
		CursorIdx:  0,
		Width:      100,
	})
	if strings.Contains(viewZeroStopped, "Net I/O:") || strings.Contains(viewZeroStopped, "Block I/O:") {
		t.Errorf("expected stopped container with 0 bytes to omit Net/Block I/O, got: %s", viewZeroStopped)
	}
}

func TestRenderContainerDetailsCard_ResourceGauges(t *testing.T) {
	// 1. Running container with memory limit
	cWithLimit := testutil.AContainer(
		testutil.WithID("cgauge01"),
		testutil.WithName("gauge-demo"),
		testutil.WithState("running", "Up 10 minutes"),
		testutil.WithMetrics(35.0, 512.0, 1024.0),
	)

	viewLimit := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cWithLimit}},
		Containers: []monitor.ContainerItem{cWithLimit},
		CursorIdx:  0,
		Width:      100,
	})

	testutil.AssertContainsAll(t, viewLimit,
		"CPU Usage:",
		"35.0%",
		"Memory Usage:",
		"50.0%",
		"512.0 MB / 1.00 GB",
	)

	// 2. Running container without memory limit
	cNoLimit := testutil.AContainer(
		testutil.WithID("cgauge02"),
		testutil.WithName("gauge-nolimit"),
		testutil.WithState("running", "Up 20 minutes"),
	)
	cNoLimit.CPUPerc = 85.0
	cNoLimit.MemUsageMB = 256.0
	cNoLimit.MemLimitMB = 0
	cNoLimit.MemPerc = 12.5

	viewNoLimit := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cNoLimit}},
		Containers: []monitor.ContainerItem{cNoLimit},
		CursorIdx:  0,
		Width:      100,
	})

	testutil.AssertContainsAll(t, viewNoLimit,
		"CPU Usage:",
		"85.0%",
		"Memory Usage:",
		"256.0 MB",
		"12.5% host",
	)

	// 3. Stopped container
	cStopped := testutil.AContainer(
		testutil.WithID("cgauge03"),
		testutil.WithName("gauge-stopped"),
		testutil.WithState("exited", "Exited (0)"),
	)

	viewStopped := RenderDocker(DockerContext{
		DockerInfo: monitor.DockerSummary{Status: "running", Containers: []monitor.ContainerItem{cStopped}},
		Containers: []monitor.ContainerItem{cStopped},
		CursorIdx:  0,
		Width:      100,
	})

	testutil.AssertContainsAll(t, viewStopped,
		"CPU Usage:",
		"- (stopped)",
		"Memory Usage:",
	)
}

func FuzzExtractPortURLs(f *testing.F) {
	// Seed corpus
	f.Add("0.0.0.0:80->80/tcp")
	f.Add("0.0.0.0:443->443/tcp")
	f.Add("0.0.0.0:8080->8080/tcp, :::8080->8080/tcp")
	f.Add("127.0.0.1:3000->3000/tcp, 0.0.0.0:53->53/udp")
	f.Add(":::80->80/tcp")
	f.Add("invalid")
	f.Add("")
	f.Add("0.0.0.0:999999->80/tcp")
	f.Add("[::1]:8080->8080/tcp")
	f.Add(strings.Repeat("0.0.0.0:80->80/tcp,", 50))

	f.Fuzz(func(t *testing.T, portsStr string) {
		urls := ExtractPortURLs(portsStr)
		for _, u := range urls {
			if !strings.HasPrefix(u, "http://localhost") && !strings.HasPrefix(u, "https://localhost") {
				t.Fatalf("ExtractPortURLs produced invalid URL prefix: %s", u)
			}
		}
	})
}

func FuzzFormatPortDisplay(f *testing.F) {
	f.Add("0.0.0.0:80->80/tcp")
	f.Add("0.0.0.0:8080->8080/tcp, :::8080->8080/tcp")
	f.Add(":::443->443/tcp")
	f.Add("invalid")
	f.Add("")
	f.Add("0.0.0.0:1000-1010->1000-1010/tcp")

	f.Fuzz(func(_ *testing.T, portsStr string) {
		// Must never panic on arbitrary input
		_ = FormatPortDisplay(portsStr)
	})
}
