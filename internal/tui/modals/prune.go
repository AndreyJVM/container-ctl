package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

const (
	pruneColTypeWidth   = 16
	pruneColSizeWidth   = 12
	pruneColActiveWidth = 14
	pruneTableWidth     = 58
)

// RenderPrune renders the Docker resource pruning menu and confirmation modal.
func RenderPrune(usage monitor.DockerDiskUsage, usageErr string, confirmStep int) string {
	var s strings.Builder

	s.WriteString(theme.TitleStyle.Render("CLEANUP & PRUNE DOCKER RESOURCES") + "\n\n")

	if usageErr != "" {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.DimStyle.Render("Status:"), theme.RedStyle.Render(usageErr)))
	} else {
		s.WriteString(renderPruneUsageTable(usage))
	}

	if confirmStep > 0 {
		s.WriteString(renderPruneConfirmStep(confirmStep))
	} else {
		s.WriteString(renderPruneActionMenu())
	}

	return theme.BoxStyle.Render(s.String())
}

func renderPruneRow(name string, item monitor.DockerDiskItem) string {
	return fmt.Sprintf("  %-16s %-12s %-14s %s\n",
		name,
		monitor.FormatBytes(item.TotalSize),
		fmt.Sprintf("%d / %d", item.ActiveCount, item.TotalCount),
		theme.YellowStyle.Render(monitor.FormatBytes(item.Reclaimable)),
	)
}

func renderPruneUsageTable(u monitor.DockerDiskUsage) string {
	var sb strings.Builder
	sb.WriteString("  " + theme.BoldStyle.Render("CURRENT RECLAIMABLE DISK SPACE") + "\n")
	sb.WriteString(fmt.Sprintf("  %-16s %-12s %-14s %s\n",
		theme.DimStyle.Render("TYPE"),
		theme.DimStyle.Render("TOTAL SIZE"),
		theme.DimStyle.Render("ACTIVE / TOTAL"),
		theme.DimStyle.Render("RECLAIMABLE"),
	))
	sb.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", pruneTableWidth)) + "\n")

	categories := []struct {
		name string
		item monitor.DockerDiskItem
	}{
		{"Images", u.Images},
		{"Containers", u.Containers},
		{"Local Volumes", u.Volumes},
		{"Build Cache", u.BuildCache},
	}

	for _, c := range categories {
		sb.WriteString(renderPruneRow(c.name, c.item))
	}

	totalSize := u.TotalSize
	totalRecl := u.TotalRecl
	if totalSize == 0 && totalRecl == 0 {
		totalSize = u.Images.TotalSize + u.Containers.TotalSize + u.Volumes.TotalSize + u.BuildCache.TotalSize
		totalRecl = u.Images.Reclaimable + u.Containers.Reclaimable + u.Volumes.Reclaimable + u.BuildCache.Reclaimable
	}

	sb.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", pruneTableWidth)) + "\n")
	sb.WriteString(fmt.Sprintf("  %-16s %-12s %-14s %s\n\n",
		theme.BoldStyle.Render("Total:"),
		theme.BoldStyle.Render(monitor.FormatBytes(totalSize)),
		"",
		theme.GreenStyle.Render(monitor.FormatBytes(totalRecl)),
	))

	return sb.String()
}

func getPruneActionName(step int) string {
	switch step {
	case 1:
		return "Prune Stopped Containers"
	case 2:
		return "Prune Dangling & Unused Images"
	case 3:
		return "Prune Unused Volumes"
	case 4:
		return "Prune System (All Unused Resources)"
	default:
		return ""
	}
}

func renderPruneConfirmStep(confirmStep int) string {
	actionName := getPruneActionName(confirmStep)
	var sb strings.Builder
	sb.WriteString("  " + theme.RedStyle.Render("WARNING: ") + theme.BoldStyle.Render(actionName) + "\n")
	sb.WriteString("  This action is permanent and cannot be undone.\n\n")
	sb.WriteString(fmt.Sprintf("  %s %s  |  %s %s\n",
		theme.GreenStyle.Render("[Y]"), "Confirm Prune",
		theme.DimStyle.Render("[N / Esc]"), "Cancel",
	))
	return sb.String()
}

func renderPruneActionMenu() string {
	var sb strings.Builder
	sb.WriteString("  " + theme.BoldStyle.Render("Choose a resource cleanup action:") + "\n")
	sb.WriteString(fmt.Sprintf("  %s %s\n", theme.GreenStyle.Render("[1]"), "Prune Stopped Containers (docker container prune)"))
	sb.WriteString(fmt.Sprintf("  %s %s\n", theme.GreenStyle.Render("[2]"), "Prune Dangling & Unused Images (docker image prune -a)"))
	sb.WriteString(fmt.Sprintf("  %s %s\n", theme.GreenStyle.Render("[3]"), "Prune Unused Volumes (docker volume prune)"))
	sb.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render("[4]"), "Prune Everything: All Unused Resources (docker system prune -a --volumes)"))
	sb.WriteString(fmt.Sprintf("  %s %s\n", theme.DimStyle.Render("[Esc / Q]"), "Close"))
	return sb.String()
}
