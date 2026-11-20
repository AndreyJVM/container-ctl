package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestRenderComposeView(t *testing.T) {
	// Empty
	emptyView := RenderCompose(ComposeContext{Projects: nil})
	if !strings.Contains(emptyView, "No Docker Compose projects detected") {
		t.Errorf("expected empty compose message, got: %s", emptyView)
	}

	// With projects
	projs := []monitor.ComposeProject{
		{
			Name:            "web-stack",
			Status:          "running",
			TotalServices:   1,
			RunningServices: 1,
			Services: []monitor.ComposeService{
				{Name: "web", State: "running", Image: "nginx:alpine", Ports: "0.0.0.0:8080->80/tcp, [::]:8080->80/tcp"},
			},
		},
	}
	view := RenderCompose(ComposeContext{
		Projects:     projs,
		CursorIdx:    0,
		FormatStatus: func(s, _ string) string { return s },
		Width:        100,
	})
	if !strings.Contains(view, "web-stack") || !strings.Contains(view, "nginx:alpine") {
		t.Errorf("expected project details in compose view, got: %s", view)
	}
	if !strings.Contains(view, "8080->80/tcp") || strings.Contains(view, "0.0.0.0:") {
		t.Errorf("expected cleaned port without 0.0.0.0 in compose view, got: %s", view)
	}
}
