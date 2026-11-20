package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

// Constants for inspect modal constraints.
const DefaultMaxVisibleInspect = 18

// BuildInspectLines converts container inspection data into formatted display lines.
func BuildInspectLines(d monitor.ContainerDetail, dict *i18n.Dictionary) []string {
	dict = getModalDictionary(dict)
	var lines []string

	lines = append(lines, fmt.Sprintf("%s %s", theme.DimStyle.Render("ID:"), d.ID))
	lines = append(lines, fmt.Sprintf("%s %s", theme.DimStyle.Render(dict.ImageLabel), theme.BoldStyle.Render(d.Image)))

	healthBadge := theme.DimStyle.Render("-")
	switch d.Health {
	case "healthy":
		healthBadge = theme.GreenStyle.Render("[healthy]")
	case "unhealthy":
		healthBadge = theme.RedStyle.Render("[unhealthy]")
	case "starting":
		healthBadge = theme.YellowStyle.Render("[starting]")
	}

	lines = append(lines, fmt.Sprintf("%s %s    %s %s",
		theme.DimStyle.Render(dict.StatusLabel), theme.GreenStyle.Render(d.Status),
		theme.DimStyle.Render(dict.HealthLabel), healthBadge,
	))

	if d.State == "running" {
		if d.CPUPerc > 0 || d.MemUsageMB > 0 {
			lines = append(lines, fmt.Sprintf("%s %.1f%%    %s %.1f MB (%.1f%%)",
				theme.DimStyle.Render("CPU:"), d.CPUPerc,
				theme.DimStyle.Render(dict.MemLabel), d.MemUsageMB, d.MemPerc,
			))
		}
		if d.NetRxBytes > 0 || d.NetTxBytes > 0 || d.BlockReadBytes > 0 || d.BlockWriteBytes > 0 {
			lines = append(lines, fmt.Sprintf("%s %s / %s    %s %s / %s",
				theme.DimStyle.Render("Net I/O:"),
				monitor.FormatUintBytes(d.NetRxBytes),
				monitor.FormatUintBytes(d.NetTxBytes),
				theme.DimStyle.Render("Block I/O:"),
				monitor.FormatUintBytes(d.BlockReadBytes),
				monitor.FormatUintBytes(d.BlockWriteBytes),
			))
		}
	}

	restartVal := d.RestartPolicy
	if restartVal == "" {
		restartVal = dict.DetailNone
	}
	lines = append(lines, fmt.Sprintf("%s %s    %s %s",
		theme.DimStyle.Render(dict.CreatedLabel), d.Created,
		theme.DimStyle.Render(dict.DetailRestartPolicyLabel), theme.CyanStyle.Render(restartVal),
	))
	lines = append(lines, fmt.Sprintf("%s %s    %s %s",
		theme.DimStyle.Render(dict.IPAddrLabel), theme.BoldStyle.Render(d.IP),
		theme.DimStyle.Render(dict.GatewayLabel), d.Gateway,
	))
	lines = append(lines, fmt.Sprintf("%s %s", theme.DimStyle.Render(dict.CommandLabel), components.Truncate(d.Command, 60)))

	if len(d.Mounts) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render(dict.MountsTitle))
		for _, mnt := range d.Mounts {
			lines = append(lines, fmt.Sprintf(" • %s", mnt))
		}
	}

	if len(d.Env) > 0 {
		envTitle := dict.DetailEnvLabel
		if envTitle == "" {
			envTitle = "Environment:"
		}
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render(envTitle))
		for _, e := range d.Env {
			lines = append(lines, fmt.Sprintf(" • %s", components.Truncate(MaskEnvVar(e), 76)))
		}
	}

	return lines
}

// RenderInspect renders the detailed container inspection modal with optional scrolling.
func RenderInspect(d monitor.ContainerDetail, dict *i18n.Dictionary, scrollOpts ...int) string {
	var s strings.Builder
	dict = getModalDictionary(dict)

	title := fmt.Sprintf(dict.InspectTitle, theme.BoldStyle.Render(d.Name))
	s.WriteString(theme.TitleStyle.Render(title) + "\n\n")

	lines := BuildInspectLines(d, dict)
	scroll := 0
	if len(scrollOpts) > 0 {
		scroll = scrollOpts[0]
	}

	if len(lines) <= DefaultMaxVisibleInspect {
		for _, l := range lines {
			s.WriteString("  " + l + "\n")
		}
	} else {
		start, end := calculateLogRange(len(lines), scroll, DefaultMaxVisibleInspect)
		for i := start; i < end; i++ {
			s.WriteString("  " + lines[i] + "\n")
		}
		s.WriteString(fmt.Sprintf("\n  %s\n", theme.DimStyle.Render(fmt.Sprintf("[Lines %d-%d of %d] (scroll ↑/↓/PgUp/PgDn, [y] Copy ID)", start+1, end, len(lines)))))
	}

	return theme.BoxStyle.Render(s.String())
}

// MaskEnvVar obscures values of sensitive environment variables containing passwords, secrets, keys, or tokens.
func MaskEnvVar(entry string) string {
	idx := strings.IndexByte(entry, '=')
	if idx == -1 {
		return entry
	}
	key := entry[:idx]
	val := entry[idx+1:]
	upperKey := strings.ToUpper(key)
	if strings.Contains(upperKey, "PASS") ||
		strings.Contains(upperKey, "SECRET") ||
		strings.Contains(upperKey, "TOKEN") ||
		strings.Contains(upperKey, "KEY") ||
		strings.Contains(upperKey, "AUTH") ||
		strings.Contains(upperKey, "CREDENTIAL") {
		return key + "=******"
	}
	return key + "=" + val
}
