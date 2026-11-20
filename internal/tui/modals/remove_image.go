package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

// RenderRemoveImage renders the Docker image deletion confirmation modal.
func RenderRemoveImage(imageName, imageID string, inUse bool) string {
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render("CONFIRM IMAGE REMOVAL") + "\n\n")

	if inUse {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.RedStyle.Render("Warning:"), "Warning: Image is currently in use by containers. Force removal (-f) will be applied."))
	} else {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render("Warning:"), "The Docker image will be deleted (docker rmi)."))
	}

	s.WriteString(fmt.Sprintf("  Are you sure you want to remove image %s?\n\n", theme.BoldStyle.Render(imageName)))
	s.WriteString(fmt.Sprintf("  %-18s %s\n", "ID:", imageID))
	s.WriteString(fmt.Sprintf("  %-18s %s\n\n", "Repository:", imageName))
	s.WriteString(fmt.Sprintf("  %s %s (Remove Image)    %s %s\n", theme.RedStyle.Render("[Y]"), "Confirm", theme.DimStyle.Render("[N / Esc]"), "Cancel"))
	return theme.BoxStyle.Render(s.String())
}
