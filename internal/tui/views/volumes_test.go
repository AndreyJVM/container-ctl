package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
)

func TestRenderVolumesView(t *testing.T) {
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
	})

	for _, expected := range []string{"shop_data", "4.0 KB", "/var/lib/docker/volumes/shop_data/_data", "shop", "In Use", "Driver / Scope:"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in volumes view, got: %s", expected, view)
		}
	}

	// Reclaimable volume (refcount == 0)
	unusedVol := testutil.AVolume(func(v *monitor.DockerVolume) {
		v.Name = "old_backup"
		v.RefCount = 0
	})
	unusedView := RenderVolumes(VolumesContext{
		Volumes:   []monitor.DockerVolume{unusedVol},
		CursorIdx: 0,
		Width:     100,
	})
	if !strings.Contains(unusedView, "Unused (reclaimable)") || !strings.Contains(unusedView, "Prune") {
		t.Errorf("expected reclaimable volume notice and prune hint, got: %s", unusedView)
	}

	errorView := RenderVolumes(VolumesContext{Error: "daemon unavailable"})
	if !strings.Contains(errorView, "daemon unavailable") {
		t.Errorf("expected discovery error in volumes view, got: %s", errorView)
	}

	emptyView := RenderVolumes(VolumesContext{Volumes: nil})
	if !strings.Contains(emptyView, "No Docker volumes found.") {
		t.Errorf("expected no volumes notice, got: %s", emptyView)
	}
}
