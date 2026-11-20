package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// VolumesContext contains data needed to render the Docker Volumes tab.
type VolumesContext struct {
	Volumes           []monitor.DockerVolume
	TotalVolumesCount int
	CursorIdx         int
	Error             string
	IsFiltering       bool
	FilterQuery       string
	Width             int
}

// RenderVolumes renders Docker volumes and details for the selected volume.
func RenderVolumes(ctx VolumesContext) string {
	var s strings.Builder

	count := len(ctx.Volumes)
	if ctx.TotalVolumesCount > 0 {
		count = ctx.TotalVolumesCount
	}

	s.WriteString(theme.TitleStyle.Render(fmt.Sprintf("DOCKER VOLUMES (%d)", count)) + "\n\n")

	if ctx.Error != "" {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("Docker volumes are unavailable: %s", ctx.Error)) + "\n")
		return s.String()
	}

	if ctx.IsFiltering {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("[Search: %s█]", ctx.FilterQuery)) + "  " + theme.DimStyle.Render("(Enter: apply, Esc: cancel)") + "\n\n")
	} else if ctx.FilterQuery != "" {
		badge := theme.ActiveTabStyle.Render(fmt.Sprintf("Filter: %s", ctx.FilterQuery)) + "  " +
			theme.DimStyle.Render(fmt.Sprintf("(Found: %d of %d, Esc: reset)", len(ctx.Volumes), ctx.TotalVolumesCount))
		s.WriteString("  " + badge + "\n\n")
	}

	if len(ctx.Volumes) == 0 {
		if ctx.FilterQuery != "" {
			s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("No Docker volumes found matching %q.", ctx.FilterQuery)) + "\n")
			s.WriteString("  " + theme.DimStyle.Render("Press Esc to clear search filter.") + "\n")
		} else {
			s.WriteString("  " + theme.DimStyle.Render("No Docker volumes found.") + "\n")
		}
		return s.String()
	}

	contentWidth := max(ctx.Width-4, 68)
	extraSpace := max(0, contentWidth-68)
	nameWidth := 24 + extraSpace

	s.WriteString(fmt.Sprintf("  %-*s  %-10s  %-8s  %-6s  %s\n",
		nameWidth, "NAME", "DRIVER", "SCOPE", "REFS", "SIZE"))
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	for index, volume := range ctx.Volumes {
		size := "-"
		if volume.Size >= 0 {
			size = monitor.FormatBytes(volume.Size)
		}
		references := "-"
		if volume.RefCount >= 0 {
			references = fmt.Sprintf("%d", volume.RefCount)
		}
		namePadded := components.PadRight(components.Truncate(volume.Name, nameWidth), nameWidth)
		driverPadded := components.PadRight(components.Truncate(volume.Driver, 10), 10)
		scopePadded := components.PadRight(components.Truncate(volume.Scope, 8), 8)
		refPadded := components.PadRight(references, 6)

		row := fmt.Sprintf("%-*s  %-10s  %-8s  %-6s  %s",
			nameWidth,
			namePadded,
			driverPadded,
			scopePadded,
			refPadded,
			size,
		)
		if index == ctx.CursorIdx {
			s.WriteString(theme.GreenStyle.Render("► ") + theme.SelectedRowStyle.Render(row) + "\n")
		} else {
			s.WriteString("  " + row + "\n")
		}
	}

	if ctx.CursorIdx >= 0 && ctx.CursorIdx < len(ctx.Volumes) {
		volume := ctx.Volumes[ctx.CursorIdx]
		s.WriteString("\n")

		cardWidth := contentWidth
		title := fmt.Sprintf(" SELECTED VOLUME DETAILS: %s ", volume.Name)
		topDashLen := cardWidth - lipgloss.Width(title) - 3
		if topDashLen < 2 {
			topDashLen = 2
		}
		s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

		s.WriteString(renderDetailRow("Driver / Scope:", fmt.Sprintf("%s / %s", valueOrDash(volume.Driver), valueOrDash(volume.Scope))))

		sizeDisplay := "-"
		if volume.Size >= 0 {
			sizeDisplay = monitor.FormatBytes(volume.Size)
		}
		s.WriteString(renderDetailRow("Size:", sizeDisplay))

		var statusDisplay string
		if volume.RefCount > 0 {
			statusDisplay = theme.GreenStyle.Render(fmt.Sprintf("● In Use (%d container reference(s))", volume.RefCount))
		} else if volume.RefCount == 0 {
			statusDisplay = theme.YellowStyle.Render("○ Unused (reclaimable) — Press 'C' to prune")
		} else {
			statusDisplay = theme.DimStyle.Render("Unknown")
		}
		s.WriteString(renderDetailRow("Status / Usage:", statusDisplay))

		s.WriteString(renderDetailRow("Mountpoint:", valueOrDash(volume.Mountpoint)))
		s.WriteString(renderDetailRow("Compose project:", valueOrDash(volume.ComposeProject)))

		var actions []string
		if volume.RefCount == 0 {
			actions = append(actions, theme.BoldStyle.Render("[C]")+" Prune", theme.BoldStyle.Render("[y]")+" Copy Name")
		} else {
			actions = append(actions, theme.BoldStyle.Render("[y]")+" Copy Name")
		}
		s.WriteString(renderDetailRow("Quick Actions:", strings.Join(actions, "  ")))

		s.WriteString("  " + theme.CyanStyle.Render("└"+strings.Repeat("─", cardWidth-2)+"┘") + "\n")
	}
	return s.String()
}
