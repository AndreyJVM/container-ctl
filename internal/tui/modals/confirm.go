package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
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

// RenderRemoveContainerBatch renders confirmation modal for removing multiple containers.
func RenderRemoveContainerBatch(items []monitor.ContainerItem) string {
	var s strings.Builder
	title := fmt.Sprintf("CONFIRM REMOVAL OF %d CONTAINERS", len(items))
	s.WriteString(theme.TitleStyle.Render(title) + "\n\n")

	var names []string
	const maxNames = 5
	for i, c := range items {
		if i >= maxNames {
			names = append(names, fmt.Sprintf("... (+%d more)", len(items)-maxNames))
			break
		}
		names = append(names, c.Name)
	}

	s.WriteString(fmt.Sprintf("  Selected: %s\n\n", theme.BoldStyle.Render(strings.Join(names, ", "))))
	s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render("Warning:"), "All selected containers and their non-persisted state will be permanently deleted."))
	s.WriteString(fmt.Sprintf("  Are you sure you want to remove %d container(s)?\n\n", len(items)))
	s.WriteString(fmt.Sprintf("  %s %s (Remove All)    %s %s\n", theme.RedStyle.Render("[Y]"), "Confirm", theme.DimStyle.Render("[N / Esc]"), "Cancel"))
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

// RenderRemoveNetwork renders the Docker network deletion confirmation modal.
func RenderRemoveNetwork(networkName, networkID string, containersCount int) string {
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render("CONFIRM NETWORK REMOVAL") + "\n\n")

	if containersCount > 0 {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.RedStyle.Render("Warning:"), fmt.Sprintf("Network is currently in use by %d container(s). Removal may fail until disconnected.", containersCount)))
	} else {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render("Warning:"), "The Docker network configuration will be permanently removed (docker network rm)."))
	}

	label := networkName
	if networkID != "" && networkID != networkName {
		label = fmt.Sprintf("%s (%s)", networkName, networkID)
	}
	s.WriteString(fmt.Sprintf("  Are you sure you want to remove network %s?\n\n", theme.BoldStyle.Render(label)))
	s.WriteString(fmt.Sprintf("  %s %s (Remove Network)    %s %s\n", theme.RedStyle.Render("[Y]"), "Confirm", theme.DimStyle.Render("[N / Esc]"), "Cancel"))
	return theme.BoxStyle.Render(s.String())
}

// RenderRemoveVolume renders the Docker volume deletion confirmation modal.
func RenderRemoveVolume(volumeName string, inUse bool) string {
	var s strings.Builder
	s.WriteString(theme.TitleStyle.Render("CONFIRM VOLUME REMOVAL") + "\n\n")

	if inUse {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.RedStyle.Render("Warning:"), "Volume is currently referenced by container(s). Force removal (-f) will be applied."))
	} else {
		s.WriteString(fmt.Sprintf("  %s %s\n\n", theme.YellowStyle.Render("Warning:"), "The volume and all stored persistent data will be permanently deleted (docker volume rm)."))
	}

	s.WriteString(fmt.Sprintf("  Are you sure you want to remove volume %s?\n\n", theme.BoldStyle.Render(volumeName)))
	s.WriteString(fmt.Sprintf("  %s %s (Remove Volume)    %s %s\n", theme.RedStyle.Render("[Y]"), "Confirm", theme.DimStyle.Render("[N / Esc]"), "Cancel"))
	return theme.BoxStyle.Render(s.String())
}
