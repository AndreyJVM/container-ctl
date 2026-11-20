package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

const (
	colWidthID     = 8
	colWidthName   = 20
	colWidthImage  = 22
	colWidthStatus = 20
	colWidthCPU    = 7
	colWidthMem    = 14
	minPortsWidth  = 20
	minCardWidth   = 96
	detailColWidth = 18
)

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
	I18n                 *i18n.Dictionary
}

// RenderDocker renders the Docker Containers tab.
func RenderDocker(ctx DockerContext) string {
	var s strings.Builder
	dict := getDictionary(ctx.I18n)

	renderDockerHeader(dict, &s)

	if len(ctx.DockerInfo.Containers) == 0 {
		renderEmptyContainersNotice(&ctx, dict, &s)
		return s.String()
	}

	renderFilterAndSortBadges(&ctx, dict, &s)

	if len(ctx.Containers) == 0 {
		renderNoContainersFoundNotice(&ctx, dict, &s)
		return s.String()
	}

	renderContainerTable(&ctx, dict, &s)
	renderContainerDetailsCard(&ctx, dict, &s)

	return s.String()
}

func getDictionary(dict *i18n.Dictionary) *i18n.Dictionary {
	if dict != nil {
		return dict
	}
	d := i18n.Get()
	return &d
}

func renderDockerHeader(dict *i18n.Dictionary, s *strings.Builder) {
	s.WriteString(theme.TitleStyle.Render(dict.DockerTabTitle) + "\n")
	s.WriteString("\n")
}

func renderEmptyContainersNotice(ctx *DockerContext, dict *i18n.Dictionary, s *strings.Builder) {
	if ctx.DockerInfo.Status == "running" {
		s.WriteString("  " + theme.DimStyle.Render(dict.ContainersNotRunning) + "\n")
	} else {
		s.WriteString("  " + theme.YellowStyle.Render(dict.DockerDaemonDown) + "\n")
	}
}

func renderFilterAndSortBadges(ctx *DockerContext, dict *i18n.Dictionary, s *strings.Builder) {
	if ctx.IsFiltering {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf(dict.FilterPrompt, ctx.FilterQuery)) + "  " + theme.DimStyle.Render(dict.FilterHint) + "\n\n")
		return
	}

	var statusBadges []string
	if ctx.FilterQuery != "" {
		statusBadges = append(statusBadges, theme.ActiveTabStyle.Render(fmt.Sprintf(dict.FilterActiveBadge, ctx.FilterQuery))+"  "+theme.DimStyle.Render(fmt.Sprintf(dict.FilterResultsCount, len(ctx.Containers), ctx.TotalContainersCount)))
	}
	if ctx.SortMode > 0 && ctx.SortMode < len(ctx.SortNames) {
		statusBadges = append(statusBadges, theme.CyanStyle.Render(fmt.Sprintf(dict.SortBadge, ctx.SortNames[ctx.SortMode])))
	}
	if len(statusBadges) > 0 {
		s.WriteString("  " + strings.Join(statusBadges, "   ") + "\n\n")
	}
}

func renderNoContainersFoundNotice(ctx *DockerContext, dict *i18n.Dictionary, s *strings.Builder) {
	s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf(dict.NoContainersFound, ctx.FilterQuery)) + "\n")
	s.WriteString("  " + theme.DimStyle.Render(dict.PressEscToClear) + "\n")
}

func renderContainerTable(ctx *DockerContext, dict *i18n.Dictionary, s *strings.Builder) {
	headerText := fmt.Sprintf("  %-8s  %-20s  %-22s  %-20s  %-7s  %-14s  %s",
		dict.ColID,
		dict.ColName,
		dict.ColImage,
		dict.ColStatus,
		dict.ColCPU,
		dict.ColMem,
		dict.ColPorts,
	)
	s.WriteString(theme.DimStyle.Render(headerText) + "\n")
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", max(ctx.Width-4, minCardWidth))) + "\n")

	for i, c := range ctx.Containers {
		isSelected := (i == ctx.CursorIdx)
		s.WriteString(renderContainerRow(c, isSelected, ctx.Width))
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

func renderContainerRow(c monitor.ContainerItem, isSelected bool, termWidth int) string {
	prefix := "  "
	if isSelected {
		prefix = theme.GreenStyle.Render("► ")
	}

	cID := components.PadRight(components.Truncate(c.ID, colWidthID), colWidthID)
	cName := components.PadRight(components.Truncate(c.Name, colWidthName), colWidthName)
	cImg := components.PadRight(components.Truncate(c.Image, colWidthImage), colWidthImage)

	statusText, statusStyle := formatContainerStatus(c.State, c.Health, c.Status)
	statusPadded := components.PadRight(components.Truncate(statusText, colWidthStatus), colWidthStatus)

	cpuRaw, memRaw := formatContainerMetrics(c)
	cpuPadded := components.PadLeft(cpuRaw, colWidthCPU)
	memPadded := components.PadRight(components.Truncate(memRaw, colWidthMem), colWidthMem)
	portsPreview := components.Truncate(c.Ports, max(minPortsWidth, termWidth-105))

	if isSelected {
		plainRow := fmt.Sprintf("%-8s  %-20s  %-22s  %-20s  %-7s  %-14s  %s",
			cID, cName, cImg, statusPadded, cpuPadded, memPadded, portsPreview,
		)
		return fmt.Sprintf("%s%s\n", prefix, theme.SelectedRowStyle.Render(plainRow))
	}

	row := fmt.Sprintf("%-8s  %-20s  %-22s  %s  %s  %-14s  %s",
		cID,
		theme.BoldStyle.Render(cName),
		theme.DimStyle.Render(cImg),
		statusStyle.Render(statusPadded),
		styleCPU(c.CPUPerc, c.State, cpuPadded),
		theme.DimStyle.Render(memPadded),
		portsPreview,
	)
	return fmt.Sprintf("%s%s\n", prefix, row)
}
