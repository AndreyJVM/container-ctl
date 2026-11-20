package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

// RenderComposeDown renders the compose stack down confirmation modal.
func RenderComposeDown(target string) string {
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render("CONFIRM DOCKER COMPOSE DOWN") + "\n\n")
	s.WriteString(fmt.Sprintf("  Stack: %s\n\n", theme.BoldStyle.Render(target)))
	s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render("Warning:"), "This will stop and remove all containers, networks, and volumes defined in the project."))
	s.WriteString(fmt.Sprintf("  Are you sure you want to run compose down on %s?\n\n", theme.BoldStyle.Render(target)))
	s.WriteString(fmt.Sprintf("  %s %s (Down)    %s %s\n", theme.GreenStyle.Render("[Y]"), "Confirm", theme.DimStyle.Render("[N / Esc]"), "Cancel"))
	return theme.BoxStyle.Render(s.String())
}
