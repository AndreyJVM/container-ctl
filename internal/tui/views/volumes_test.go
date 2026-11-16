package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestRenderVolumesView(t *testing.T) {
	dict := i18n.Get()
	view := RenderVolumes(VolumesContext{
		Volumes: []monitor.DockerVolume{{
			Name:           "shop_data",
			Driver:         "local",
			Scope:          "local",
			Mountpoint:     "/var/lib/docker/volumes/shop_data/_data",
			Size:           4096,
			RefCount:       2,
			ComposeProject: "shop",
		}},
		CursorIdx: 0,
		Width:     100,
		I18n:      &dict,
	})

	for _, expected := range []string{"shop_data", "4.0 KB", "/var/lib/docker/volumes/shop_data/_data", "shop"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in volumes view, got: %s", expected, view)
		}
	}

	errorView := RenderVolumes(VolumesContext{Error: "daemon unavailable", I18n: &dict})
	if !strings.Contains(errorView, "daemon unavailable") {
		t.Errorf("expected discovery error in volumes view, got: %s", errorView)
	}

	emptyView := RenderVolumes(VolumesContext{Volumes: nil, I18n: &dict})
	if !strings.Contains(emptyView, dict.NoVolumes) {
		t.Errorf("expected no volumes notice, got: %s", emptyView)
	}
}
