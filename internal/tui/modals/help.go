package modals

import (
	"fmt"
	"strings"
	"time"

	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

const helpKeyColumnWidth = 34

type helpItem struct {
	key  string
	desc string
}

type helpSection struct {
	title string
	items []helpItem
}

// RenderHelp renders the categorized and aligned keyboard shortcuts help dialog box.
func RenderHelp(refreshInterval time.Duration, swarmVisible ...bool) string {
	var s strings.Builder

	s.WriteString(theme.TitleStyle.Render("HOTKEYS: HELP & KEYBOARD SHORTCUTS") + "\n\n")

	tabKeys := "[1] - [6]"
	if len(swarmVisible) > 0 && !swarmVisible[0] {
		tabKeys = "[1] - [5]"
	}

	sections := []helpSection{
		{
			title: "NAVIGATION",
			items: []helpItem{
				{key: tabKeys, desc: "Switch directly to Tab (1-6)"},
				{key: "[Tab] / [Shift+Tab] / [←] / [→]", desc: "Switch to next / previous tab"},
				{key: "[↑] / [↓] / [k] / [j]", desc: "Navigate items in list"},
			},
		},
		{
			title: "CONTAINER ACTIONS (Containers Tab)",
			items: []helpItem{
				{key: "[/]", desc: "Filter containers by name, image, or ID"},
				{key: "[O]", desc: "Cycle container sorting (CPU, Memory, Name, Status)"},
				{key: "[L]", desc: "View container logs (search with [/], scroll with arrows)"},
				{key: "[Enter] / [I]", desc: "Inspect detailed container metadata, mounts & env"},
				{key: "[w]", desc: "Open port URL in browser / copy"},
				{key: "[y]", desc: "Copy docker exec cmd / ID to clipboard"},
				{key: "[E]", desc: "Interactive shell inside container (bash / sh)"},
				{key: "[S]", desc: "Start / Stop container"},
				{key: "[P]", desc: "Restart container"},
				{key: "[K]", desc: "Force kill container (SIGKILL)"},
				{key: "[x]", desc: "Remove container"},
			},
		},
		{
			title: "IMAGE ACTIONS (Images Tab)",
			items: []helpItem{
				{key: "[u]", desc: "Toggle filter for unused / reclaimable images"},
				{key: "[x]", desc: "Remove image (docker rmi)"},
			},
		},
		{
			title: "COMPOSE ACTIONS (Compose Tab)",
			items: []helpItem{
				{key: "[U]", desc: "Start selected Compose project (up)"},
				{key: "[D]", desc: "Stop and remove Compose project (down)"},
			},
		},
		{
			title: "GENERAL ACTIONS",
			items: []helpItem{
				{key: "[C]", desc: "Prune unused Docker resources (interactive menu)"},
				{key: "[R]", desc: "Force refresh all Docker metrics immediately"},
				{key: "[?]", desc: "Toggle this help window"},
				{key: "[Q] / [Esc]", desc: "Exit Dockpulse"},
			},
		},
	}

	for _, sec := range sections {
		s.WriteString(fmt.Sprintf("  %s\n", theme.CyanStyle.Render("• "+sec.title)))
		for _, item := range sec.items {
			padCount := helpKeyColumnWidth - len([]rune(item.key))
			padding := ""
			if padCount > 0 {
				padding = strings.Repeat(" ", padCount)
			}
			s.WriteString(fmt.Sprintf("    %s%s %s\n",
				theme.GreenStyle.Render(item.key),
				padding,
				item.desc,
			))
		}
		s.WriteString("\n")
	}

	s.WriteString(fmt.Sprintf("  Auto-refresh interval: %s\n", theme.YellowStyle.Render(refreshInterval.String())))

	return theme.BoxStyle.Render(s.String())
}
