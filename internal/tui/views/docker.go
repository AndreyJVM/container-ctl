package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

const (
	minCardWidth   = 80
	detailColWidth = 18
)

type containerTableLayout struct {
	contentWidth int
	idWidth      int
	nameWidth    int
	imageWidth   int
	statusWidth  int
	cpuWidth     int
	memWidth     int
	portsWidth   int
}

func computeContainerTableLayout(termWidth int) containerTableLayout {
	contentWidth := max(termWidth-4, minCardWidth)

	var idWidth, statusWidth, cpuWidth, memWidth, nameWidth, imageWidth int

	if contentWidth >= 110 {
		idWidth = 8
		statusWidth = 15
		cpuWidth = 6
		memWidth = 13
		nameWidth = 18
		imageWidth = 18

		baseUsed := 14 + idWidth + statusWidth + cpuWidth + memWidth + nameWidth + imageWidth
		extra := contentWidth - baseUsed - 16 // reserve at least 16 for ports
		if extra > 0 {
			nameBonus := extra * 25 / 100
			imgBonus := extra * 25 / 100
			nameWidth += nameBonus
			imageWidth += imgBonus
		}
	} else if contentWidth >= 90 {
		idWidth = 8
		statusWidth = 14
		cpuWidth = 6
		memWidth = 10
		extra := contentWidth - 90
		nameWidth = 14 + (extra * 2 / 10)
		imageWidth = 14 + (extra * 2 / 10)
	} else {
		// Compact mode (< 90 columns, e.g. 80-89)
		idWidth = 8
		statusWidth = 14
		cpuWidth = 6
		memWidth = 9
		nameWidth = 11
		imageWidth = 11
	}

	used := 14 + idWidth + nameWidth + imageWidth + statusWidth + cpuWidth + memWidth
	portsWidth := contentWidth - used
	if portsWidth < 6 {
		portsWidth = 6
		contentWidth = used + portsWidth
	}

	return containerTableLayout{
		contentWidth: contentWidth,
		idWidth:      idWidth,
		nameWidth:    nameWidth,
		imageWidth:   imageWidth,
		statusWidth:  statusWidth,
		cpuWidth:     cpuWidth,
		memWidth:     memWidth,
		portsWidth:   portsWidth,
	}
}

// CleanTablePorts formats raw ports string into a clean, compact representation for table display.
func CleanTablePorts(portsStr string) string {
	tr := strings.TrimSpace(portsStr)
	if tr == "" {
		return "-"
	}

	rawParts := strings.Split(tr, ",")
	seen := make(map[string]bool)
	var result []string

	for _, p := range rawParts {
		entry := strings.TrimSpace(p)
		if entry == "" {
			continue
		}

		// Deduplicate IPv6 if identical IPv4 mapping exists
		if strings.HasPrefix(entry, "[::]:") || strings.HasPrefix(entry, "::: ") {
			v4Suffix := strings.TrimPrefix(entry, "[::]:")
			if seen["0.0.0.0:"+v4Suffix] || seen[v4Suffix] {
				continue
			}
		}

		// Simplify 0.0.0.0: prefix for table brevity
		cleanEntry := entry
		if strings.HasPrefix(cleanEntry, "0.0.0.0:") {
			cleanEntry = strings.TrimPrefix(cleanEntry, "0.0.0.0:")
		}

		if seen[cleanEntry] || seen[entry] {
			continue
		}
		seen[cleanEntry] = true
		seen[entry] = true
		result = append(result, cleanEntry)
	}

	if len(result) == 0 {
		return "-"
	}
	return strings.Join(result, ", ")
}

func padRightVisual(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func padLeftVisual(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// DockerContext contains data needed to render the Docker Containers tab.
type DockerContext struct {
	DockerInfo           monitor.DockerSummary
	Containers           []monitor.ContainerItem
	TotalContainersCount int
	CursorIdx            int
	IsFiltering          bool
	FilterQuery          string
	SortMode             int
	SortNames            []string
	Width                int
}

// RenderDocker renders the Docker Containers tab.
func RenderDocker(ctx DockerContext) string {
	var s strings.Builder

	renderDockerHeader(&s)

	if len(ctx.DockerInfo.Containers) == 0 {
		renderEmptyContainersNotice(&ctx, &s)
		return s.String()
	}

	renderFilterAndSortBadges(&ctx, &s)

	if len(ctx.Containers) == 0 {
		renderNoContainersFoundNotice(&ctx, &s)
		return s.String()
	}

	renderContainerTable(&ctx, &s)
	renderContainerDetailsCard(&ctx, &s)

	return s.String()
}

func renderDockerHeader(s *strings.Builder) {
	s.WriteString(theme.TitleStyle.Render("DOCKER CONTAINERS") + "\n\n")
}

func renderEmptyContainersNotice(ctx *DockerContext, s *strings.Builder) {
	if ctx.DockerInfo.Status == "running" {
		s.WriteString("  " + theme.DimStyle.Render("No running containers found.") + "\n")
	} else {
		s.WriteString("  " + theme.YellowStyle.Render("Docker daemon is stopped or unreachable.") + "\n")
	}
}

func renderFilterAndSortBadges(ctx *DockerContext, s *strings.Builder) {
	if ctx.IsFiltering {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("[Search: %s█]", ctx.FilterQuery)) + "  " + theme.DimStyle.Render("(Enter: apply, Esc: cancel)") + "\n\n")
		return
	}

	var statusBadges []string
	if ctx.FilterQuery != "" {
		statusBadges = append(statusBadges, theme.ActiveTabStyle.Render(fmt.Sprintf("[Filter: %s]", ctx.FilterQuery))+"  "+theme.DimStyle.Render(fmt.Sprintf("(%d of %d containers)", len(ctx.Containers), ctx.TotalContainersCount)))
	}
	if ctx.SortMode > 0 && ctx.SortMode < len(ctx.SortNames) {
		statusBadges = append(statusBadges, theme.CyanStyle.Render(fmt.Sprintf("Sort: %s", ctx.SortNames[ctx.SortMode])))
	}
	if len(statusBadges) > 0 {
		s.WriteString("  " + strings.Join(statusBadges, "   ") + "\n\n")
	}
}

func renderNoContainersFoundNotice(ctx *DockerContext, s *strings.Builder) {
	s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("No containers found matching %q.", ctx.FilterQuery)) + "\n")
	s.WriteString("  " + theme.DimStyle.Render("Press Esc to clear search filter.") + "\n")
}

func renderContainerTable(ctx *DockerContext, s *strings.Builder) {
	layout := computeContainerTableLayout(ctx.Width)

	headerText := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %*s  %-*s  %-*s",
		layout.idWidth, "ID",
		layout.nameWidth, "NAME",
		layout.imageWidth, "IMAGE",
		layout.statusWidth, "STATUS",
		layout.cpuWidth, "CPU",
		layout.memWidth, "MEM USAGE",
		layout.portsWidth, "PORTS",
	)
	s.WriteString(theme.DimStyle.Render(headerText) + "\n")
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", layout.contentWidth)) + "\n")

	for i, c := range ctx.Containers {
		isSelected := (i == ctx.CursorIdx)
		s.WriteString(renderContainerRow(c, isSelected, layout))
	}
}

func formatContainerMetrics(c monitor.ContainerItem) (string, string) {
	if c.State != "running" {
		return "-", "-"
	}

	cpuRaw := fmt.Sprintf("%.1f%%", c.CPUPerc)
	var memRaw string
	if c.MemLimitMB > 0 {
		memRaw = fmt.Sprintf("%.1fM (%.1f%%)", c.MemUsageMB, c.MemPerc)
	} else {
		memRaw = fmt.Sprintf("%.1fM", c.MemUsageMB)
	}
	return cpuRaw, memRaw
}

func renderContainerRow(c monitor.ContainerItem, isSelected bool, layout containerTableLayout) string {
	prefix := "  "
	if isSelected {
		prefix = theme.GreenStyle.Render("► ")
	}

	cID := padRightVisual(components.Truncate(c.ID, layout.idWidth), layout.idWidth)
	cName := padRightVisual(components.Truncate(c.Name, layout.nameWidth), layout.nameWidth)
	cImg := padRightVisual(components.Truncate(c.Image, layout.imageWidth), layout.imageWidth)

	statusText, statusStyle := formatContainerStatus(c.State, c.Health, c.Status)
	statusPadded := padRightVisual(components.Truncate(statusText, layout.statusWidth), layout.statusWidth)

	cpuRaw, memRaw := formatContainerMetrics(c)
	cpuPadded := padLeftVisual(cpuRaw, layout.cpuWidth)
	memPadded := padRightVisual(components.Truncate(memRaw, layout.memWidth), layout.memWidth)

	portsClean := CleanTablePorts(c.Ports)
	portsTrunc := components.Truncate(portsClean, layout.portsWidth)
	portsPadded := padRightVisual(portsTrunc, layout.portsWidth)

	if isSelected {
		plainRow := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %*s  %-*s  %-*s",
			layout.idWidth, cID,
			layout.nameWidth, cName,
			layout.imageWidth, cImg,
			layout.statusWidth, statusPadded,
			layout.cpuWidth, cpuPadded,
			layout.memWidth, memPadded,
			layout.portsWidth, portsPadded,
		)
		return fmt.Sprintf("%s%s\n", prefix, theme.SelectedRowStyle.Render(plainRow))
	}

	portsStyled := portsPadded
	if portsClean == "-" {
		portsStyled = theme.DimStyle.Render(portsPadded)
	}

	row := fmt.Sprintf("%s  %s  %s  %s  %s  %s  %s",
		cID,
		theme.BoldStyle.Render(cName),
		theme.DimStyle.Render(cImg),
		statusStyle.Render(statusPadded),
		styleCPU(c.CPUPerc, c.State, cpuPadded),
		theme.DimStyle.Render(memPadded),
		portsStyled,
	)
	return fmt.Sprintf("%s%s\n", prefix, row)
}
