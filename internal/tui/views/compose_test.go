package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestRenderComposeView_EmptyStates(t *testing.T) {
	t.Parallel()

	t.Run("empty without filter displays missing label message", func(t *testing.T) {
		t.Parallel()
		view := RenderCompose(ComposeContext{Projects: nil})
		if !strings.Contains(view, "No Docker Compose projects detected (missing com.docker.compose.project labels).") {
			t.Errorf("expected empty message, got: %s", view)
		}
		if !strings.Contains(view, "DOCKER COMPOSE PROJECTS (0)") {
			t.Errorf("expected header with count 0, got: %s", view)
		}
	})

	t.Run("empty with active search query displays no matches message", func(t *testing.T) {
		t.Parallel()
		view := RenderCompose(ComposeContext{
			Projects:           nil,
			TotalProjectsCount: 4,
			FilterQuery:        "nonexistent",
		})
		if !strings.Contains(view, `No Compose projects found matching "nonexistent".`) {
			t.Errorf("expected search no match message, got: %s", view)
		}
		if !strings.Contains(view, "Press Esc to clear search filter.") {
			t.Errorf("expected esc hint, got: %s", view)
		}
		if !strings.Contains(view, "DOCKER COMPOSE PROJECTS (4)") {
			t.Errorf("expected header with total count 4, got: %s", view)
		}
	})
}

func TestRenderComposeView_FilteringHeaders(t *testing.T) {
	t.Parallel()

	t.Run("typing search query displays interactive search prompt", func(t *testing.T) {
		t.Parallel()
		view := RenderCompose(ComposeContext{
			Projects: []monitor.ComposeProject{
				{Name: "web-stack"},
			},
			IsFiltering:  true,
			FilterQuery:  "web",
			FormatStatus: func(s, _ string) string { return s },
		})
		if !strings.Contains(view, "[Search: web█]") {
			t.Errorf("expected search prompt in view, got: %s", view)
		}
		if !strings.Contains(view, "(Enter: apply, Esc: cancel)") {
			t.Errorf("expected filter apply/cancel hints, got: %s", view)
		}
	})

	t.Run("applied filter displays badge with match counts", func(t *testing.T) {
		t.Parallel()
		view := RenderCompose(ComposeContext{
			Projects: []monitor.ComposeProject{
				{Name: "web-stack"},
			},
			TotalProjectsCount: 3,
			IsFiltering:        false,
			FilterQuery:        "web",
			FormatStatus:       func(s, _ string) string { return s },
		})
		if !strings.Contains(view, "Filter: web") {
			t.Errorf("expected active filter badge, got: %s", view)
		}
		if !strings.Contains(view, "(Found: 1 of 3, Esc: reset)") {
			t.Errorf("expected found count in filter badge, got: %s", view)
		}
	})
}

func TestRenderComposeView_ProjectsAndCursorSelection(t *testing.T) {
	t.Parallel()

	projects := []monitor.ComposeProject{
		{
			Name:            "frontend-stack",
			Status:          "running",
			StatusDisplay:   "Running",
			TotalServices:   2,
			RunningServices: 2,
		},
		{
			Name:            "backend-stack",
			Status:          "partially_running",
			StatusDisplay:   "Partial",
			TotalServices:   3,
			RunningServices: 1,
		},
	}

	mockFormatStatus := func(s, disp string) string {
		if disp != "" {
			return disp
		}
		return s
	}

	t.Run("cursor on first project", func(t *testing.T) {
		t.Parallel()
		view := RenderCompose(ComposeContext{
			Projects:     projects,
			CursorIdx:    0,
			FormatStatus: mockFormatStatus,
			Width:        100,
		})

		lines := strings.Split(view, "\n")
		var frontendLine, backendLine string
		for _, l := range lines {
			if strings.Contains(l, "frontend-stack") {
				frontendLine = l
			}
			if strings.Contains(l, "backend-stack") {
				backendLine = l
			}
		}

		if !strings.Contains(frontendLine, "► ") {
			t.Errorf("expected cursor pointer '► ' on frontend-stack, got line: %q", frontendLine)
		}
		if strings.Contains(backendLine, "► ") {
			t.Errorf("expected no cursor pointer on backend-stack, got line: %q", backendLine)
		}
		if !strings.Contains(frontendLine, "[Running]  (Services up: 2/2)") {
			t.Errorf("expected running services 2/2 in frontend line, got: %q", frontendLine)
		}
		if !strings.Contains(backendLine, "[Partial]  (Services up: 1/3)") {
			t.Errorf("expected partial services 1/3 in backend line, got: %q", backendLine)
		}
	})

	t.Run("cursor on second project", func(t *testing.T) {
		t.Parallel()
		view := RenderCompose(ComposeContext{
			Projects:     projects,
			CursorIdx:    1,
			FormatStatus: mockFormatStatus,
			Width:        100,
		})

		lines := strings.Split(view, "\n")
		var frontendLine, backendLine string
		for _, l := range lines {
			if strings.Contains(l, "frontend-stack") {
				frontendLine = l
			}
			if strings.Contains(l, "backend-stack") {
				backendLine = l
			}
		}

		if strings.Contains(frontendLine, "► ") {
			t.Errorf("expected no cursor on frontend-stack, got line: %q", frontendLine)
		}
		if !strings.Contains(backendLine, "► ") {
			t.Errorf("expected cursor pointer '► ' on backend-stack, got line: %q", backendLine)
		}
	})
}

func TestRenderComposeView_ServicesAndPorts(t *testing.T) {
	t.Parallel()

	projs := []monitor.ComposeProject{
		{
			Name:            "microservices",
			Status:          "running",
			TotalServices:   2,
			RunningServices: 1,
			Services: []monitor.ComposeService{
				{
					Name:  "api-gateway",
					State: "running",
					Image: "traefik:v3.1.2-alpine-long-image-name",
					Ports: "0.0.0.0:80->80/tcp, [::]:80->80/tcp, 0.0.0.0:443->443/tcp",
				},
				{
					Name:  "worker-service",
					State: "exited",
					Image: "python:3.12-slim",
					Ports: "",
				},
			},
		},
	}

	view := RenderCompose(ComposeContext{
		Projects:     projs,
		CursorIdx:    0,
		FormatStatus: func(s, _ string) string { return s },
		Width:        120,
	})

	t.Run("running and non-running service states", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(view, "● running") {
			t.Errorf("expected running bullet '● running', got: %s", view)
		}
		if !strings.Contains(view, "○ exited") {
			t.Errorf("expected exited bullet '○ exited', got: %s", view)
		}
	})

	t.Run("service names and image truncation", func(t *testing.T) {
		t.Parallel()
		if !strings.Contains(view, "api-gateway") || !strings.Contains(view, "worker-service") {
			t.Errorf("expected service names in view, got: %s", view)
		}
		// Image is truncated to 20 chars
		if !strings.Contains(view, "traefik:v3.1.2-al...") {
			t.Errorf("expected truncated image name, got: %s", view)
		}
	})

	t.Run("port cleaning cleans 0.0.0.0 and IPv6 bindings", func(t *testing.T) {
		t.Parallel()
		if strings.Contains(view, "0.0.0.0:") || strings.Contains(view, "[::]:") {
			t.Errorf("expected cleaned ports without IP prefixes, got: %s", view)
		}
		if !strings.Contains(view, "80->80/tcp") {
			t.Errorf("expected cleaned port '80->80/tcp', got: %s", view)
		}
	})
}

func TestRenderComposeView_WidthConstraints(t *testing.T) {
	t.Parallel()

	projs := []monitor.ComposeProject{
		{
			Name: "mini-stack",
			Services: []monitor.ComposeService{
				{
					Name:  "svc",
					State: "running",
					Image: "alpine",
					Ports: "8080->80/tcp",
				},
			},
		},
	}

	t.Run("narrow width clamps to minimum content width 65", func(t *testing.T) {
		t.Parallel()
		viewNarrow := RenderCompose(ComposeContext{
			Projects:     projs,
			CursorIdx:    0,
			FormatStatus: func(s, _ string) string { return s },
			Width:        30,
		})
		// Border line repeat should be at least 65
		minBorder := "└" + strings.Repeat("─", 65)
		if !strings.Contains(viewNarrow, minBorder) {
			t.Errorf("expected clamped bottom border of length >= 65, got: %s", viewNarrow)
		}
	})

	t.Run("wide width scales border properly", func(t *testing.T) {
		t.Parallel()
		viewWide := RenderCompose(ComposeContext{
			Projects:     projs,
			CursorIdx:    0,
			FormatStatus: func(s, _ string) string { return s },
			Width:        100,
		})
		// contentWidth = max(100-4, 65) = 96
		wideBorder := "└" + strings.Repeat("─", 96)
		if !strings.Contains(viewWide, wideBorder) {
			t.Errorf("expected scaled bottom border of length 96, got: %s", viewWide)
		}
	})
}
