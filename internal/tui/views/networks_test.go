package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
)

func TestRenderNetworks_EmptyAndErrorStates(t *testing.T) {
	t.Parallel()

	t.Run("daemon error banner suppresses table and shows error message", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Error: "docker daemon connection refused",
			Width: 80,
		})

		if !strings.Contains(view, "Docker networks are unavailable: docker daemon connection refused") {
			t.Errorf("expected daemon error message, got: %s", view)
		}
		if strings.Contains(view, "DRIVER") || strings.Contains(view, "CONTAINERS") {
			t.Errorf("expected table header suppressed on error, got: %s", view)
		}
	})

	t.Run("empty list without search filter", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Networks: nil,
			Width:    80,
		})

		if !strings.Contains(view, "DOCKER NETWORKS (0)") {
			t.Errorf("expected count 0 in header, got: %s", view)
		}
		if !strings.Contains(view, "No Docker networks found.") {
			t.Errorf("expected no networks notice, got: %s", view)
		}
	})

	t.Run("empty list with active filter query shows no matches and reset hint", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Networks:           nil,
			TotalNetworksCount: 4,
			FilterQuery:        "custom-overlay",
			Width:              80,
		})

		if !strings.Contains(view, "DOCKER NETWORKS (4)") {
			t.Errorf("expected total count 4 in header, got: %s", view)
		}
		if !strings.Contains(view, `No Docker networks found matching "custom-overlay".`) {
			t.Errorf("expected filter no match message, got: %s", view)
		}
		if !strings.Contains(view, "Press Esc to clear search filter.") {
			t.Errorf("expected Esc hint, got: %s", view)
		}
	})
}

func TestRenderNetworks_SearchAndFiltering(t *testing.T) {
	t.Parallel()

	t.Run("interactive filter typing mode", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Networks: []monitor.DockerNetwork{
				{ID: "net1", Name: "bridge-net", Driver: "bridge"},
			},
			IsFiltering: true,
			FilterQuery: "bri",
			Width:       100,
		})

		if !strings.Contains(view, "[Search: bri█]") {
			t.Errorf("expected search prompt, got: %s", view)
		}
		if !strings.Contains(view, "(Enter: apply, Esc: cancel)") {
			t.Errorf("expected enter/esc hints, got: %s", view)
		}
	})

	t.Run("applied filter badge with matches count", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Networks: []monitor.DockerNetwork{
				{ID: "net1", Name: "bridge-net", Driver: "bridge"},
			},
			TotalNetworksCount: 6,
			IsFiltering:        false,
			FilterQuery:        "bridge",
			Width:              100,
		})

		if !strings.Contains(view, "Filter: bridge") {
			t.Errorf("expected applied filter badge, got: %s", view)
		}
		if !strings.Contains(view, "(Found: 1 of 6, Esc: reset)") {
			t.Errorf("expected match count info, got: %s", view)
		}
	})
}

func TestRenderNetworks_TableAndCursorSelection(t *testing.T) {
	t.Parallel()

	nets := []monitor.DockerNetwork{
		testutil.ANetwork(func(n *monitor.DockerNetwork) {
			n.ID = "id-net-aaa"
			n.Name = "alpha-net"
			n.Driver = "bridge"
			n.Containers = 3
			n.Subnet = "172.18.0.0/16"
		}),
		testutil.ANetwork(func(n *monitor.DockerNetwork) {
			n.ID = "id-net-bbb"
			n.Name = "beta-net"
			n.Driver = "overlay"
			n.Containers = 0
			n.Subnet = "10.0.0.0/24"
		}),
	}

	t.Run("cursor on first network", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Networks:  nets,
			CursorIdx: 0,
			Width:     100,
		})

		lines := strings.Split(view, "\n")
		var alphaLine, betaLine string
		for _, l := range lines {
			if strings.Contains(l, "alpha-net") && !strings.Contains(l, "SELECTED") {
				alphaLine = l
			}
			if strings.Contains(l, "beta-net") && !strings.Contains(l, "SELECTED") {
				betaLine = l
			}
		}

		if !strings.Contains(alphaLine, "► ") {
			t.Errorf("expected cursor pointer '► ' on alpha-net, got: %q", alphaLine)
		}
		if strings.Contains(betaLine, "► ") {
			t.Errorf("did not expect cursor pointer on beta-net, got: %q", betaLine)
		}
		if !strings.Contains(view, "SELECTED NETWORK DETAILS: alpha-net") {
			t.Errorf("expected details card for alpha-net, got: %s", view)
		}
	})

	t.Run("cursor on second network", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Networks:  nets,
			CursorIdx: 1,
			Width:     100,
		})

		lines := strings.Split(view, "\n")
		var alphaLine, betaLine string
		for _, l := range lines {
			if strings.Contains(l, "alpha-net") && !strings.Contains(l, "SELECTED") {
				alphaLine = l
			}
			if strings.Contains(l, "beta-net") && !strings.Contains(l, "SELECTED") {
				betaLine = l
			}
		}

		if strings.Contains(alphaLine, "► ") {
			t.Errorf("did not expect cursor pointer on alpha-net, got: %q", alphaLine)
		}
		if !strings.Contains(betaLine, "► ") {
			t.Errorf("expected cursor pointer '► ' on beta-net, got: %q", betaLine)
		}
		if !strings.Contains(view, "SELECTED NETWORK DETAILS: beta-net") {
			t.Errorf("expected details card for beta-net, got: %s", view)
		}
	})

	t.Run("cursor out of range suppresses details card", func(t *testing.T) {
		t.Parallel()
		view := RenderNetworks(NetworksContext{
			Networks:  nets,
			CursorIdx: -1,
			Width:     100,
		})

		if strings.Contains(view, "► ") {
			t.Errorf("did not expect cursor pointer when cursor is -1, got: %s", view)
		}
		if strings.Contains(view, "SELECTED NETWORK DETAILS") {
			t.Errorf("did not expect details card when cursor is out of range, got: %s", view)
		}
	})
}

func TestRenderNetworks_DetailsCardFields(t *testing.T) {
	t.Parallel()

	t.Run("connected containers, external flag and compose project", func(t *testing.T) {
		t.Parallel()
		net := testutil.ANetwork(func(n *monitor.DockerNetwork) {
			n.ID = "net123456789"
			n.Name = "prod-tier"
			n.Driver = "bridge"
			n.Scope = "local"
			n.Subnet = "192.168.1.0/24"
			n.Gateway = "192.168.1.1"
			n.Containers = 5
			n.Internal = false
			n.Attachable = true
			n.ComposeProject = "ecommerce"
		})

		view := RenderNetworks(NetworksContext{
			Networks:  []monitor.DockerNetwork{net},
			CursorIdx: 0,
			Width:     110,
		})

		testutil.AssertContainsAll(t, view,
			"prod-tier",
			"bridge / local",
			"192.168.1.0/24",
			"192.168.1.1",
			"5 container(s) connected",
			"external",
			"attachable",
			"ecommerce",
			"[y] Copy Name",
			"[C] Prune Unused",
		)
	})

	t.Run("unused network, internal isolated flag and missing optional fields", func(t *testing.T) {
		t.Parallel()
		net := testutil.ANetwork(func(n *monitor.DockerNetwork) {
			n.ID = "net999"
			n.Name = "isolated-sec"
			n.Driver = "overlay"
			n.Scope = ""
			n.Subnet = ""
			n.Gateway = ""
			n.Containers = 0
			n.Internal = true
			n.Attachable = false
			n.ComposeProject = ""
		})

		view := RenderNetworks(NetworksContext{
			Networks:  []monitor.DockerNetwork{net},
			CursorIdx: 0,
			Width:     100,
		})

		testutil.AssertContainsAll(t, view,
			"isolated-sec",
			"0 containers connected (unused)",
			"internal (isolated)",
		)
		if strings.Contains(view, "external") {
			t.Errorf("did not expect 'external' flag when network is internal, got: %s", view)
		}
		if strings.Contains(view, "attachable") {
			t.Errorf("did not expect 'attachable' flag when network is not attachable, got: %s", view)
		}
	})
}

func TestRenderNetworks_WidthClamping(t *testing.T) {
	t.Parallel()

	net := testutil.ANetwork(func(n *monitor.DockerNetwork) {
		n.ID = "narrow-net-123"
		n.Name = "narrow-net"
	})

	t.Run("narrow width clamps content to minimum 76 chars", func(t *testing.T) {
		t.Parallel()
		viewNarrow := RenderNetworks(NetworksContext{
			Networks:  []monitor.DockerNetwork{net},
			CursorIdx: 0,
			Width:     40,
		})

		expectedLine := strings.Repeat("─", 76)
		if !strings.Contains(viewNarrow, expectedLine) {
			t.Errorf("expected table divider clamped to 76 chars, got: %s", viewNarrow)
		}
	})

	t.Run("wide width scales table divider", func(t *testing.T) {
		t.Parallel()
		viewWide := RenderNetworks(NetworksContext{
			Networks:  []monitor.DockerNetwork{net},
			CursorIdx: 0,
			Width:     120,
		})

		// contentWidth = max(120-4, 76) = 116
		expectedLine := strings.Repeat("─", 116)
		if !strings.Contains(viewWide, expectedLine) {
			t.Errorf("expected table divider scaled to 116 chars, got: %s", viewWide)
		}
	})
}

func TestRenderNetworks_HideDetails(t *testing.T) {
	t.Parallel()

	net := testutil.ANetwork(func(n *monitor.DockerNetwork) {
		n.ID = "net-details-123"
		n.Name = "production-net"
	})

	// 1. Details visible
	viewVisible := RenderNetworks(NetworksContext{
		Networks:    []monitor.DockerNetwork{net},
		CursorIdx:   0,
		Width:       100,
		HideDetails: false,
	})
	if !strings.Contains(viewVisible, "SELECTED NETWORK DETAILS") {
		t.Errorf("expected details card when HideDetails is false")
	}

	// 2. Details hidden
	viewHidden := RenderNetworks(NetworksContext{
		Networks:    []monitor.DockerNetwork{net},
		CursorIdx:   0,
		Width:       100,
		HideDetails: true,
	})
	if strings.Contains(viewHidden, "SELECTED NETWORK DETAILS") {
		t.Errorf("expected details card to be hidden when HideDetails is true")
	}
	if !strings.Contains(viewHidden, "production-net") {
		t.Errorf("expected network row to still be rendered when HideDetails is true")
	}
}
