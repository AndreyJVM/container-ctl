package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestRenderSwarmView(t *testing.T) {
	// Inactive
	inactView := RenderSwarm(SwarmContext{
		Swarm:        monitor.SwarmSummary{IsActive: false},
		FormatStatus: func(_ bool) string { return "Inactive" },
	})
	if !strings.Contains(inactView, "Host is not part of an active Swarm cluster.") {
		t.Errorf("expected swarm inactive notice, got: %s", inactView)
	}

	// Active
	actView := RenderSwarm(SwarmContext{
		Swarm: monitor.SwarmSummary{
			IsActive:      true,
			NodesCount:    3,
			ManagersCount: 1,
			Nodes: []monitor.SwarmNode{
				{ID: "node1", Hostname: "worker-1", Role: "worker", State: "ready", Availability: "active"},
			},
			Services: []monitor.SwarmService{
				{ID: "svc1", Name: "db-svc", Image: "postgres:16", Mode: "replicated", Replicas: "1/1", RunningReplicas: 1, TargetReplicas: 1},
			},
		},
		CursorIdx:    0,
		FormatStatus: func(_ bool) string { return "Active" },
		Width:        100,
	})
	if !strings.Contains(actView, "worker-1") || !strings.Contains(actView, "db-svc") {
		t.Errorf("expected swarm nodes and services in view, got: %s", actView)
	}
}
