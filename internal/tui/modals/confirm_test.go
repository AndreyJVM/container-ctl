package modals

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
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

func TestRenderRemoveNetwork(t *testing.T) {
	// Unused network
	view := RenderRemoveNetwork("frontend", "net12345", 0)
	if !strings.Contains(view, "CONFIRM NETWORK REMOVAL") || !strings.Contains(view, "frontend (net12345)") {
		t.Errorf("expected network name, id and title, got: %s", view)
	}
	if !strings.Contains(view, "docker network rm") {
		t.Errorf("expected docker network rm warning, got: %s", view)
	}

	// In use network with container count warning
	viewInUse := RenderRemoveNetwork("backend", "net67890", 3)
	if !strings.Contains(viewInUse, "currently in use by 3 container(s)") {
		t.Errorf("expected in-use warning with count 3, got: %s", viewInUse)
	}
}

func TestRenderRemoveVolume(t *testing.T) {
	// Unused volume
	view := RenderRemoveVolume("pg_data", false)
	if !strings.Contains(view, "CONFIRM VOLUME REMOVAL") || !strings.Contains(view, "pg_data") {
		t.Errorf("expected volume name and title, got: %s", view)
	}
	if !strings.Contains(view, "persistent data will be permanently deleted") {
		t.Errorf("expected permanent deletion warning, got: %s", view)
	}

	// In use volume with force removal warning
	viewInUse := RenderRemoveVolume("app_data", true)
	if !strings.Contains(viewInUse, "Force removal (-f) will be applied") {
		t.Errorf("expected force removal warning, got: %s", viewInUse)
	}
}

func TestRenderRemoveContainerBatch(t *testing.T) {
	items := []monitor.ContainerItem{
		{ID: "c1", Name: "api-gateway"},
		{ID: "c2", Name: "auth-service"},
		{ID: "c3", Name: "postgres-db"},
	}

	view := RenderRemoveContainerBatch(items)
	if !strings.Contains(view, "CONFIRM REMOVAL OF 3 CONTAINERS") {
		t.Errorf("expected title with count 3, got: %s", view)
	}
	if !strings.Contains(view, "api-gateway") || !strings.Contains(view, "auth-service") || !strings.Contains(view, "postgres-db") {
		t.Errorf("expected container names in batch remove modal, got: %s", view)
	}
	if !strings.Contains(view, "permanently deleted") {
		t.Errorf("expected permanent deletion warning, got: %s", view)
	}
}
