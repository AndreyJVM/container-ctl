package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
)

// RenderRemoveContainer renders the container deletion confirmation modal.
func RenderRemoveContainer(containerName, containerID string) string {
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render("CONFIRM CONTAINER REMOVAL") + "\n\n")
	label := fmt.Sprintf("%s (%s)", containerName, containerID)
	s.WriteString(fmt.Sprintf("  Container: %s\n\n", theme.BoldStyle.Render(label)))
	s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render("Warning:"), "The container and all non-persisted state will be permanently deleted (docker rm)."))
	s.WriteString(fmt.Sprintf("  Are you sure you want to remove container %s?\n\n", theme.BoldStyle.Render(containerName)))
	s.WriteString(fmt.Sprintf("  %s %s (Remove)    %s %s\n", theme.RedStyle.Render("[Y]"), "Confirm", theme.DimStyle.Render("[N / Esc]"), "Cancel"))
	return theme.BoxStyle.Render(s.String())
}
