package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/components"
	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
)

// ComposeContext contains data needed to render the Compose Stacks tab.
type ComposeContext struct {
	Projects           []monitor.ComposeProject
	TotalProjectsCount int
	CursorIdx          int
	FormatStatus       func(status, display string) string
	IsFiltering        bool
	FilterQuery        string
	Width              int
}

// RenderCompose renders the Compose Stacks tab.
func RenderCompose(ctx ComposeContext) string {
	var s strings.Builder

	count := len(ctx.Projects)
	if ctx.TotalProjectsCount > 0 {
		count = ctx.TotalProjectsCount
	}

	s.WriteString(theme.TitleStyle.Render(fmt.Sprintf("DOCKER COMPOSE PROJECTS (%d)", count)) + "\n\n")

	if ctx.IsFiltering {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("[Search: %s█]", ctx.FilterQuery)) + "  " + theme.DimStyle.Render("(Enter: apply, Esc: cancel)") + "\n\n")
	} else if ctx.FilterQuery != "" {
		badge := theme.ActiveTabStyle.Render(fmt.Sprintf("Filter: %s", ctx.FilterQuery)) + "  " +
			theme.DimStyle.Render(fmt.Sprintf("(Found: %d of %d, Esc: reset)", len(ctx.Projects), ctx.TotalProjectsCount))
		s.WriteString("  " + badge + "\n\n")
	}

	if len(ctx.Projects) == 0 {
		if ctx.FilterQuery != "" {
			s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("No Compose projects found matching %q.", ctx.FilterQuery)) + "\n")
			s.WriteString("  " + theme.DimStyle.Render("Press Esc to clear search filter.") + "\n")
		} else {
			s.WriteString("  " + theme.DimStyle.Render("No Docker Compose projects detected (missing com.docker.compose.project labels).") + "\n")
		}
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
			cleanPorts := CleanTablePorts(svc.Ports)
			portsTrunc := components.Truncate(cleanPorts, portsLen)
			s.WriteString(fmt.Sprintf("│   • %-16s %-20s %-22s %s\n",
				theme.BoldStyle.Render(svc.Name), st, components.Truncate(svc.Image, 20), portsTrunc,
			))
		}
		s.WriteString("└" + strings.Repeat("─", contentWidth) + "\n\n")
	}

	return s.String()
}
