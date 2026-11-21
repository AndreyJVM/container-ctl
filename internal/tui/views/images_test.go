package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
)

func TestRenderImagesView(t *testing.T) {
	view := RenderImages(ImagesContext{
		Images: []monitor.DockerImage{
			testutil.AnImage(
				testutil.WithImageID("c557bd415052", "sha256:c557bd415052858081f023e8d17500073317df437ae79ea4d2c4077cd9a1a6ae"),
				testutil.WithRepoTag("sambryx-web", "latest (+1)"),
				testutil.WithAllTags("sambryx-web:latest", "sambryx-web:v1.0"),
				testutil.WithImageStatus(true, false, 1),
			),
			testutil.AnImage(
				testutil.WithImageID("a857ab57fe2d", "sha256:a857ab57fe2d98a0a33432586f16a6b0d2e256ed9d915ec56ecfaaeb81ac36bd"),
				testutil.WithRepoTag("<none>", "<none>"),
				testutil.WithImageStatus(false, true, 0),
			),
		},
		CursorIdx: 0,
		Width:     100,
	})

	for _, expected := range []string{"c557bd415052", "sambryx-web", "latest (+1)", "DOCKER IMAGES", "[In Use]", "[Dangling]"} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in images view, got: %s", expected, view)
		}
	}

	// Details card for selected image
	if !strings.Contains(view, "SELECTED IMAGE DETAILS") || !strings.Contains(view, "sambryx-web:latest, sambryx-web:v1.0") || !strings.Contains(view, "Image Size:") {
		t.Errorf("expected image details card with Image Size in images view, got: %s", view)
	}

	// In Use and Reclaimable summary in header
	if !strings.Contains(view, "In Use:") || !strings.Contains(view, "Reclaimable:") {
		t.Errorf("expected In Use and Reclaimable summary in images view, got: %s", view)
	}

	// Action row in details card
	if !strings.Contains(view, "Status / Action:") {
		t.Errorf("expected status/action row in images view, got: %s", view)
	}

	// OnlyUnused filter mode
	unusedOnlyView := RenderImages(ImagesContext{
		Images: []monitor.DockerImage{
			testutil.AnImage(testutil.WithImageID("a857ab57fe2d", ""), testutil.WithImageStatus(false, false, 0)),
		},
		OnlyUnused: true,
		Width:      100,
	})
	if !strings.Contains(unusedOnlyView, "[Filter: Unused Images Only — press 'u' or 'Esc' to show all]") {
		t.Errorf("expected unused badge in unused only view, got: %s", unusedOnlyView)
	}
	if !strings.Contains(unusedOnlyView, "Unused (reclaimable) — Press 'x' to delete, 'C' to prune") {
		t.Errorf("expected reclaimable action hint for unused image, got: %s", unusedOnlyView)
	}

	// Error view
	errorView := RenderImages(ImagesContext{Error: "daemon unavailable"})
	if !strings.Contains(errorView, "daemon unavailable") {
		t.Errorf("expected discovery error in images view, got: %s", errorView)
	}

	// Compact width vs wide width adaptive rendering
	compactView := RenderImages(ImagesContext{
		Images: []monitor.DockerImage{{ID: "c557bd415052", Repository: "sambryx-web", Tag: "latest"}},
		Width:  80,
	})
	wideView := RenderImages(ImagesContext{
		Images: []monitor.DockerImage{{ID: "c557bd415052", Repository: "sambryx-web", Tag: "latest"}},
		Width:  160,
	})
	if !strings.Contains(compactView, "sambryx-web") || !strings.Contains(wideView, "sambryx-web") {
		t.Errorf("expected repository in both compact and wide views")
	}

	// Empty view
	emptyView := RenderImages(ImagesContext{})
	if !strings.Contains(emptyView, "No Docker images found.") {
		t.Errorf("expected no images notice in images view, got: %s", emptyView)
	}
}
