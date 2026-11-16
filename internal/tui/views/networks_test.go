package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestRenderNetworksView(t *testing.T) {
	dict := i18n.Get()
	view := RenderNetworks(NetworksContext{
		Networks: []monitor.DockerNetwork{{
			ID:             "abc123456789",
			Name:           "frontend",
			Driver:         "bridge",
			Scope:          "local",
			Containers:     2,
			Subnet:         "172.20.0.0/16",
			Gateway:        "172.20.0.1",
			Attachable:     true,
			ComposeProject: "shop",
		}},
		CursorIdx: 0,
		Width:     100,
		I18n:      &dict,
	})

	for _, expected := range []string{"frontend", "bridge", "172.20.0.0/16", "172.20.0.1", "shop"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in networks view, got: %s", expected, view)
		}
	}

	errorView := RenderNetworks(NetworksContext{Error: "daemon unavailable", I18n: &dict})
	if !strings.Contains(errorView, "daemon unavailable") {
		t.Errorf("expected discovery error in networks view, got: %s", errorView)
	}

	emptyView := RenderNetworks(NetworksContext{Networks: nil, I18n: &dict})
	if !strings.Contains(emptyView, dict.NoNetworks) {
		t.Errorf("expected no networks notice, got: %s", emptyView)
	}
}
