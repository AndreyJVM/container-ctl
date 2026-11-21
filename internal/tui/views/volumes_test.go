package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
)

func TestRenderVolumes_EmptyAndErrorStates(t *testing.T) {
	t.Parallel()

	t.Run("daemon error banner suppresses table and shows error message", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Error: "volume daemon socket closed",
			Width: 80,
		})

		if !strings.Contains(view, "Docker volumes are unavailable: volume daemon socket closed") {
			t.Errorf("expected daemon error message, got: %s", view)
		}
		if strings.Contains(view, "DRIVER") || strings.Contains(view, "REFS") {
			t.Errorf("expected table header suppressed on error, got: %s", view)
		}
	})

	t.Run("empty list without search filter", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Volumes: nil,
			Width:   80,
		})

		if !strings.Contains(view, "DOCKER VOLUMES (0)") {
			t.Errorf("expected count 0 in header, got: %s", view)
		}
		if !strings.Contains(view, "No Docker volumes found.") {
			t.Errorf("expected no volumes notice, got: %s", view)
		}
	})

	t.Run("empty list with active filter query shows no matches and reset hint", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Volumes:           nil,
			TotalVolumesCount: 5,
			FilterQuery:       "backup_storage",
			Width:             80,
		})

		if !strings.Contains(view, "DOCKER VOLUMES (5)") {
			t.Errorf("expected total count 5 in header, got: %s", view)
		}
		if !strings.Contains(view, `No Docker volumes found matching "backup_storage".`) {
			t.Errorf("expected filter no match message, got: %s", view)
		}
		if !strings.Contains(view, "Press Esc to clear search filter.") {
			t.Errorf("expected Esc hint, got: %s", view)
		}
	})
}

func TestRenderVolumes_SearchAndFiltering(t *testing.T) {
	t.Parallel()

	t.Run("interactive filter typing mode", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Volumes: []monitor.DockerVolume{
				{Name: "data_vol", Driver: "local", Scope: "local"},
			},
			IsFiltering: true,
			FilterQuery: "data",
			Width:       100,
		})

		if !strings.Contains(view, "[Search: data█]") {
			t.Errorf("expected search prompt, got: %s", view)
		}
		if !strings.Contains(view, "(Enter: apply, Esc: cancel)") {
			t.Errorf("expected enter/esc hints, got: %s", view)
		}
	})

	t.Run("applied filter badge with matches count", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Volumes: []monitor.DockerVolume{
				{Name: "data_vol", Driver: "local", Scope: "local"},
			},
			TotalVolumesCount: 8,
			IsFiltering:       false,
			FilterQuery:       "data",
			Width:             100,
		})

		if !strings.Contains(view, "Filter: data") {
			t.Errorf("expected applied filter badge, got: %s", view)
		}
		if !strings.Contains(view, "(Found: 1 of 8, Esc: reset)") {
			t.Errorf("expected match count info, got: %s", view)
		}
	})
}

func TestRenderVolumes_TableAndCursorSelection(t *testing.T) {
	t.Parallel()

	vols := []monitor.DockerVolume{
		testutil.AVolume(func(v *monitor.DockerVolume) {
			v.Name = "alpha_vol"
			v.Driver = "local"
			v.Scope = "local"
			v.RefCount = 2
			v.Size = 4096
		}),
		testutil.AVolume(func(v *monitor.DockerVolume) {
			v.Name = "beta_vol"
			v.Driver = "local"
			v.Scope = "local"
			v.RefCount = 0
			v.Size = 8192
		}),
	}

	t.Run("cursor on first volume", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Volumes:   vols,
			CursorIdx: 0,
			Width:     100,
		})

		lines := strings.Split(view, "\n")
		var alphaLine, betaLine string
		for _, l := range lines {
			if strings.Contains(l, "alpha_vol") && !strings.Contains(l, "SELECTED") {
				alphaLine = l
			}
			if strings.Contains(l, "beta_vol") && !strings.Contains(l, "SELECTED") {
				betaLine = l
			}
		}

		if !strings.Contains(alphaLine, "► ") {
			t.Errorf("expected cursor pointer '► ' on alpha_vol, got: %q", alphaLine)
		}
		if strings.Contains(betaLine, "► ") {
			t.Errorf("did not expect cursor pointer on beta_vol, got: %q", betaLine)
		}
		if !strings.Contains(view, "SELECTED VOLUME DETAILS: alpha_vol") {
			t.Errorf("expected details card for alpha_vol, got: %s", view)
		}
	})

	t.Run("cursor on second volume", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Volumes:   vols,
			CursorIdx: 1,
			Width:     100,
		})

		lines := strings.Split(view, "\n")
		var alphaLine, betaLine string
		for _, l := range lines {
			if strings.Contains(l, "alpha_vol") && !strings.Contains(l, "SELECTED") {
				alphaLine = l
			}
			if strings.Contains(l, "beta_vol") && !strings.Contains(l, "SELECTED") {
				betaLine = l
			}
		}

		if strings.Contains(alphaLine, "► ") {
			t.Errorf("did not expect cursor pointer on alpha_vol, got: %q", alphaLine)
		}
		if !strings.Contains(betaLine, "► ") {
			t.Errorf("expected cursor pointer '► ' on beta_vol, got: %q", betaLine)
		}
		if !strings.Contains(view, "SELECTED VOLUME DETAILS: beta_vol") {
			t.Errorf("expected details card for beta_vol, got: %s", view)
		}
	})

	t.Run("cursor out of range suppresses details card", func(t *testing.T) {
		t.Parallel()
		view := RenderVolumes(VolumesContext{
			Volumes:   vols,
			CursorIdx: -1,
			Width:     100,
		})

		if strings.Contains(view, "► ") {
			t.Errorf("did not expect cursor pointer when cursor is -1, got: %s", view)
		}
		if strings.Contains(view, "SELECTED VOLUME DETAILS") {
			t.Errorf("did not expect details card when cursor is out of range, got: %s", view)
		}
	})
}

func TestRenderVolumes_DetailsCardAndUsageStatuses(t *testing.T) {
	t.Parallel()

	t.Run("in-use volume with positive reference count", func(t *testing.T) {
		t.Parallel()
		vol := testutil.AVolume(func(v *monitor.DockerVolume) {
			v.Name = "postgres_storage"
			v.Driver = "local"
			v.Scope = "local"
			v.Size = 1048576 // 1.0 MB
			v.RefCount = 3
			v.Mountpoint = "/var/lib/docker/volumes/postgres_storage/_data"
			v.ComposeProject = "database-stack"
		})

		view := RenderVolumes(VolumesContext{
			Volumes:   []monitor.DockerVolume{vol},
			CursorIdx: 0,
			Width:     110,
		})

		testutil.AssertContainsAll(t, view,
			"postgres_storage",
			"local / local",
			"1.0 MB",
			"● In Use (3 container reference(s))",
			"/var/lib/docker/volumes/postgres_storage/_data",
			"database-stack",
			"[y] Copy Name",
		)
		// Prune should NOT be in quick actions when in use
		if strings.Contains(view, "[C] Prune") {
			t.Errorf("did not expect '[C] Prune' action for in-use volume, got: %s", view)
		}
	})

	t.Run("unused reclaimable volume includes prune action", func(t *testing.T) {
		t.Parallel()
		vol := testutil.AVolume(func(v *monitor.DockerVolume) {
			v.Name = "old_dump_vol"
			v.RefCount = 0
			v.Size = 5242880 // 5.0 MB
		})

		view := RenderVolumes(VolumesContext{
			Volumes:   []monitor.DockerVolume{vol},
			CursorIdx: 0,
			Width:     100,
		})

		testutil.AssertContainsAll(t, view,
			"old_dump_vol",
			"5.0 MB",
			"○ Unused (reclaimable) — Press 'C' to prune",
			"[C] Prune",
			"[y] Copy Name",
		)
	})

	t.Run("unknown usage when refcount is negative and missing optional fields", func(t *testing.T) {
		t.Parallel()
		vol := testutil.AVolume(func(v *monitor.DockerVolume) {
			v.Name = "remote_gluster"
			v.Driver = "glusterfs"
			v.Scope = ""
			v.RefCount = -1
			v.Size = -1
			v.Mountpoint = ""
			v.ComposeProject = ""
		})

		view := RenderVolumes(VolumesContext{
			Volumes:   []monitor.DockerVolume{vol},
			CursorIdx: 0,
			Width:     100,
		})

		testutil.AssertContainsAll(t, view,
			"remote_gluster",
			"glusterfs / -",
			"Unknown",
		)
	})
}

func TestRenderVolumes_WidthClamping(t *testing.T) {
	t.Parallel()

	vol := testutil.AVolume(func(v *monitor.DockerVolume) {
		v.Name = "narrow_vol"
	})

	t.Run("narrow width clamps content to minimum 68 chars", func(t *testing.T) {
		t.Parallel()
		viewNarrow := RenderVolumes(VolumesContext{
			Volumes:   []monitor.DockerVolume{vol},
			CursorIdx: 0,
			Width:     30,
		})

		expectedLine := strings.Repeat("─", 68)
		if !strings.Contains(viewNarrow, expectedLine) {
			t.Errorf("expected table divider clamped to 68 chars, got: %s", viewNarrow)
		}
	})

	t.Run("wide width scales table divider", func(t *testing.T) {
		t.Parallel()
		viewWide := RenderVolumes(VolumesContext{
			Volumes:   []monitor.DockerVolume{vol},
			CursorIdx: 0,
			Width:     120,
		})

		// contentWidth = max(120-4, 68) = 116
		expectedLine := strings.Repeat("─", 116)
		if !strings.Contains(viewWide, expectedLine) {
			t.Errorf("expected table divider scaled to 116 chars, got: %s", viewWide)
		}
	})
}
