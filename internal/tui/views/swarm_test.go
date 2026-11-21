package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
)

func TestRenderSwarm_InactiveCluster(t *testing.T) {
	t.Parallel()

	view := RenderSwarm(SwarmContext{
		Swarm: monitor.SwarmSummary{
			IsActive:      false,
			StatusDisplay: "Inactive",
		},
		FormatStatus: func(isActive bool) string {
			if isActive {
				return "Active"
			}
			return "Inactive"
		},
		Width: 80,
	})

	testutil.AssertContainsAll(t, view,
		"DOCKER SWARM CLUSTER",
		"Status: Inactive",
		"Host is not part of an active Swarm cluster.",
		"To initialize a cluster, run: docker swarm init",
	)

	if strings.Contains(view, "CLUSTER NODES:") || strings.Contains(view, "SWARM SERVICES:") {
		t.Errorf("expected nodes and services tables absent when swarm is inactive, got: %s", view)
	}
}

func TestRenderSwarm_ActiveClusterHeader(t *testing.T) {
	t.Parallel()

	view := RenderSwarm(SwarmContext{
		Swarm: monitor.SwarmSummary{
			IsActive:      true,
			NodesCount:    5,
			ManagersCount: 3,
		},
		FormatStatus: func(_ bool) string { return "Active (Leader)" },
		Width:        100,
	})

	testutil.AssertContainsAll(t, view,
		"DOCKER SWARM CLUSTER",
		"Status: Active (Leader)",
		"Cluster nodes: 5",
		"Managers: 3",
	)
}

func TestRenderSwarm_NodesTable(t *testing.T) {
	t.Parallel()

	nodes := []monitor.SwarmNode{
		{
			ID:           "node-leader-1",
			Hostname:     "manager-node-primary",
			Role:         "manager",
			State:        "ready",
			Availability: "active",
			ActiveTasks:  4,
			IsLeader:     true,
		},
		{
			ID:           "node-worker-2",
			Hostname:     "worker-long-hostname-overflow",
			Role:         "worker",
			State:        "ready",
			Availability: "", // empty availability should default to "active"
			ActiveTasks:  2,
			IsLeader:     false,
		},
	}

	view := RenderSwarm(SwarmContext{
		Swarm: monitor.SwarmSummary{
			IsActive:      true,
			NodesCount:    2,
			ManagersCount: 1,
			Nodes:         nodes,
		},
		FormatStatus: func(_ bool) string { return "Active" },
		Width:        120,
	})

	t.Run("headers and leader mark", func(t *testing.T) {
		t.Parallel()
		testutil.AssertContainsAll(t, view,
			"CLUSTER NODES:",
			"NODE ID", "HOSTNAME", "ROLE", "STATE", "AVAILABILITY", "TASKS", "LEADER",
			"node-leader-1",
			"[Leader]",
		)
	})

	t.Run("hostname truncation and empty availability fallback", func(t *testing.T) {
		t.Parallel()
		lines := strings.Split(view, "\n")
		var workerLine string
		for _, l := range lines {
			if strings.Contains(l, "node-worker-2") {
				workerLine = l
				break
			}
		}

		if workerLine == "" {
			t.Fatalf("could not find worker node line in view: %s", view)
		}
		// Hostname should be truncated to 18 chars
		if !strings.Contains(workerLine, "worker-long-hos...") {
			t.Errorf("expected truncated hostname in line, got: %q", workerLine)
		}
		// Empty availability should display "active"
		if !strings.Contains(workerLine, "active") {
			t.Errorf("expected default 'active' availability in line, got: %q", workerLine)
		}
		// Non-leader should not have [Leader]
		if strings.Contains(workerLine, "[Leader]") {
			t.Errorf("non-leader node should not have [Leader] tag, got: %q", workerLine)
		}
	})
}

func TestRenderSwarm_ServicesTableAndReplicas(t *testing.T) {
	t.Parallel()

	services := []monitor.SwarmService{
		{
			ID:              "svc-healthy",
			Name:            "web-frontend",
			Image:           "nginx:1.25-alpine",
			Mode:            "replicated",
			Replicas:        "3/3",
			RunningReplicas: 3,
			TargetReplicas:  3,
			Ports:           "80->80/tcp",
		},
		{
			ID:              "svc-degraded",
			Name:            "api-backend",
			Image:           "golang:1.22-alpine",
			Mode:            "replicated",
			Replicas:        "1/3",
			RunningReplicas: 1,
			TargetReplicas:  3,
			Ports:           "8080->8080/tcp",
		},
		{
			ID:              "svc-dead",
			Name:            "cache-redis",
			Image:           "redis:7.2",
			Mode:            "replicated",
			Replicas:        "0/2",
			RunningReplicas: 0,
			TargetReplicas:  2,
			Ports:           "6379->6379/tcp",
		},
		{
			ID:              "svc-global",
			Name:            "log-collector-daemon-long-name",
			Image:           "fluentbit:latest-release-tag",
			Mode:            "global",
			Replicas:        "5/5",
			RunningReplicas: 5,
			TargetReplicas:  0,
			Ports:           "",
		},
	}

	t.Run("cursor selection pointer on active row", func(t *testing.T) {
		t.Parallel()
		view0 := RenderSwarm(SwarmContext{
			Swarm: monitor.SwarmSummary{
				IsActive: true,
				Services: services,
			},
			CursorIdx:    0,
			FormatStatus: func(_ bool) string { return "Active" },
			Width:        120,
		})

		lines := strings.Split(view0, "\n")
		var line0, line1 string
		for _, l := range lines {
			if strings.Contains(l, "svc-healthy") {
				line0 = l
			}
			if strings.Contains(l, "svc-degraded") {
				line1 = l
			}
		}

		if !strings.Contains(line0, "► ") {
			t.Errorf("expected cursor pointer '► ' on row 0, got: %q", line0)
		}
		if strings.Contains(line1, "► ") {
			t.Errorf("did not expect cursor pointer on row 1, got: %q", line1)
		}
	})

	t.Run("replica status indicators and modes", func(t *testing.T) {
		t.Parallel()
		view := RenderSwarm(SwarmContext{
			Swarm: monitor.SwarmSummary{
				IsActive: true,
				Services: services,
			},
			CursorIdx:    1,
			FormatStatus: func(_ bool) string { return "Active" },
			Width:        120,
		})

		testutil.AssertContainsAll(t, view,
			"SWARM SERVICES:",
			"SERVICE ID", "NAME", "IMAGE", "MODE", "REPLICAS", "PORTS",
			"3/3",
			"1/3",
			"0/2",
			"5/5",
			"80->80/tcp",
		)
	})

	t.Run("truncation of service name and image", func(t *testing.T) {
		t.Parallel()
		view := RenderSwarm(SwarmContext{
			Swarm: monitor.SwarmSummary{
				IsActive: true,
				Services: services,
			},
			CursorIdx:    0,
			FormatStatus: func(_ bool) string { return "Active" },
			Width:        120,
		})

		// Name "log-collector-daemon-long-name" truncated to 18 chars
		if !strings.Contains(view, "log-collector-d...") {
			t.Errorf("expected truncated service name, got: %s", view)
		}
		// Image "fluentbit:latest-release-tag" truncated to 18 chars (15 + 3 dots)
		if !strings.Contains(view, "fluentbit:lates...") {
			t.Errorf("expected truncated service image, got: %s", view)
		}
	})
}

func TestRenderSwarm_WidthClamping(t *testing.T) {
	t.Parallel()

	swarm := monitor.SwarmSummary{
		IsActive: true,
		Nodes: []monitor.SwarmNode{
			{ID: "n1", Hostname: "host1"},
		},
		Services: []monitor.SwarmService{
			{ID: "s1", Name: "app1"},
		},
	}

	t.Run("narrow width clamps content divider to minimum 88 chars", func(t *testing.T) {
		t.Parallel()
		viewNarrow := RenderSwarm(SwarmContext{
			Swarm:        swarm,
			CursorIdx:    0,
			FormatStatus: func(_ bool) string { return "Active" },
			Width:        50,
		})

		expectedDivider := strings.Repeat("─", 88)
		if !strings.Contains(viewNarrow, expectedDivider) {
			t.Errorf("expected divider clamped to 88 chars, got: %s", viewNarrow)
		}
	})

	t.Run("wide width scales content divider", func(t *testing.T) {
		t.Parallel()
		viewWide := RenderSwarm(SwarmContext{
			Swarm:        swarm,
			CursorIdx:    0,
			FormatStatus: func(_ bool) string { return "Active" },
			Width:        140,
		})

		// contentWidth = max(140-4, 88) = 136
		expectedDivider := strings.Repeat("─", 136)
		if !strings.Contains(viewWide, expectedDivider) {
			t.Errorf("expected divider scaled to 136 chars, got: %s", viewWide)
		}
	})
}
