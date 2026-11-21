package modals

import (
	"strings"
	"testing"
)

func TestRenderExec(t *testing.T) {
	// 1. Menu view
	menuView := RenderExec(ExecModalContext{
		ContainerName: "my-web",
		ContainerID:   "c999",
		CursorIdx:     0,
		IsCustomInput: false,
	})

	if !strings.Contains(menuView, "my-web") || !strings.Contains(menuView, "c999") {
		t.Errorf("expected container name and id in exec modal, got: %s", menuView)
	}
	if !strings.Contains(menuView, "Embedded Bash / Sh") || !strings.Contains(menuView, "Docker Attach") {
		t.Errorf("expected modes in exec modal, got: %s", menuView)
	}

	// 2. Custom input view
	customView := RenderExec(ExecModalContext{
		ContainerName: "my-web",
		ContainerID:   "c999",
		IsCustomInput: true,
		CustomCmd:     "htop",
	})

	if !strings.Contains(customView, "htop") {
		t.Errorf("expected custom command in custom input view, got: %s", customView)
	}
}

func TestRenderExecOffersEmbeddedAndExternalShells(t *testing.T) {
	view := RenderExec(ExecModalContext{
		ContainerName: "api",
		ContainerID:   "c123",
	})

	if !strings.Contains(view, "Embedded Bash / Sh") {
		t.Fatalf("expected the recommended embedded shell option, got: %s", view)
	}
	if !strings.Contains(view, "External Bash / Sh") {
		t.Fatalf("expected an external shell fallback, got: %s", view)
	}
}
