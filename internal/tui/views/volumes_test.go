package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/testutil"
)

func TestRenderVolumesView(t *testing.T) {
	dict := i18n.Get()
	vol := testutil.AVolume(func(v *monitor.DockerVolume) {
		v.Name = "shop_data"
		v.Mountpoint = "/var/lib/docker/volumes/shop_data/_data"
		v.Size = 4096
		v.RefCount = 2
		v.ComposeProject = "shop"
	})
	view := RenderVolumes(VolumesContext{
		Volumes:   []monitor.DockerVolume{vol},
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
