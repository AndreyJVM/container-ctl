package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestRenderImagesView(t *testing.T) {
	dict := i18n.Get()
	view := RenderImages(ImagesContext{
		Images: []monitor.DockerImage{
			{
				ID:         "c557bd415052",
				FullID:     "sha256:c557bd415052858081f023e8d17500073317df437ae79ea4d2c4077cd9a1a6ae",
				Repository: "sambryx-web",
				Tag:        "latest (+1)",
				AllTags:    []string{"sambryx-web:latest", "sambryx-web:v1.0"},
				Size:       359214108,
				Created:    "2026-11-16 15:23",
				Containers: 1,
				Digest:     "1234567890ab",
				InUse:      true,
			},
			{
				ID:         "a857ab57fe2d",
				FullID:     "sha256:a857ab57fe2d98a0a33432586f16a6b0d2e256ed9d915ec56ecfaaeb81ac36bd",
				Repository: "<none>",
				Tag:        "<none>",
				Size:       1048576,
				Created:    "2026-11-10 12:46",
				Containers: 0,
				Dangling:   true,
			},
		},
		CursorIdx: 0,
		Width:     100,
		I18n:      &dict,
	})

	for _, expected := range []string{"c557bd415052", "sambryx-web", "latest (+1)", "DOCKER IMAGES", dict.ImageInUseBadge, dict.ImageDanglingBadge} {
		if !strings.Contains(view, expected) {
			t.Errorf("expected %q in images view, got: %s", expected, view)
		}
	}

	// Details card for selected image
	if !strings.Contains(view, dict.ImageDetailsTitle) || !strings.Contains(view, "sambryx-web:latest, sambryx-web:v1.0") {
		t.Errorf("expected image details card in images view, got: %s", view)
	}

	// Error view
	errorView := RenderImages(ImagesContext{Error: "daemon unavailable", I18n: &dict})
	if !strings.Contains(errorView, "daemon unavailable") {
		t.Errorf("expected discovery error in images view, got: %s", errorView)
	}

	// Compact width vs wide width adaptive rendering
	compactView := RenderImages(ImagesContext{
		Images: []monitor.DockerImage{{ID: "c557bd415052", Repository: "sambryx-web", Tag: "latest"}},
		Width:  80,
		I18n:   &dict,
	})
	wideView := RenderImages(ImagesContext{
		Images: []monitor.DockerImage{{ID: "c557bd415052", Repository: "sambryx-web", Tag: "latest"}},
		Width:  160,
		I18n:   &dict,
	})
	if !strings.Contains(compactView, "sambryx-web") || !strings.Contains(wideView, "sambryx-web") {
		t.Errorf("expected repository in both compact and wide views")
	}

	// Empty view
	emptyView := RenderImages(ImagesContext{I18n: &dict})
	if !strings.Contains(emptyView, dict.NoImages) {
		t.Errorf("expected no images notice in images view, got: %s", emptyView)
	}
}
