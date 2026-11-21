package modals

import (
	"strings"
	"testing"
)

func TestRenderComposeDown(t *testing.T) {
	view := RenderComposeDown("my-stack")
	if !strings.Contains(view, "my-stack") || !strings.Contains(view, "CONFIRM DOCKER COMPOSE DOWN") {
		t.Errorf("expected project name and title in compose down modal, got: %s", view)
	}
}

func TestRenderRemoveContainer(t *testing.T) {
	view := RenderRemoveContainer("my-db", "c98765")
	if !strings.Contains(view, "my-db") || !strings.Contains(view, "c98765") || !strings.Contains(view, "CONFIRM CONTAINER REMOVAL") {
		t.Errorf("expected container name, id and title in remove modal, got: %s", view)
	}
}

func TestRenderRemoveImage(t *testing.T) {
	// Normal unused image
	view := RenderRemoveImage("nginx:alpine", "img12345", false)
	if !strings.Contains(view, "nginx:alpine") || !strings.Contains(view, "img12345") || !strings.Contains(view, "CONFIRM IMAGE REMOVAL") {
		t.Errorf("expected image name, id and title in remove image modal, got: %s", view)
	}

	// In use image with warning
	viewInUse := RenderRemoveImage("redis:latest", "img67890", true)
	if !strings.Contains(viewInUse, "Warning: Image is currently in use by containers") {
		t.Errorf("expected in-use warning in modal, got: %s", viewInUse)
	}
}
