package e2e

import (
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

// createMultiComposeModel builds a model populated with multiple Compose projects in various states.
func createMultiComposeModel() tui.Model {
	m := tui.NewModelWithOptions(
		2*time.Second,
		int(tui.TabCompose),
		"v0.22.7",
		tui.WithShowSwarm(true),
	)

	data := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Status:        "running",
			StatusDisplay: "Running",
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c-fe-1"), testutil.WithName("web-frontend"), testutil.WithCompose("frontend-app", "web"), testutil.WithState("running", "Up 1h")),
				testutil.AContainer(testutil.WithID("c-fe-2"), testutil.WithName("ssr-frontend"), testutil.WithCompose("frontend-app", "ssr"), testutil.WithState("running", "Up 1h")),
				testutil.AContainer(testutil.WithID("c-an-1"), testutil.WithName("analytics-worker"), testutil.WithCompose("analytics-stack", "worker"), testutil.WithState("running", "Up 30m")),
				testutil.AContainer(testutil.WithID("c-an-2"), testutil.WithName("analytics-db"), testutil.WithCompose("analytics-stack", "clickhouse"), testutil.WithState("exited", "Exited (1) 5m ago")),
				testutil.AContainer(testutil.WithID("c-mon-1"), testutil.WithName("mon-grafana"), testutil.WithCompose("monitoring-tools", "grafana"), testutil.WithState("exited", "Exited (0) 2h ago")),
				testutil.AContainer(testutil.WithID("c-mon-2"), testutil.WithName("mon-prom"), testutil.WithCompose("monitoring-tools", "prometheus"), testutil.WithState("exited", "Exited (0) 2h ago")),
			},
		},
		Compose: []monitor.ComposeProject{
			{
				Name:            "frontend-app",
				Status:          "running",
				StatusDisplay:   "Running",
				TotalServices:   2,
				RunningServices: 2,
				Services: []monitor.ComposeService{
					{Name: "web", ContainerName: "web-frontend", State: "running", Status: "Up 1h", Image: "frontend-web:latest"},
					{Name: "ssr", ContainerName: "ssr-frontend", State: "running", Status: "Up 1h", Image: "frontend-ssr:latest"},
				},
			},
			{
				Name:            "analytics-stack",
				Status:          "partially running",
				StatusDisplay:   "Partially Running",
				TotalServices:   2,
				RunningServices: 1,
				Services: []monitor.ComposeService{
					{Name: "worker", ContainerName: "analytics-worker", State: "running", Status: "Up 30m", Image: "python:3.11"},
					{Name: "clickhouse", ContainerName: "analytics-db", State: "exited", Status: "Exited (1) 5m ago", Image: "clickhouse/clickhouse-server"},
				},
			},
			{
				Name:            "monitoring-tools",
				Status:          "stopped",
				StatusDisplay:   "Stopped",
				TotalServices:   2,
				RunningServices: 0,
				Services: []monitor.ComposeService{
					{Name: "grafana", ContainerName: "mon-grafana", State: "exited", Status: "Exited (0) 2h ago", Image: "grafana/grafana:latest"},
					{Name: "prometheus", ContainerName: "mon-prom", State: "exited", Status: "Exited (0) 2h ago", Image: "prom/prometheus:latest"},
				},
			},
		},
		Timestamp: "14:00:00",
	}

	updated, _ := m.Update(data)
	return updated.(tui.Model)
}

func TestE2E_ComposeDeepNavigationAndFilter(t *testing.T) {
	t.Parallel()
	m := createMultiComposeModel()
	assertTab(t, m, tui.TabCompose)

	projects := m.FilteredCompose()
	if len(projects) != 3 {
		t.Fatalf("expected 3 compose projects, got %d", len(projects))
	}

	// Initial cursor on 0 (frontend-app)
	assertCursor(t, m, 0)

	// Cyclic wrap with up arrow: 0 -> 2 (monitoring-tools)
	m, _ = sendKey(m, "up")
	assertCursor(t, m, 2)

	// Move down arrow: 2 -> 0
	m, _ = sendKey(m, "down")
	assertCursor(t, m, 0)

	// Search by project name: "/" -> "analytics"
	m = sendFilter(m, "analytics")
	assertFiltering(t, m, true)
	assertFilterQuery(t, m, "analytics")

	filtered := m.FilteredCompose()
	if len(filtered) != 1 || filtered[0].Name != "analytics-stack" {
		t.Fatalf("expected 1 project 'analytics-stack', got %d: %+v", len(filtered), filtered)
	}

	// Clear filter with Esc
	m, _ = sendKey(m, "esc")
	assertFilterQuery(t, m, "")
	if len(m.FilteredCompose()) != 3 {
		t.Fatalf("expected 3 projects restored after Esc, got %d", len(m.FilteredCompose()))
	}

	// Search deep by service name inside project: "grafana"
	m = sendFilter(m, "grafana")
	filtered = m.FilteredCompose()
	if len(filtered) != 1 || filtered[0].Name != "monitoring-tools" {
		t.Fatalf("expected 'monitoring-tools' found by service name 'grafana', got %d: %+v", len(filtered), filtered)
	}

	// Clear filter
	m, _ = sendKey(m, "esc")

	// Search by container image inside compose service: "clickhouse"
	m = sendFilter(m, "clickhouse")
	filtered = m.FilteredCompose()
	if len(filtered) != 1 || filtered[0].Name != "analytics-stack" {
		t.Fatalf("expected 'analytics-stack' found by service image 'clickhouse', got %d: %+v", len(filtered), filtered)
	}

	// Clear filter
	m, _ = sendKey(m, "esc")
	if len(m.FilteredCompose()) != 3 {
		t.Fatalf("expected 3 projects restored, got %d", len(m.FilteredCompose()))
	}
}

func TestE2E_ComposeLifecycleActionsAndModals(t *testing.T) {
	t.Parallel()
	m := createMultiComposeModel()
	assertTab(t, m, tui.TabCompose)

	// Cursor is at 0 ("frontend-app")
	assertCursor(t, m, 0)

	// Press 's' to stop running Compose project
	m, stopCmd := sendKey(m, "s")
	if stopCmd == nil {
		t.Fatal("expected non-nil cmd when stopping Compose project")
	}
	assertContains(t, m.View(), "Stopping Compose project frontend-app...")

	// Press 'p' to restart Compose project
	m, restartCmd := sendKey(m, "p")
	if restartCmd == nil {
		t.Fatal("expected non-nil cmd when restarting Compose project")
	}
	assertContains(t, m.View(), "Restarting Compose project frontend-app...")

	// Move cursor to stopped project ("monitoring-tools" at index 2)
	m, _ = sendKey(m, "down") // index 1
	m, _ = sendKey(m, "down") // index 2
	assertCursor(t, m, 2)

	// Press 'u' to start stopped Compose project
	m, startCmd := sendKey(m, "u")
	if startCmd == nil {
		t.Fatal("expected non-nil cmd when starting Compose project")
	}
	assertContains(t, m.View(), "Starting Compose project monitoring-tools...")

	// Press 'd' to open Compose Down confirmation modal
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalComposeDown)
	assertContains(t, m.View(), "monitoring-tools")

	// Cancel compose down modal with 'esc'
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// Press 'l' to open Compose Logs modal
	m, _ = sendKey(m, "l")
	assertModal(t, m, tui.ModalLogs)
	assertContains(t, m.View(), "monitoring-tools")

	// Close logs modal with 'esc'
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}
