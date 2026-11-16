package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// VolumesContext contains data needed to render the Docker Volumes tab.
type VolumesContext struct {
	Volumes   []monitor.DockerVolume
	CursorIdx int
	Error     string
	Width     int
	I18n      *i18n.Dictionary
}

// RenderVolumes renders Docker volumes and details for the selected volume.
func RenderVolumes(ctx VolumesContext) string {
	d := getDictionary(ctx.I18n)
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render(fmt.Sprintf(d.VolumesTabTitle, len(ctx.Volumes))) + "\n\n")

	if ctx.Error != "" {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf(d.VolumesUnavailable, ctx.Error)) + "\n")
		return s.String()
	}
	if len(ctx.Volumes) == 0 {
		s.WriteString("  " + theme.DimStyle.Render(d.NoVolumes) + "\n")
		return s.String()
	}

	contentWidth := max(ctx.Width-4, 68)
	extraSpace := max(0, contentWidth-68)
	nameWidth := 24 + extraSpace

	s.WriteString(fmt.Sprintf("  %-*s  %-10s  %-8s  %-6s  %s\n",
		nameWidth, d.ColName, d.ColDriver, d.ColScope, d.ColReferences, d.ColSize))
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
		title := fmt.Sprintf(" %s: %s ", d.VolumeDetailsTitle, volume.Name)
		topDashLen := cardWidth - lipgloss.Width(title) - 3
		if topDashLen < 2 {
			topDashLen = 2
		}
		s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

		s.WriteString(renderDetailRow(d.VolumeMountLabel, valueOrDash(volume.Mountpoint)))
		s.WriteString(renderDetailRow(d.ComposeProjectLabel, valueOrDash(volume.ComposeProject)))

		s.WriteString("  " + theme.CyanStyle.Render("└"+strings.Repeat("─", cardWidth-2)+"┘") + "\n")
	}
	return s.String()
}
