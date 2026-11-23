package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/components"
	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
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
	showGauges   bool
}

func computeContainerTableLayout(termWidth int) containerTableLayout {
	contentWidth := max(termWidth-4, minCardWidth)

	var idWidth, statusWidth, cpuWidth, memWidth, nameWidth, imageWidth int
	showGauges := false

	idWidth = 8
	cpuWidth = 6
	memWidth = 8
	statusWidth = 13

	if contentWidth >= 120 {
		nameWidth = 20
		imageWidth = 22
		baseUsed := 14 + idWidth + statusWidth + cpuWidth + memWidth + nameWidth + imageWidth
		extra := contentWidth - baseUsed - 20
		if extra > 0 {
			nameWidth += extra * 30 / 100
			imageWidth += extra * 30 / 100
		}
	} else if contentWidth >= 100 {
		nameWidth = 15
		imageWidth = 15
	} else if contentWidth >= 90 {
		nameWidth = 13
		imageWidth = 13
	} else {
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
		showGauges:   showGauges,
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
		if strings.HasPrefix(entry, "[::]:") || strings.HasPrefix(entry, ":::") {
			v4Suffix := strings.TrimPrefix(strings.TrimPrefix(entry, "[::]:"), ":::")
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
	MaxRows              int
	HideDetails          bool
	SelectedIDs          map[string]struct{}
}

// RenderDocker renders the Docker Containers tab.
func RenderDocker(ctx DockerContext) string {
	var s strings.Builder

	renderDockerHeader(&ctx, &s)

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
	if !ctx.HideDetails {
		renderContainerDetailsCard(&ctx, &s)
	}

	return s.String()
}

func renderDockerHeader(ctx *DockerContext, s *strings.Builder) {
	total := len(ctx.Containers)
	if ctx.TotalContainersCount > 0 {
		total = ctx.TotalContainersCount
	}

	if total > 0 {
		var runningCount, stoppedCount, failedCount int
		for _, c := range ctx.DockerInfo.Containers {
			if failed, _ := monitor.IsContainerFailed(c.State, c.Status); failed || c.Health == monitor.HealthStatusUnhealthy {
				failedCount++
			} else if c.State == "running" {
				runningCount++
			} else {
				stoppedCount++
			}
		}

		summaryLine := theme.TitleStyle.Render(fmt.Sprintf("DOCKER CONTAINERS (%d)", total))
		var badges []string
		if runningCount > 0 {
			badges = append(badges, theme.GreenStyle.Render(fmt.Sprintf("● %d active", runningCount)))
		}
		if stoppedCount > 0 {
			badges = append(badges, theme.DimStyle.Render(fmt.Sprintf("○ %d stopped", stoppedCount)))
		}
		if failedCount > 0 {
			badges = append(badges, theme.RedStyle.Render(fmt.Sprintf("▲ %d attention", failedCount)))
		}

		if len(badges) > 0 {
			summaryLine += "    " + strings.Join(badges, "   ")
		}
		if len(ctx.SelectedIDs) > 0 {
			summaryLine += "   " + theme.CyanStyle.Render(fmt.Sprintf("● %d selected", len(ctx.SelectedIDs)))
		}
		if ctx.MaxRows > 0 && len(ctx.Containers) > ctx.MaxRows {
			start, end := components.CalculateTableWindow(len(ctx.Containers), ctx.CursorIdx, ctx.MaxRows)
			if badge := components.FormatWindowBadge(start, end, len(ctx.Containers)); badge != "" {
				summaryLine += "   " + theme.CyanStyle.Render(badge)
			}
		}
		s.WriteString(summaryLine + "\n\n")
		return
	}

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

	headerText := fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %*s  %*s  %-*s",
		layout.idWidth, "ID",
		layout.nameWidth, "NAME",
		layout.imageWidth, "IMAGE",
		layout.statusWidth, "STATUS",
		layout.cpuWidth, "CPU",
		layout.memWidth, "MEM",
		layout.portsWidth, "PORTS",
	)
	s.WriteString(theme.DimStyle.Render(headerText) + "\n")
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", layout.contentWidth)) + "\n")

	start, end := 0, len(ctx.Containers)
	if ctx.MaxRows > 0 && len(ctx.Containers) > ctx.MaxRows {
		start, end = components.CalculateTableWindow(len(ctx.Containers), ctx.CursorIdx, ctx.MaxRows)
	}

	for i := start; i < end; i++ {
		c := ctx.Containers[i]
		isSelected := (i == ctx.CursorIdx)
		isMarked := false
		if ctx.SelectedIDs != nil {
			_, isMarked = ctx.SelectedIDs[c.ID]
		}
		s.WriteString(renderContainerRow(c, isSelected, isMarked, layout))
	}

	if start > 0 || end < len(ctx.Containers) {
		var hints []string
		if start > 0 {
			hints = append(hints, fmt.Sprintf("▲ %d more above", start))
		}
		if end < len(ctx.Containers) {
			hints = append(hints, fmt.Sprintf("▼ %d more below", len(ctx.Containers)-end))
		}
		s.WriteString("  " + theme.DimStyle.Render(strings.Join(hints, "  │  ")) + "\n")
	}
}

func formatContainerMetrics(c monitor.ContainerItem) (string, string) {
	if c.State != "running" {
		return "-", "-"
	}

	cpuRaw := fmt.Sprintf("%.1f%%", c.CPUPerc)
	var memRaw string
	if c.MemUsageMB >= 1024 {
		memRaw = fmt.Sprintf("%.2fG", c.MemUsageMB/1024)
	} else {
		memRaw = fmt.Sprintf("%.1fM", c.MemUsageMB)
	}
	return cpuRaw, memRaw
}

func renderContainerRow(c monitor.ContainerItem, isSelected, isMarked bool, layout containerTableLayout) string {
	prefix := "  "
	if isMarked && isSelected {
		prefix = theme.CyanStyle.Render("►●")
	} else if isMarked {
		prefix = theme.CyanStyle.Render("● ")
	} else if isSelected {
		prefix = theme.GreenStyle.Render("► ")
	}

	cID := padRightVisual(components.Truncate(c.ID, layout.idWidth), layout.idWidth)
	cName := padRightVisual(components.Truncate(c.Name, layout.nameWidth), layout.nameWidth)
	cImg := padRightVisual(components.Truncate(c.Image, layout.imageWidth), layout.imageWidth)

	statusText, statusStyle := formatContainerStatus(c.State, c.Health, c.Status)
	statusPadded := padRightVisual(components.Truncate(statusText, layout.statusWidth), layout.statusWidth)

	cpuRaw, memRaw := formatContainerMetrics(c)
	cpuColPlain := padLeftVisual(cpuRaw, layout.cpuWidth)
	memColPlain := padLeftVisual(memRaw, layout.memWidth)
	cpuStyled := styleCPU(c.CPUPerc, c.State, cpuColPlain)
	memStyled := styleMem(c.MemUsageMB, c.MemPerc, c.State, memColPlain)

	portsClean := CleanTablePorts(c.Ports)
	portsTrunc := components.Truncate(portsClean, layout.portsWidth)
	portsPadded := padRightVisual(portsTrunc, layout.portsWidth)

	if isSelected {
		plainRow := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %*s  %*s  %-*s",
			layout.idWidth, cID,
			layout.nameWidth, cName,
			layout.imageWidth, cImg,
			layout.statusWidth, statusPadded,
			layout.cpuWidth, cpuColPlain,
			layout.memWidth, memColPlain,
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
		cpuStyled,
		memStyled,
		portsStyled,
	)
	return fmt.Sprintf("%s%s\n", prefix, row)
}

func styleMem(mb, perc float64, state, paddedStr string) string {
	if state != "running" || mb <= 0 {
		return theme.DimStyle.Render(paddedStr)
	}
	if perc >= 85 {
		return theme.RedStyle.Render(paddedStr)
	} else if perc >= 70 {
		return theme.YellowStyle.Render(paddedStr)
	}
	return theme.DimStyle.Render(paddedStr)
}
