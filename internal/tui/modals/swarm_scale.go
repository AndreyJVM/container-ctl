package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
)

// RenderSwarmScale renders the Swarm service scaling modal dialog.
func RenderSwarmScale(serviceName, serviceID string, currentReplicas, desiredReplicas int) string {
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render("SCALE DOCKER SWARM SERVICE") + "\n\n")

	label := serviceName
	if serviceID != "" && serviceID != serviceName {
		label = fmt.Sprintf("%s (%s)", serviceName, serviceID)
	}
	s.WriteString(fmt.Sprintf("  Service:          %s\n", theme.BoldStyle.Render(label)))
	s.WriteString(fmt.Sprintf("  Current Replicas: %s\n\n", theme.CyanStyle.Render(fmt.Sprintf("%d", currentReplicas))))

	diff := desiredReplicas - currentReplicas
	diffStr := ""
	if diff > 0 {
		diffStr = theme.GreenStyle.Render(fmt.Sprintf(" (+%d)", diff))
	} else if diff < 0 {
		diffStr = theme.RedStyle.Render(fmt.Sprintf(" (%d)", diff))
	} else {
		diffStr = theme.DimStyle.Render(" (no change)")
	}

	scaleDisplay := fmt.Sprintf("  Desired Replicas: %s%s\n\n",
		theme.BoldStyle.Render(fmt.Sprintf("[ %2d ]", desiredReplicas)),
		diffStr,
	)
	s.WriteString(scaleDisplay)

	s.WriteString(fmt.Sprintf("  %s %s\n", theme.YellowStyle.Render("Controls:"), "Use [+] / [↑] to increase, [-] / [↓] to decrease"))
	s.WriteString(fmt.Sprintf("            Use [0-9] to type replica count directly\n\n"))

	s.WriteString(fmt.Sprintf("  %s %s (Apply)    %s %s\n",
		theme.GreenStyle.Render("[Enter]"), "Confirm",
		theme.DimStyle.Render("[Esc / Q]"), "Cancel",
	))

	return theme.BoxStyle.Render(s.String())
}
