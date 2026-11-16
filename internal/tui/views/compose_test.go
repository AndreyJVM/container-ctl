package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestRenderComposeView(t *testing.T) {
	dict := i18n.Get()

	// Empty
	emptyView := RenderCompose(ComposeContext{Projects: nil, I18n: &dict})
	if !strings.Contains(emptyView, dict.NoComposeProjects) {
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
				{Name: "web", State: "running", Image: "nginx:alpine", Ports: "80:80"},
			},
		},
	}
	view := RenderCompose(ComposeContext{
		Projects:     projs,
		CursorIdx:    0,
		FormatStatus: func(s, _ string) string { return s },
		Width:        100,
		I18n:         &dict,
	})
	if !strings.Contains(view, "web-stack") || !strings.Contains(view, "nginx:alpine") {
		t.Errorf("expected project details in compose view, got: %s", view)
	}
}
