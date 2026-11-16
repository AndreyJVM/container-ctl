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

	headerText := fmt.Sprintf("  %-12s  %-24s  %-16s  %-10s  %-16s  %s",
		d.ColID, d.ColRepository, d.ColTag, d.ColSize, d.ColCreated, d.ColStatus)
	s.WriteString(headerText + "\n")
	s.WriteString("  " + theme.DimStyle.Render(strings.Repeat("─", 88)) + "\n")

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

		row := fmt.Sprintf("%-12s  %-24s  %-16s  %-10s  %-16s  %s",
			components.Truncate(img.ID, colWidthImageID),
			components.Truncate(img.Repository, colWidthImageRepo),
			components.Truncate(img.Tag, colWidthImageTag),
			components.Truncate(sizeStr, colWidthImageSize),
			components.Truncate(img.Created, colWidthImageDate),
			statusBadge,
		)

		if index == ctx.CursorIdx {
			s.WriteString(theme.GreenStyle.Render("► ") + theme.SelectedRowStyle.Render(row) + "\n")
		} else {
			// Color status badge when not selected
			var coloredStatus string
			if img.InUse {
				coloredStatus = theme.GreenStyle.Render(statusBadge)
			} else if img.Dangling {
				coloredStatus = theme.YellowStyle.Render(statusBadge)
			} else {
				coloredStatus = theme.DimStyle.Render(statusBadge)
			}
			coloredRow := fmt.Sprintf("%-12s  %-24s  %-16s  %-10s  %-16s  %s",
				components.Truncate(img.ID, colWidthImageID),
				components.Truncate(img.Repository, colWidthImageRepo),
				components.Truncate(img.Tag, colWidthImageTag),
				components.Truncate(sizeStr, colWidthImageSize),
				components.Truncate(img.Created, colWidthImageDate),
				coloredStatus,
			)
			s.WriteString("  " + coloredRow + "\n")
		}
	}

	if ctx.CursorIdx >= 0 && ctx.CursorIdx < len(ctx.Images) {
		img := ctx.Images[ctx.CursorIdx]
		s.WriteString("\n" + theme.TitleStyle.Render(d.ImageDetailsTitle) + "\n")
		s.WriteString(fmt.Sprintf("  %-18s %s\n", d.ImageFullIDLabel, valueOrDash(img.FullID)))
		s.WriteString(fmt.Sprintf("  %-18s %s\n", d.ImageDigestLabel, valueOrDash(img.Digest)))

		tagsStr := "<none>"
		if len(img.AllTags) > 0 {
			tagsStr = strings.Join(img.AllTags, ", ")
		}
		s.WriteString(fmt.Sprintf("  %-18s %s\n", d.ImageTagsLabel, tagsStr))

		containersStr := "-"
		if img.Containers >= 0 {
			if img.Containers > 0 {
				containersStr = fmt.Sprintf("%d %s", img.Containers, theme.GreenStyle.Render(d.ImageInUseBadge))
			} else {
				containersStr = fmt.Sprintf("%d %s", img.Containers, theme.DimStyle.Render(d.ImageUnusedBadge))
			}
		}
		s.WriteString(fmt.Sprintf("  %-18s %s\n", d.ImageContainersLabel, containersStr))
	}

	return s.String()
}
