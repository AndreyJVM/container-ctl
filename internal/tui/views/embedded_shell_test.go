package views

import (
	"strings"
	"testing"
)

func TestRenderEmbeddedShellView(t *testing.T) {
	// Unfocused with default starting output
	viewUnf := RenderEmbeddedShell(EmbeddedShellContext{
		ContainerName: "my-web-app",
		Output:        "",
		Width:         80,
		Height:        20,
		Focused:       false,
	})
	if !strings.Contains(viewUnf, "my-web-app") || !strings.Contains(viewUnf, "DOCKPULSE FOCUSED") {
		t.Errorf("expected unfocused shell title and status, got: %s", viewUnf)
	}
	if !strings.Contains(viewUnf, "Starting shell...") {
		t.Errorf("expected shell starting message, got: %s", viewUnf)
	}

	// Focused with output
	viewFoc := RenderEmbeddedShell(EmbeddedShellContext{
		ContainerName: "redis-db",
		Output:        "root@container:/# ls -la",
		Width:         100,
		Height:        25,
		Focused:       true,
	})
	if !strings.Contains(viewFoc, "redis-db") || !strings.Contains(viewFoc, "FOCUSED") {
		t.Errorf("expected focused shell title and status, got: %s", viewFoc)
	}
	if !strings.Contains(viewFoc, "root@container") {
		t.Errorf("expected shell output in view, got: %s", viewFoc)
	}
}
