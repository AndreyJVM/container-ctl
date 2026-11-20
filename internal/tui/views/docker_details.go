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

func renderDetailRow(label, value string) string {
	var formattedLabel string
	if label == "" {
		formattedLabel = strings.Repeat(" ", detailColWidth)
	} else {
		formattedLabel = theme.BoldStyle.Render(components.PadRight(label, detailColWidth))
	}
	return fmt.Sprintf("  %s %s %s\n",
		theme.CyanStyle.Render("│"),
		formattedLabel,
		value,
	)
}

// FormatPortDisplay converts a raw comma-separated ports string into clean,
// deduplicated mappings with unicode arrows (e.g. "0.0.0.0:8080 → 8080/tcp").
func FormatPortDisplay(portsStr string) []string {
	if strings.TrimSpace(portsStr) == "" {
		return nil
	}
	rawParts := strings.Split(portsStr, ",")
	seenMap := make(map[string]bool)
	var result []string

	for _, p := range rawParts {
		tr := strings.TrimSpace(p)
		if tr == "" {
			continue
		}

		clean := strings.ReplaceAll(tr, "->", " → ")

		// Deduplicate IPv6 if identical IPv4 mapping exists
		dedupKey := clean
		if strings.HasPrefix(clean, "[::]:") {
			dedupKey = "0.0.0.0:" + strings.TrimPrefix(clean, "[::]:")
		}

		if seenMap[dedupKey] || seenMap[clean] {
			continue
		}
		seenMap[dedupKey] = true
		seenMap[clean] = true
		result = append(result, clean)
	}
	return result
}

func renderContainerDetailsCard(ctx *DockerContext, dict *i18n.Dictionary, s *strings.Builder) {
	if ctx.CursorIdx < 0 || ctx.CursorIdx >= len(ctx.Containers) {
		return
	}

	sel := ctx.Containers[ctx.CursorIdx]
	s.WriteString("\n")
	cardWidth := max(ctx.Width-4, minCardWidth)
	title := fmt.Sprintf(" %s: %s (%s) ", dict.SelectedContainerDetails, sel.Name, sel.ID)
	topDashLen := cardWidth - lipgloss.Width(title) - 3
	if topDashLen < 2 {
		topDashLen = 2
	}
	s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

	// 1. IP / Network
	ipDisplay := sel.IPAddress
	if ipDisplay == "" {
		if sel.NetworkName != "" {
			ipDisplay = fmt.Sprintf("%s (%s)", theme.DimStyle.Render(dict.DetailNone), sel.NetworkName)
		} else {
			ipDisplay = theme.DimStyle.Render(dict.DetailNone)
		}
	} else if sel.NetworkName != "" {
		ipDisplay = fmt.Sprintf("%s  %s", theme.BoldStyle.Render(ipDisplay), theme.DimStyle.Render(fmt.Sprintf("(%s)", sel.NetworkName)))
	} else {
		ipDisplay = theme.BoldStyle.Render(ipDisplay)
	}
	s.WriteString(renderDetailRow(dict.DetailIPLabel, ipDisplay))

	// 2. Uptime / Started
	uptimeDisplay := sel.Status
	if uptimeDisplay == "" {
		uptimeDisplay = sel.State
	}

	var statusStyled string
	if failed, _ := monitor.IsContainerFailed(sel.State, sel.Status); failed {
		statusStyled = theme.RedStyle.Render(uptimeDisplay)
	} else if sel.State == "running" {
		statusStyled = theme.GreenStyle.Render(uptimeDisplay)
	} else if sel.State == "paused" || sel.State == "restarting" {
		statusStyled = theme.YellowStyle.Render(uptimeDisplay)
	} else {
		statusStyled = theme.DimStyle.Render(uptimeDisplay)
	}

	if sel.Created != "" {
		uptimeDisplay = fmt.Sprintf("%s  %s %s",
			statusStyled,
			theme.DimStyle.Render("• "+dict.CreatedLabel),
			theme.DimStyle.Render(sel.Created),
		)
	} else {
		uptimeDisplay = statusStyled
	}
	s.WriteString(renderDetailRow(dict.DetailUptimeLabel, uptimeDisplay))

	// 3. Restart Policy
	restartDisplay := sel.RestartPolicy
	if restartDisplay == "" {
		restartDisplay = theme.DimStyle.Render(dict.DetailNone)
	} else {
		switch strings.ToLower(restartDisplay) {
		case "always", "unless-stopped":
			restartDisplay = theme.GreenStyle.Render(restartDisplay)
		case "on-failure":
			restartDisplay = theme.YellowStyle.Render(restartDisplay)
		case "no":
			restartDisplay = theme.DimStyle.Render(restartDisplay)
		default:
			restartDisplay = theme.CyanStyle.Render(restartDisplay)
		}
	}
	s.WriteString(renderDetailRow(dict.DetailRestartPolicyLabel, restartDisplay))

	// 4. Ports
	portItems := FormatPortDisplay(sel.Ports)
	if len(portItems) == 0 {
		s.WriteString(renderDetailRow(dict.DetailPortsLabel, theme.DimStyle.Render(dict.DetailNone)))
	} else if len(portItems) == 1 {
		s.WriteString(renderDetailRow(dict.DetailPortsLabel, theme.CyanStyle.Render(portItems[0])))
	} else {
		for idx, port := range portItems {
			label := ""
			if idx == 0 {
				label = dict.DetailPortsLabel
			}
			s.WriteString(renderDetailRow(label, theme.CyanStyle.Render("• ")+theme.CyanStyle.Render(port)))
		}
	}

	// 5. Volumes / Mounts
	if len(sel.Mounts) == 0 {
		s.WriteString(renderDetailRow(dict.DetailMountsLabel, theme.DimStyle.Render(dict.DetailNone)))
	} else {
		for idx, mnt := range sel.Mounts {
			label := ""
			if idx == 0 {
				label = dict.DetailMountsLabel
			}
			s.WriteString(renderDetailRow(label, theme.YellowStyle.Render("• ")+mnt))
		}
	}

	// 6. Network I/O
	netIODisplay := theme.DimStyle.Render("0 B / 0 B")
	if sel.State == "running" {
		netIODisplay = fmt.Sprintf("%s / %s",
			theme.GreenStyle.Render(monitor.FormatUintBytes(sel.NetRxBytes)),
			theme.CyanStyle.Render(monitor.FormatUintBytes(sel.NetTxBytes)),
		)
	}
	s.WriteString(renderDetailRow(dict.DetailNetIOLabel, netIODisplay))

	// 7. Block I/O
	blockIODisplay := theme.DimStyle.Render("0 B / 0 B")
	if sel.State == "running" {
		blockIODisplay = fmt.Sprintf("%s / %s",
			theme.GreenStyle.Render(monitor.FormatUintBytes(sel.BlockReadBytes)),
			theme.YellowStyle.Render(monitor.FormatUintBytes(sel.BlockWriteBytes)),
		)
	}
	s.WriteString(renderDetailRow(dict.DetailBlockIOLabel, blockIODisplay))

	// 8. Compose Info
	if sel.ComposeProject != "" {
		composeDisplay := fmt.Sprintf("%s / %s", theme.BoldStyle.Render(sel.ComposeProject), sel.ComposeService)
		s.WriteString(renderDetailRow(dict.DetailComposeLabel, composeDisplay))
	}

	s.WriteString("  " + theme.CyanStyle.Render("└"+strings.Repeat("─", cardWidth-2)+"┘") + "\n")
}

func formatContainerStatus(state, health string, rawStatus ...string) (string, lipgloss.Style) {
	status := ""
	if len(rawStatus) > 0 {
		status = rawStatus[0]
	}

	if state == "running" {
		switch health {
		case "healthy":
			return "● [healthy]", theme.GreenStyle
		case "unhealthy":
			return "▲ [unhealthy]", theme.RedStyle
		case "starting":
			return "⏳ [starting]", theme.YellowStyle
		default:
			return "● running", theme.GreenStyle
		}
	} else if state == "paused" {
		return "⏸ paused", theme.YellowStyle
	} else if state == "restarting" {
		return "↻ restarting", theme.YellowStyle
	} else if state == "dead" {
		return "✖ dead", theme.RedStyle
	}

	if failed, code := monitor.IsContainerFailed(state, status); failed {
		if code != -1 {
			return fmt.Sprintf("✖ failed (%d)", code), theme.RedStyle
		}
		return "✖ failed", theme.RedStyle
	}

	if hasCode, code := monitor.ParseExitCode(status); hasCode && code == 0 {
		return "○ exited (0)", theme.DimStyle
	}

	if state == "" {
		state = "stopped"
	}
	return "○ " + state, theme.DimStyle
}

func styleCPU(val float64, state, paddedStr string) string {
	if state != "running" {
		return theme.DimStyle.Render(paddedStr)
	}
	if val >= 80 {
		return theme.RedStyle.Render(paddedStr)
	} else if val >= 50 {
		return theme.YellowStyle.Render(paddedStr)
	}
	return theme.GreenStyle.Render(paddedStr)
}
