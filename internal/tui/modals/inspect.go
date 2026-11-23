package modals

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/components"
	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
)

// Constants for inspect modal constraints.
const DefaultMaxVisibleInspect = 18

// BuildInspectLines converts container inspection data into formatted display lines.
func BuildInspectLines(d monitor.ContainerDetail) []string {
	var lines []string

	lines = append(lines, fmt.Sprintf("%s %s", theme.DimStyle.Render("ID:"), d.ID))
	lines = append(lines, fmt.Sprintf("%s %s", theme.DimStyle.Render("Image:"), theme.BoldStyle.Render(d.Image)))

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
		theme.DimStyle.Render("Status:"), theme.GreenStyle.Render(d.Status),
		theme.DimStyle.Render("Health:"), healthBadge,
	))

	if d.State == "running" {
		if d.CPUPerc > 0 || d.MemUsageMB > 0 {
			lines = append(lines, fmt.Sprintf("%s %.1f%%    %s %.1f MB (%.1f%%)",
				theme.DimStyle.Render("CPU:"), d.CPUPerc,
				theme.DimStyle.Render("Memory:"), d.MemUsageMB, d.MemPerc,
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
		restartVal = "none"
	}
	lines = append(lines, fmt.Sprintf("%s %s    %s %s",
		theme.DimStyle.Render("Created:"), d.Created,
		theme.DimStyle.Render("Restart Policy:"), theme.CyanStyle.Render(restartVal),
	))
	lines = append(lines, fmt.Sprintf("%s %s    %s %s",
		theme.DimStyle.Render("IP Address:"), theme.BoldStyle.Render(d.IP),
		theme.DimStyle.Render("Gateway:"), d.Gateway,
	))
	lines = append(lines, fmt.Sprintf("%s %s", theme.DimStyle.Render("Command:"), components.Truncate(d.Command, 60)))

	if len(d.Mounts) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("Mounts:"))
		for _, mnt := range d.Mounts {
			lines = append(lines, fmt.Sprintf(" • %s", mnt))
		}
	}

	if len(d.Env) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("Environment:"))
		for _, e := range d.Env {
			lines = append(lines, fmt.Sprintf(" • %s", components.Truncate(MaskEnvVar(e), 76)))
		}
	}

	return lines
}

// BuildNetworkInspectLines converts network data and inspect JSON into structured display lines.
func BuildNetworkInspectLines(net monitor.DockerNetwork, rawJSON string) []string {
	var lines []string

	lines = append(lines, fmt.Sprintf("%s %s    %s %s",
		theme.DimStyle.Render("ID:"), net.ID,
		theme.DimStyle.Render("Name:"), theme.BoldStyle.Render(net.Name),
	))

	createdStr := net.Created
	if createdStr == "" {
		createdStr = "-"
	}
	lines = append(lines, fmt.Sprintf("%s %s    %s %s    %s %s",
		theme.DimStyle.Render("Driver:"), theme.CyanStyle.Render(net.Driver),
		theme.DimStyle.Render("Scope:"), net.Scope,
		theme.DimStyle.Render("Created:"), createdStr,
	))

	// IPAM section
	lines = append(lines, "")
	lines = append(lines, theme.BoldStyle.Render("IPAM Configuration:"))
	subnetVal := net.Subnet
	if subnetVal == "" {
		subnetVal = "-"
	}
	gatewayVal := net.Gateway
	if gatewayVal == "" {
		gatewayVal = "-"
	}
	lines = append(lines, fmt.Sprintf("  %s %s    %s %s",
		theme.DimStyle.Render("Subnet:"), theme.GreenStyle.Render(subnetVal),
		theme.DimStyle.Render("Gateway:"), theme.GreenStyle.Render(gatewayVal),
	))
	ipRange := net.IPRange
	if ipRange == "" {
		ipRange = "-"
	}
	ipamDriver := net.IPAMDriver
	if ipamDriver == "" {
		ipamDriver = "default"
	}
	ipv6Status := theme.DimStyle.Render("disabled")
	if net.EnableIPv6 {
		ipv6Status = theme.GreenStyle.Render("enabled")
	}
	lines = append(lines, fmt.Sprintf("  %s %s    %s %s    %s %s",
		theme.DimStyle.Render("IP Range:"), ipRange,
		theme.DimStyle.Render("IPAM Driver:"), ipamDriver,
		theme.DimStyle.Render("IPv6:"), ipv6Status,
	))

	// Security & Flags
	lines = append(lines, "")
	lines = append(lines, theme.BoldStyle.Render("Flags & Security:"))
	var flags []string
	if net.Internal {
		flags = append(flags, theme.YellowStyle.Render("internal (isolated from host/internet)"))
	} else {
		flags = append(flags, theme.DimStyle.Render("external (access enabled)"))
	}
	if net.Attachable {
		flags = append(flags, theme.CyanStyle.Render("attachable"))
	}
	if net.Ingress {
		flags = append(flags, theme.CyanStyle.Render("ingress"))
	}
	lines = append(lines, fmt.Sprintf("  %s", strings.Join(flags, " • ")))

	// Connected Containers
	lines = append(lines, "")
	lines = append(lines, theme.BoldStyle.Render(fmt.Sprintf("Connected Containers (%d):", net.Containers)))
	if len(net.ContainerDetails) > 0 {
		for _, c := range net.ContainerDetails {
			ip := c.IPv4Address
			if ip == "" {
				ip = "-"
			}
			mac := c.MacAddress
			if mac == "" {
				mac = "-"
			}
			lines = append(lines, fmt.Sprintf("  • %s (ID: %s)", theme.BoldStyle.Render(c.Name), c.ID))
			lines = append(lines, fmt.Sprintf("    IPv4: %s   MAC: %s", theme.GreenStyle.Render(ip), theme.DimStyle.Render(mac)))
		}
	} else if len(net.ConnectedContainers) > 0 {
		for _, c := range net.ConnectedContainers {
			lines = append(lines, fmt.Sprintf("  • %s", c))
		}
	} else {
		lines = append(lines, "  "+theme.DimStyle.Render("○ No containers currently connected (network unused)"))
	}

	// Options
	if len(net.Options) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("Driver Options:"))
		for k, v := range net.Options {
			lines = append(lines, fmt.Sprintf("  • %s: %s", theme.DimStyle.Render(k), v))
		}
	}

	// Labels
	if len(net.Labels) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("Labels:"))
		for k, v := range net.Labels {
			lines = append(lines, fmt.Sprintf("  • %s: %s", theme.DimStyle.Render(k), v))
		}
	}

	// Raw JSON
	if trimmed := strings.TrimSpace(rawJSON); trimmed != "" {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("── RAW DOCKER INSPECT JSON ──"))
		for _, l := range strings.Split(trimmed, "\n") {
			lines = append(lines, l)
		}
	}

	return lines
}

// BuildVolumeInspectLines converts volume data and inspect JSON into structured display lines.
func BuildVolumeInspectLines(vol monitor.DockerVolume, rawJSON string) []string {
	var lines []string

	lines = append(lines, fmt.Sprintf("%s %s",
		theme.DimStyle.Render("Volume Name:"), theme.BoldStyle.Render(vol.Name),
	))

	createdStr := vol.Created
	if createdStr == "" {
		createdStr = "-"
	}
	lines = append(lines, fmt.Sprintf("%s %s    %s %s    %s %s",
		theme.DimStyle.Render("Driver:"), theme.CyanStyle.Render(vol.Driver),
		theme.DimStyle.Render("Scope:"), vol.Scope,
		theme.DimStyle.Render("Created:"), createdStr,
	))

	// Storage section
	lines = append(lines, "")
	lines = append(lines, theme.BoldStyle.Render("Storage & Mountpoint:"))
	lines = append(lines, fmt.Sprintf("  %s %s",
		theme.DimStyle.Render("Mountpoint (Host):"), theme.GreenStyle.Render(vol.Mountpoint),
	))
	sizeDisplay := "-"
	if vol.Size >= 0 {
		sizeDisplay = monitor.FormatBytes(vol.Size)
	}
	refsDisplay := "-"
	if vol.RefCount >= 0 {
		refsDisplay = fmt.Sprintf("%d", vol.RefCount)
	}
	statusDisplay := theme.DimStyle.Render("unknown")
	if vol.RefCount > 0 || len(vol.Containers) > 0 {
		statusDisplay = theme.GreenStyle.Render("in-use")
	} else if vol.RefCount == 0 {
		statusDisplay = theme.YellowStyle.Render("unused (reclaimable)")
	}
	lines = append(lines, fmt.Sprintf("  %s %s    %s %s    %s %s",
		theme.DimStyle.Render("Size:"), theme.BoldStyle.Render(sizeDisplay),
		theme.DimStyle.Render("References:"), refsDisplay,
		theme.DimStyle.Render("Status:"), statusDisplay,
	))

	// Containers Mounted
	lines = append(lines, "")
	lines = append(lines, theme.BoldStyle.Render("Mounted By Containers:"))
	if len(vol.ContainerMounts) > 0 {
		for _, m := range vol.ContainerMounts {
			lines = append(lines, fmt.Sprintf("  • %s", m))
		}
	} else if len(vol.Containers) > 0 {
		for _, c := range vol.Containers {
			lines = append(lines, fmt.Sprintf("  • %s", c))
		}
	} else if vol.RefCount == 0 {
		lines = append(lines, "  "+theme.YellowStyle.Render("○ Not mounted by any container (reclaimable with prune)"))
	} else {
		lines = append(lines, "  "+theme.DimStyle.Render("○ No container mount metadata available"))
	}

	// Options
	if len(vol.Options) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("Driver Options:"))
		for k, v := range vol.Options {
			lines = append(lines, fmt.Sprintf("  • %s: %s", theme.DimStyle.Render(k), v))
		}
	}

	// Labels
	if len(vol.Labels) > 0 {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("Labels:"))
		for k, v := range vol.Labels {
			lines = append(lines, fmt.Sprintf("  • %s: %s", theme.DimStyle.Render(k), v))
		}
	}

	// Raw JSON
	if trimmed := strings.TrimSpace(rawJSON); trimmed != "" {
		lines = append(lines, "")
		lines = append(lines, theme.BoldStyle.Render("── RAW DOCKER INSPECT JSON ──"))
		for _, l := range strings.Split(trimmed, "\n") {
			lines = append(lines, l)
		}
	}

	return lines
}

// RenderInspectLines renders an inspection modal with custom title and lines.
func RenderInspectLines(title string, lines []string, scrollOpts ...int) string {
	var s strings.Builder

	s.WriteString(theme.TitleStyle.Render(title) + "\n\n")

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

// RenderInspect renders the detailed container inspection modal with optional scrolling.
func RenderInspect(d monitor.ContainerDetail, scrollOpts ...int) string {
	title := fmt.Sprintf("CONTAINER INSPECTION: %s", theme.BoldStyle.Render(d.Name))
	lines := BuildInspectLines(d)
	return RenderInspectLines(title, lines, scrollOpts...)
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
		strings.Contains(upperKey, "PWD") ||
		strings.Contains(upperKey, "SECRET") ||
		strings.Contains(upperKey, "TOKEN") ||
		strings.Contains(upperKey, "KEY") ||
		strings.Contains(upperKey, "AUTH") ||
		strings.Contains(upperKey, "CREDENTIAL") ||
		strings.Contains(upperKey, "SALT") ||
		strings.Contains(upperKey, "SIGNATURE") ||
		strings.Contains(upperKey, "CERT") ||
		strings.Contains(upperKey, "PRIVATE") ||
		strings.Contains(upperKey, "WEBHOOK") ||
		strings.Contains(upperKey, "DSN") ||
		strings.Contains(upperKey, "BEARER") ||
		strings.Contains(upperKey, "PIN") {
		return key + "=******"
	}

	// Redact embedded credentials in URLs (e.g. postgres://user:password@host/db)
	if strings.Contains(val, "://") && strings.Contains(val, "@") {
		if u, err := url.Parse(val); err == nil && u.User != nil {
			return key + "=" + u.Redacted()
		}
	}

	// Check for embedded password fields in connection strings
	upperVal := strings.ToUpper(val)
	if strings.Contains(upperVal, "PASSWORD=") || strings.Contains(upperVal, "PWD=") {
		return key + "=******"
	}

	return key + "=" + val
}
