package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

// RenderRemoveImage renders the Docker image deletion confirmation modal.
func RenderRemoveImage(imageName, imageID string, inUse bool, dict *i18n.Dictionary) string {
	d := getModalDictionary(dict)
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render(d.RemoveImageTitle) + "\n\n")

	if inUse {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.RedStyle.Render(d.WarningLabel), d.RemoveImageInUseWarning))
	} else {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render(d.WarningLabel), d.RemoveImageNotice))
	}

	s.WriteString(fmt.Sprintf("  "+d.RemoveImageConfirm+"\n\n", theme.BoldStyle.Render(imageName)))
	s.WriteString(fmt.Sprintf("  %-18s %s\n", d.ColID, imageID))
	s.WriteString(fmt.Sprintf("  %-18s %s\n\n", d.ColRepository, imageName))
	s.WriteString(fmt.Sprintf("  %s %s (Remove Image)    %s %s\n", theme.RedStyle.Render("[Y]"), d.FooterConfirm, theme.DimStyle.Render("[N / Esc]"), d.FooterCancel))
	return theme.BoxStyle.Render(s.String())
}
