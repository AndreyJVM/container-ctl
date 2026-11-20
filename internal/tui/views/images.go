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
	colWidthImageID   = 12
	colWidthImageRepo = 24
	colWidthImageTag  = 16
	colWidthImageSize = 10
	colWidthImageDate = 16
)

// ImagesContext contains data needed to render the Docker Images tab.
type ImagesContext struct {
	Images           []monitor.DockerImage
	AllImages        []monitor.DockerImage
	TotalImagesCount int
	CursorIdx        int
	Error            string
	IsFiltering      bool
	FilterQuery      string
	OnlyUnused       bool
	Width            int
}

// RenderImages renders Docker images and details for the selected image.
func RenderImages(ctx ImagesContext) string {
	var s strings.Builder

	count := len(ctx.Images)
	if ctx.TotalImagesCount > 0 {
		count = ctx.TotalImagesCount
	}

	sourceImages := ctx.AllImages
	if len(sourceImages) == 0 {
		sourceImages = ctx.Images
	}

	var inUseCount, inUseSize int64
	var unusedCount, unusedSize int64
	for _, img := range sourceImages {
		if img.InUse {
			inUseCount++
			if img.Size > 0 {
				inUseSize += img.Size
			}
		} else {
			unusedCount++
			if img.Size > 0 {
				unusedSize += img.Size
			}
		}
	}

	headerLine := theme.TitleStyle.Render(fmt.Sprintf("DOCKER IMAGES (%d)", count))
	if len(sourceImages) > 0 {
		headerLine += fmt.Sprintf("  %s    %s",
			theme.GreenStyle.Render("● "+fmt.Sprintf("In Use: %d (%s)", inUseCount, monitor.FormatBytes(inUseSize))),
			theme.YellowStyle.Render("○ "+fmt.Sprintf("Reclaimable: %d (%s)", unusedCount, monitor.FormatBytes(unusedSize))),
		)
	}
	s.WriteString(headerLine + "\n\n")

	if ctx.Error != "" {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("Docker images are unavailable: %s", ctx.Error)) + "\n")
		return s.String()
	}

	if ctx.OnlyUnused {
		s.WriteString("  " + theme.ActiveTabStyle.Render("[Filter: Unused Images Only — press 'u' or 'Esc' to show all]") + "\n\n")
	}

	if ctx.IsFiltering {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("[Search: %s█]", ctx.FilterQuery)) + "  " + theme.DimStyle.Render("(Enter: apply, Esc: cancel)") + "\n\n")
	} else if ctx.FilterQuery != "" {
		badge := theme.ActiveTabStyle.Render(fmt.Sprintf("Filter: %s", ctx.FilterQuery)) + "  " +
			theme.DimStyle.Render(fmt.Sprintf("(Found: %d of %d, Esc: reset)", len(ctx.Images), ctx.TotalImagesCount))
		s.WriteString("  " + badge + "\n\n")
	}

	if len(ctx.Images) == 0 {
		if ctx.OnlyUnused {
			s.WriteString("  " + theme.GreenStyle.Render("No unused images detected. All images are currently in use!") + "\n")
		} else if ctx.FilterQuery != "" {
			s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf("No containers found matching %q.", ctx.FilterQuery)) + "\n")
			s.WriteString("  " + theme.DimStyle.Render("Press Esc to clear search filter.") + "\n")
		} else {
			s.WriteString("  " + theme.DimStyle.Render("No Docker images found.") + "\n")
		}
		return s.String()
	}

	contentWidth := max(ctx.Width-4, 88)

	// Dynamically distribute extra space to repository and tag columns
	extraSpace := max(0, contentWidth-88)
	repoWidth := colWidthImageRepo + (extraSpace * 6 / 10)
	tagWidth := colWidthImageTag + (extraSpace * 4 / 10)

	headerText := fmt.Sprintf("  %-12s  %-*s  %-*s  %-10s  %-16s  %s",
		"ID", repoWidth, "REPOSITORY", tagWidth, "TAG", "SIZE", "CREATED", "STATUS")
	s.WriteString(headerText + "\n")
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", contentWidth)) + "\n")

	for index, img := range ctx.Images {
		sizeStr := "-"
		if img.Size >= 0 {
			sizeStr = monitor.FormatBytes(img.Size)
		}

		var statusBadge string
		var plainStatus string
		if img.InUse {
			plainStatus = "● [In Use]"
			statusBadge = theme.GreenStyle.Render(plainStatus)
		} else if img.Dangling {
			plainStatus = "▲ [Dangling]"
			statusBadge = theme.RedStyle.Render(plainStatus)
		} else {
			plainStatus = "○ [Unused]"
			statusBadge = theme.YellowStyle.Render(plainStatus)
		}

		idPadded := components.Truncate(img.ID, colWidthImageID)
		repoPadded := components.PadRight(components.Truncate(img.Repository, repoWidth), repoWidth)
		tagPadded := components.PadRight(components.Truncate(img.Tag, tagWidth), tagWidth)
		sizePadded := components.PadRight(components.Truncate(sizeStr, colWidthImageSize), colWidthImageSize)
		datePadded := components.PadRight(components.Truncate(img.Created, colWidthImageDate), colWidthImageDate)

		if index == ctx.CursorIdx {
			plainRow := fmt.Sprintf("%-12s  %-*s  %-*s  %-10s  %-16s  %s",
				idPadded, repoWidth, repoPadded, tagWidth, tagPadded, sizePadded, datePadded, plainStatus)
			s.WriteString(theme.GreenStyle.Render("► ") + theme.SelectedRowStyle.Render(plainRow) + "\n")
		} else {
			displaySize := sizePadded
			if !img.InUse {
				displaySize = theme.YellowStyle.Render(sizePadded)
			}
			coloredRow := fmt.Sprintf("%-12s  %-*s  %-*s  %-10s  %-16s  %s",
				idPadded, repoWidth, repoPadded, tagWidth, tagPadded, displaySize, datePadded, statusBadge)
			s.WriteString("  " + coloredRow + "\n")
		}
	}

	if ctx.CursorIdx >= 0 && ctx.CursorIdx < len(ctx.Images) {
		img := ctx.Images[ctx.CursorIdx]
		s.WriteString("\n")

		cardWidth := contentWidth
		title := fmt.Sprintf(" SELECTED IMAGE DETAILS: %s:%s (%s) ", img.Repository, img.Tag, img.ID)
		topDashLen := cardWidth - lipgloss.Width(title) - 3
		if topDashLen < 2 {
			topDashLen = 2
		}
		s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

		s.WriteString(renderDetailRow("Full ID:", valueOrDash(img.FullID)))
		s.WriteString(renderDetailRow("Digest:", valueOrDash(img.Digest)))

		tagsStr := "<none>"
		if len(img.AllTags) > 0 {
			tagsStr = strings.Join(img.AllTags, ", ")
		}
		s.WriteString(renderDetailRow("All Tags:", tagsStr))

		containersStr := "-"
		if img.Containers >= 0 {
			if img.Containers > 0 {
				containersStr = fmt.Sprintf("%d %s", img.Containers, theme.GreenStyle.Render("[In Use]"))
			} else {
				containersStr = fmt.Sprintf("%d %s", img.Containers, theme.YellowStyle.Render("[Unused]"))
			}
		}
		s.WriteString(renderDetailRow("Containers:", containersStr))

		var actionStr string
		if img.InUse {
			actionStr = theme.GreenStyle.Render("● ") + fmt.Sprintf("In Use by %d container(s) — Cannot be removed while containers exist", img.Containers)
		} else {
			actionStr = theme.YellowStyle.Render("○ ") + "Unused (reclaimable) — Press 'x' to delete, 'C' to prune"
		}
		s.WriteString(renderDetailRow("Status / Action:", actionStr))

		s.WriteString("  " + theme.CyanStyle.Render("└"+strings.Repeat("─", cardWidth-2)+"┘") + "\n")
	}

	return s.String()
}
