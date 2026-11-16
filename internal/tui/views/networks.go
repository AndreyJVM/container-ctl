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

// NetworksContext contains data needed to render the Docker Networks tab.
type NetworksContext struct {
	Networks  []monitor.DockerNetwork
	CursorIdx int
	Error     string
	Width     int
	I18n      *i18n.Dictionary
}

// RenderNetworks renders Docker networks and details for the selected network.
func RenderNetworks(ctx NetworksContext) string {
	d := getDictionary(ctx.I18n)
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render(fmt.Sprintf(d.NetworksTabTitle, len(ctx.Networks))) + "\n\n")

	if ctx.Error != "" {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf(d.NetworksUnavailable, ctx.Error)) + "\n")
		return s.String()
	}
	if len(ctx.Networks) == 0 {
		s.WriteString("  " + theme.DimStyle.Render(d.NoNetworks) + "\n")
		return s.String()
	}

	contentWidth := max(ctx.Width-4, 76)
	extraSpace := max(0, contentWidth-76)
	nameWidth := 20 + (extraSpace * 4 / 10)
	subnetWidth := max(16, contentWidth-(12+nameWidth+10+10+10))

	s.WriteString(fmt.Sprintf("  %-12s  %-*s  %-10s  %-10s  %s\n",
		d.ColID, nameWidth, d.ColName, d.ColDriver, d.ColContainers, d.ColSubnet))
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", contentWidth)) + "\n")
	for index, network := range ctx.Networks {
		idPadded := components.Truncate(network.ID, 12)
		namePadded := components.PadRight(components.Truncate(network.Name, nameWidth), nameWidth)
		driverPadded := components.PadRight(components.Truncate(network.Driver, 10), 10)
		subnetPadded := components.Truncate(network.Subnet, subnetWidth)

		row := fmt.Sprintf("%-12s  %-*s  %-10s  %-10d  %s",
			idPadded,
			nameWidth,
			namePadded,
			driverPadded,
			network.Containers,
			subnetPadded,
		)
		if index == ctx.CursorIdx {
			s.WriteString(theme.GreenStyle.Render("► ") + theme.SelectedRowStyle.Render(row) + "\n")
		} else {
			s.WriteString("  " + row + "\n")
		}
	}

	if ctx.CursorIdx >= 0 && ctx.CursorIdx < len(ctx.Networks) {
		network := ctx.Networks[ctx.CursorIdx]
		s.WriteString("\n")

		cardWidth := contentWidth
		title := fmt.Sprintf(" %s: %s (%s) ", d.NetworkDetailsTitle, network.Name, network.ID)
		topDashLen := cardWidth - lipgloss.Width(title) - 3
		if topDashLen < 2 {
			topDashLen = 2
		}
		s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

		s.WriteString(renderDetailRow(d.NetworkGatewayLabel, valueOrDash(network.Gateway)))
		s.WriteString(renderDetailRow(d.NetworkScopeLabel, valueOrDash(network.Scope)))
		s.WriteString(renderDetailRow(d.NetworkInternalLabel, fmt.Sprintf("%t", network.Internal)))
		s.WriteString(renderDetailRow(d.NetworkAttachableLabel, fmt.Sprintf("%t", network.Attachable)))
		s.WriteString(renderDetailRow(d.ComposeProjectLabel, valueOrDash(network.ComposeProject)))

		s.WriteString("  " + theme.CyanStyle.Render("└"+strings.Repeat("─", cardWidth-2)+"┘") + "\n")
	}
	return s.String()
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
