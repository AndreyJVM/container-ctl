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
	TotalImagesCount int
	CursorIdx        int
	Error            string
	IsFiltering      bool
	FilterQuery      string
	Width            int
	I18n             *i18n.Dictionary
}

// RenderImages renders Docker images and details for the selected image.
func RenderImages(ctx ImagesContext) string {
	d := getDictionary(ctx.I18n)
	var s strings.Builder

	count := len(ctx.Images)
	if ctx.TotalImagesCount > 0 {
		count = ctx.TotalImagesCount
	}
	s.WriteString(theme.TitleStyle.Render(fmt.Sprintf(d.ImagesTabTitle, count)) + "\n\n")

	if ctx.Error != "" {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf(d.ImagesUnavailable, ctx.Error)) + "\n")
		return s.String()
	}

	if ctx.IsFiltering {
		s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf(d.FilterPrompt, ctx.FilterQuery)) + "  " + theme.DimStyle.Render(d.FilterHint) + "\n\n")
	} else if ctx.FilterQuery != "" {
		badge := theme.ActiveTabStyle.Render(fmt.Sprintf(d.FilterActiveBadge, ctx.FilterQuery)) + "  " +
			theme.DimStyle.Render(fmt.Sprintf(d.FilterResultsCount, len(ctx.Images), ctx.TotalImagesCount))
		s.WriteString("  " + badge + "\n\n")
	}

	if len(ctx.Images) == 0 {
		if ctx.FilterQuery != "" {
			s.WriteString("  " + theme.YellowStyle.Render(fmt.Sprintf(d.NoContainersFound, ctx.FilterQuery)) + "\n")
			s.WriteString("  " + theme.DimStyle.Render(d.PressEscToClear) + "\n")
		} else {
			s.WriteString("  " + theme.DimStyle.Render(d.NoImages) + "\n")
		}
		return s.String()
	}

	contentWidth := max(ctx.Width-4, 88)

	// Dynamically distribute extra space to repository and tag columns
	extraSpace := max(0, contentWidth-88)
	repoWidth := colWidthImageRepo + (extraSpace * 6 / 10)
	tagWidth := colWidthImageTag + (extraSpace * 4 / 10)

	headerText := fmt.Sprintf("  %-12s  %-*s  %-*s  %-10s  %-16s  %s",
		d.ColID, repoWidth, d.ColRepository, tagWidth, d.ColTag, d.ColSize, d.ColCreated, d.ColStatus)
	s.WriteString(headerText + "\n")
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", contentWidth)) + "\n")

	for index, img := range ctx.Images {
		sizeStr := "-"
		if img.Size >= 0 {
			sizeStr = monitor.FormatBytes(img.Size)
		}

		statusBadge := d.ImageUnusedBadge
		if img.InUse {
			statusBadge = d.ImageInUseBadge
		} else if img.Dangling {
			statusBadge = d.ImageDanglingBadge
		}

		idPadded := components.Truncate(img.ID, colWidthImageID)
		repoPadded := components.PadRight(components.Truncate(img.Repository, repoWidth), repoWidth)
		tagPadded := components.PadRight(components.Truncate(img.Tag, tagWidth), tagWidth)
		sizePadded := components.PadRight(components.Truncate(sizeStr, colWidthImageSize), colWidthImageSize)
		datePadded := components.PadRight(components.Truncate(img.Created, colWidthImageDate), colWidthImageDate)

		if index == ctx.CursorIdx {
			plainRow := fmt.Sprintf("%-12s  %-*s  %-*s  %-10s  %-16s  %s",
				idPadded, repoWidth, repoPadded, tagWidth, tagPadded, sizePadded, datePadded, statusBadge)
			s.WriteString(theme.GreenStyle.Render("► ") + theme.SelectedRowStyle.Render(plainRow) + "\n")
		} else {
			var coloredStatus string
			if img.InUse {
				coloredStatus = theme.GreenStyle.Render(statusBadge)
			} else if img.Dangling {
				coloredStatus = theme.YellowStyle.Render(statusBadge)
			} else {
				coloredStatus = theme.DimStyle.Render(statusBadge)
			}
			coloredRow := fmt.Sprintf("%-12s  %-*s  %-*s  %-10s  %-16s  %s",
				idPadded, repoWidth, repoPadded, tagWidth, tagPadded, sizePadded, datePadded, coloredStatus)
			s.WriteString("  " + coloredRow + "\n")
		}
	}

	if ctx.CursorIdx >= 0 && ctx.CursorIdx < len(ctx.Images) {
		img := ctx.Images[ctx.CursorIdx]
		s.WriteString("\n")

		cardWidth := contentWidth
		title := fmt.Sprintf(" %s: %s:%s (%s) ", d.ImageDetailsTitle, img.Repository, img.Tag, img.ID)
		topDashLen := cardWidth - lipgloss.Width(title) - 3
		if topDashLen < 2 {
			topDashLen = 2
		}
		s.WriteString("  " + theme.CyanStyle.Render("┌─") + theme.BoldStyle.Render(title) + theme.CyanStyle.Render(strings.Repeat("─", topDashLen)+"┐") + "\n")

		s.WriteString(renderDetailRow(d.ImageFullIDLabel, valueOrDash(img.FullID)))
		s.WriteString(renderDetailRow(d.ImageDigestLabel, valueOrDash(img.Digest)))

		tagsStr := "<none>"
		if len(img.AllTags) > 0 {
			tagsStr = strings.Join(img.AllTags, ", ")
		}
		s.WriteString(renderDetailRow(d.ImageTagsLabel, tagsStr))

		containersStr := "-"
		if img.Containers >= 0 {
			if img.Containers > 0 {
				containersStr = fmt.Sprintf("%d %s", img.Containers, theme.GreenStyle.Render(d.ImageInUseBadge))
			} else {
				containersStr = fmt.Sprintf("%d %s", img.Containers, theme.DimStyle.Render(d.ImageUnusedBadge))
			}
		}
		s.WriteString(renderDetailRow(d.ImageContainersLabel, containersStr))

		s.WriteString("  " + theme.CyanStyle.Render("└"+strings.Repeat("─", cardWidth-2)+"┘") + "\n")
	}

	return s.String()
}
