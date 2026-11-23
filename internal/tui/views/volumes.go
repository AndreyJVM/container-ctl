package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/components"
	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
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
	MaxRows           int
	HideDetails       bool
	SelectedNames     map[string]struct{}
}

// RenderVolumes renders Docker volumes and details for the selected volume.
func RenderVolumes(ctx VolumesContext) string {
	var s strings.Builder

	count := len(ctx.Volumes)
	if ctx.TotalVolumesCount > 0 {
		count = ctx.TotalVolumesCount
	}

	headerText := fmt.Sprintf("DOCKER VOLUMES (%d)", count)
	summaryLine := theme.TitleStyle.Render(headerText)
	if len(ctx.SelectedNames) > 0 {
		summaryLine += "   " + theme.CyanStyle.Render(fmt.Sprintf("● %d selected", len(ctx.SelectedNames)))
	}
	if ctx.MaxRows > 0 && len(ctx.Volumes) > ctx.MaxRows {
		start, end := components.CalculateTableWindow(len(ctx.Volumes), ctx.CursorIdx, ctx.MaxRows)
		if badge := components.FormatWindowBadge(start, end, len(ctx.Volumes)); badge != "" {
			summaryLine += "   " + theme.CyanStyle.Render(badge)
		}
	}
	s.WriteString(summaryLine + "\n\n")

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

	start, end := 0, len(ctx.Volumes)
	if ctx.MaxRows > 0 && len(ctx.Volumes) > ctx.MaxRows {
		start, end = components.CalculateTableWindow(len(ctx.Volumes), ctx.CursorIdx, ctx.MaxRows)
	}

	for index := start; index < end; index++ {
		volume := ctx.Volumes[index]
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
		isMarked := false
		if ctx.SelectedNames != nil {
			_, isMarked = ctx.SelectedNames[volume.Name]
		}
		prefix := "  "
		if isMarked && index == ctx.CursorIdx {
			prefix = theme.CyanStyle.Render("►●")
		} else if isMarked {
			prefix = theme.CyanStyle.Render("● ")
		} else if index == ctx.CursorIdx {
			prefix = theme.GreenStyle.Render("► ")
		}

		if index == ctx.CursorIdx {
			s.WriteString(prefix + theme.SelectedRowStyle.Render(row) + "\n")
		} else {
			s.WriteString(prefix + row + "\n")
		}
	}

	if start > 0 || end < len(ctx.Volumes) {
		var hints []string
		if start > 0 {
			hints = append(hints, fmt.Sprintf("▲ %d more above", start))
		}
		if end < len(ctx.Volumes) {
			hints = append(hints, fmt.Sprintf("▼ %d more below", len(ctx.Volumes)-end))
		}
		s.WriteString("  " + theme.DimStyle.Render(strings.Join(hints, "  │  ")) + "\n")
	}

	if !ctx.HideDetails && ctx.CursorIdx >= 0 && ctx.CursorIdx < len(ctx.Volumes) {
		volume := ctx.Volumes[ctx.CursorIdx]
		s.WriteString("\n")

		cardWidth := contentWidth
		title := fmt.Sprintf(" SELECTED VOLUME DETAILS: %s ", volume.Name)
		topDashLen := cardWidth - lipgloss.Width(title) - 3
		if topDashLen < 2 {
			topDashLen = 2
		}
		s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

		createdStr := volume.Created
		if createdStr != "" {
			s.WriteString(renderDetailRow("Driver / Scope:", fmt.Sprintf("%s / %s  •  Created: %s", valueOrDash(volume.Driver), valueOrDash(volume.Scope), createdStr)))
		} else {
			s.WriteString(renderDetailRow("Driver / Scope:", fmt.Sprintf("%s / %s", valueOrDash(volume.Driver), valueOrDash(volume.Scope))))
		}

		sizeDisplay := "-"
		if volume.Size >= 0 {
			sizeDisplay = monitor.FormatBytes(volume.Size)
		}
		s.WriteString(renderDetailRow("Size:", sizeDisplay))

		var statusDisplay string
		if volume.RefCount > 0 || len(volume.Containers) > 0 {
			count := volume.RefCount
			if count <= 0 {
				count = len(volume.Containers)
			}
			msg := fmt.Sprintf("● In Use (%d container reference(s))", count)
			if len(volume.ContainerMounts) > 0 {
				msg += ": " + strings.Join(volume.ContainerMounts, ", ")
			} else if len(volume.Containers) > 0 {
				msg += ": " + strings.Join(volume.Containers, ", ")
			}
			statusDisplay = theme.GreenStyle.Render(msg)
		} else if volume.RefCount == 0 {
			statusDisplay = theme.YellowStyle.Render("○ Unused (reclaimable) — Press 'C' to prune or 'x' to remove")
		} else {
			statusDisplay = theme.DimStyle.Render("Unknown")
		}
		s.WriteString(renderDetailRow("Status / Usage:", statusDisplay))

		s.WriteString(renderDetailRow("Mountpoint:", valueOrDash(volume.Mountpoint)))
		s.WriteString(renderDetailRow("Compose project:", valueOrDash(volume.ComposeProject)))

		var actions []string
		actions = append(actions, theme.BoldStyle.Render("[Enter]")+" Inspect", theme.BoldStyle.Render("[x]")+" Remove")
		if volume.RefCount == 0 {
			actions = append(actions, theme.BoldStyle.Render("[C]")+" Prune", theme.BoldStyle.Render("[y]")+" Copy Name")
		} else {
			actions = append(actions, theme.BoldStyle.Render("[y]")+" Copy Name")
		}
		actions = append(actions, theme.BoldStyle.Render("[V]")+" Details")
		s.WriteString(renderDetailRow("Quick Actions:", strings.Join(actions, "  ")))

		s.WriteString("  " + theme.CyanStyle.Render("└"+strings.Repeat("─", cardWidth-2)+"┘") + "\n")
	}
	return s.String()
}
