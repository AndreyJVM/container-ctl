package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/testutil"
)

func TestRenderNetworksView(t *testing.T) {
	net := testutil.ANetwork(func(n *monitor.DockerNetwork) {
		n.ID = "abc123456789"
		n.Name = "frontend"
		n.Containers = 2
		n.Subnet = "172.20.0.0/16"
		n.Gateway = "172.20.0.1"
		n.ComposeProject = "shop"
	})
	view := RenderNetworks(NetworksContext{
		Networks:  []monitor.DockerNetwork{net},
		CursorIdx: 0,
		Width:     100,
	})

	for _, expected := range []string{"frontend", "bridge", "172.20.0.0/16", "172.20.0.1", "shop", "Driver / Scope:", "2 container(s) connected"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in networks view, got: %s", expected, view)
		}
	}

	// Unused network
	unusedNet := testutil.ANetwork(func(n *monitor.DockerNetwork) {
		n.Name = "custom-bridge"
		n.Containers = 0
	})
	unusedView := RenderNetworks(NetworksContext{
		Networks:  []monitor.DockerNetwork{unusedNet},
		CursorIdx: 0,
		Width:     100,
	})
	if !strings.Contains(unusedView, "0 containers connected (unused)") {
		t.Errorf("expected 0 containers connected notice, got: %s", unusedView)
	}

	errorView := RenderNetworks(NetworksContext{Error: "daemon unavailable"})
	if !strings.Contains(errorView, "daemon unavailable") {
		t.Errorf("expected discovery error in networks view, got: %s", errorView)
	}

	emptyView := RenderNetworks(NetworksContext{Networks: nil})
	if !strings.Contains(emptyView, "No Docker networks found.") {
		t.Errorf("expected no networks notice, got: %s", emptyView)
	}
}
