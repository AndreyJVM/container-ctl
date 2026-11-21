package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
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
