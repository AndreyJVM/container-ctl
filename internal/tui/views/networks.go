package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// NetworksContext contains data needed to render the Docker Networks tab.
type NetworksContext struct {
	Networks           []monitor.DockerNetwork
	TotalNetworksCount int
	CursorIdx          int
	Error              string
	IsFiltering        bool
	FilterQuery        string
	Width              int
}

// RenderNetworks renders Docker networks and details for the selected network.
func RenderNetworks(ctx NetworksContext) string {
	var s strings.Builder

	count := len(ctx.Networks)
	if ctx.TotalNetworksCount > 0 {
		count = ctx.TotalNetworksCount
	}

	s.WriteString(theme.TitleStyle.Render(fmt.Sprintf("DOCKER NETWORKS (%d)", count)) + "\n\n")

	if ctx.Error != "" {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("Docker networks are unavailable: %s", ctx.Error)) + "\n")
		return s.String()
	}

	if ctx.IsFiltering {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("[Search: %s█]", ctx.FilterQuery)) + "  " + theme.DimStyle.Render("(Enter: apply, Esc: cancel)") + "\n\n")
	} else if ctx.FilterQuery != "" {
		badge := theme.ActiveTabStyle.Render(fmt.Sprintf("Filter: %s", ctx.FilterQuery)) + "  " +
			theme.DimStyle.Render(fmt.Sprintf("(Found: %d of %d, Esc: reset)", len(ctx.Networks), ctx.TotalNetworksCount))
		s.WriteString("  " + badge + "\n\n")
	}

	if len(ctx.Networks) == 0 {
		if ctx.FilterQuery != "" {
			s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("No Docker networks found matching %q.", ctx.FilterQuery)) + "\n")
			s.WriteString("  " + theme.DimStyle.Render("Press Esc to clear search filter.") + "\n")
		} else {
			s.WriteString("  " + theme.DimStyle.Render("No Docker networks found.") + "\n")
		}
		return s.String()
	}

	contentWidth := max(ctx.Width-4, 76)
	extraSpace := max(0, contentWidth-76)
	nameWidth := 20 + (extraSpace * 4 / 10)
	subnetWidth := max(16, contentWidth-(12+nameWidth+10+10+10))

	s.WriteString(fmt.Sprintf("  %-12s  %-*s  %-10s  %-10s  %s\n",
		"ID", nameWidth, "NAME", "DRIVER", "CONTAINERS", "SUBNET"))
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
		title := fmt.Sprintf(" SELECTED NETWORK DETAILS: %s (%s) ", network.Name, network.ID)
		topDashLen := cardWidth - lipgloss.Width(title) - 3
		if topDashLen < 2 {
			topDashLen = 2
		}
		s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

		s.WriteString(renderDetailRow("Driver / Scope:", fmt.Sprintf("%s / %s", valueOrDash(network.Driver), valueOrDash(network.Scope))))
		s.WriteString(renderDetailRow("Subnet / Gateway:", fmt.Sprintf("%s  •  Gateway: %s", valueOrDash(network.Subnet), valueOrDash(network.Gateway))))

		var containersDisplay string
		if network.Containers > 0 {
			containersDisplay = theme.GreenStyle.Render(fmt.Sprintf("● %d container(s) connected", network.Containers))
		} else {
			containersDisplay = theme.DimStyle.Render("○ 0 containers connected (unused)")
		}
		s.WriteString(renderDetailRow("Containers:", containersDisplay))

		var flags []string
		if network.Internal {
			flags = append(flags, theme.YellowStyle.Render("internal (isolated)"))
		} else {
			flags = append(flags, theme.DimStyle.Render("external"))
		}
		if network.Attachable {
			flags = append(flags, theme.CyanStyle.Render("attachable"))
		}
		s.WriteString(renderDetailRow("Flags:", strings.Join(flags, " • ")))

		s.WriteString(renderDetailRow("Compose project:", valueOrDash(network.ComposeProject)))

		actions := []string{theme.BoldStyle.Render("[y]") + " Copy Name", theme.BoldStyle.Render("[C]") + " Prune Unused"}
		s.WriteString(renderDetailRow("Quick Actions:", strings.Join(actions, "  ")))

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
