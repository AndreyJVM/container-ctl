package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

// ComposeContext contains data needed to render the Compose Stacks tab.
type ComposeContext struct {
	Projects     []monitor.ComposeProject
	CursorIdx    int
	FormatStatus func(status, display string) string
	Width        int
}

// RenderCompose renders the Compose Stacks tab.
func RenderCompose(ctx ComposeContext) string {
	var s strings.Builder

	s.WriteString(theme.TitleStyle.Render(fmt.Sprintf("DOCKER COMPOSE PROJECTS (%d)", len(ctx.Projects))) + "\n\n")

	if len(ctx.Projects) == 0 {
		s.WriteString("  " + theme.DimStyle.Render("No Docker Compose projects detected (missing com.docker.compose.project labels).") + "\n")
		return s.String()
	}

	contentWidth := max(ctx.Width-4, 65)

	for i, p := range ctx.Projects {
		prefix := "  "
		statusDisp := ctx.FormatStatus(p.Status, p.StatusDisplay)
		headerTitle := fmt.Sprintf("┌─ %s  [%s]  (Services up: %d/%d)",
			p.Name, statusDisp, p.RunningServices, p.TotalServices,
		)
		if i == ctx.CursorIdx {
			prefix = theme.GreenStyle.Render("► ")
			s.WriteString(fmt.Sprintf("%s%s\n", prefix, theme.SelectedRowStyle.Render(headerTitle)))
		} else {
			headerText := fmt.Sprintf("┌─ %s  [%s]  (Services up: %d/%d)",
				theme.BoldStyle.Render(p.Name), statusDisp, p.RunningServices, p.TotalServices,
			)
			s.WriteString(fmt.Sprintf("%s%s\n", prefix, headerText))
		}
		for _, svc := range p.Services {
			st := theme.DimStyle.Render("○ " + svc.State)
			if svc.State == "running" {
				st = theme.GreenStyle.Render("● running")
			}
			portsLen := max(10, contentWidth-66)
			portsTrunc := components.Truncate(svc.Ports, portsLen)
			s.WriteString(fmt.Sprintf("│   • %-16s %-20s %-22s %s\n",
				theme.BoldStyle.Render(svc.Name), st, components.Truncate(svc.Image, 20), portsTrunc,
			))
		}
		s.WriteString("└" + strings.Repeat("─", contentWidth) + "\n\n")
	}

	return s.String()
}
