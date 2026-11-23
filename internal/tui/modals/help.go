package modals

import (
	"fmt"
	"strings"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
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
				{key: "[Home] / [g] / [End] / [G]", desc: "Jump to start / end of list"},
				{key: "[PgUp] / [PgDown]", desc: "Scroll page up / down"},
			},
		},
		{
			title: "CONTAINER ACTIONS (Containers Tab)",
			items: []helpItem{
				{key: "[/]", desc: "Filter containers by name, image, or ID"},
				{key: "[O]", desc: "Cycle container sorting (CPU, Memory, Name, Status)"},
				{key: "[L]", desc: "View container logs (regex search with [/], syntax highlighting)"},
				{key: "[Enter] / [I]", desc: "Inspect detailed container metadata, mounts & env"},
				{key: "[w]", desc: "Open port URL in browser / copy"},
				{key: "[y]", desc: "Copy docker exec cmd / ID to clipboard"},
				{key: "[E]", desc: "Interactive shell inside container (bash / sh)"},
				{key: "[S]", desc: "Start / Stop container"},
				{key: "[P]", desc: "Restart container"},
				{key: "[K]", desc: "Force kill container (SIGKILL)"},
				{key: "[x]", desc: "Remove container"},
				{key: "[V]", desc: "Toggle lower details card (Tabs 1, 2, 4, 5)"},
			},
		},
		{
			title: "IMAGE ACTIONS (Images Tab)",
			items: []helpItem{
				{key: "[u]", desc: "Toggle filter for unused / reclaimable images"},
				{key: "[y]", desc: "Copy image repository:tag to clipboard"},
				{key: "[x]", desc: "Remove image (docker rmi)"},
			},
		},
		{
			title: "COMPOSE ACTIONS (Compose Tab)",
			items: []helpItem{
				{key: "[/]", desc: "Search and filter Compose projects"},
				{key: "[U]", desc: "Start selected Compose project (up)"},
				{key: "[S]", desc: "Stop selected Compose project"},
				{key: "[P]", desc: "Restart selected Compose project"},
				{key: "[D]", desc: "Stop and remove Compose project (down)"},
				{key: "[L]", desc: "View Compose stack aggregated logs"},
				{key: "[y]", desc: "Copy Compose project name to clipboard"},
			},
		},
		{
			title: "NETWORKS & VOLUMES (Tabs 4 & 5)",
			items: []helpItem{
				{key: "[Enter] / [i]", desc: "Inspect network or volume JSON details"},
				{key: "[/]", desc: "Search and filter networks or volumes"},
				{key: "[x]", desc: "Remove selected network or volume"},
				{key: "[y]", desc: "Copy network or volume name to clipboard"},
				{key: "[C]", desc: "Prune unused networks or volumes"},
			},
		},
		{
			title: "GENERAL ACTIONS",
			items: []helpItem{
				{key: "[C]", desc: "Prune unused Docker resources (interactive menu)"},
				{key: "[R]", desc: "Force refresh all Docker metrics immediately"},
				{key: "[?]", desc: "Toggle this help window"},
				{key: "[Q] / [Ctrl+C]", desc: "Exit container-ctl"},
			},
		},
	}

	if len(swarmVisible) == 0 || swarmVisible[0] {
		sections = append(sections[:len(sections)-1],
			helpSection{
				title: "SWARM ACTIONS (Swarm Tab)",
				items: []helpItem{
					{key: "[s]", desc: "Scale service replicas (interactive dialog)"},
					{key: "[+] / [-]", desc: "Quick increase / decrease service replicas (+1 / -1)"},
					{key: "[L]", desc: "View Swarm service logs"},
					{key: "[y]", desc: "Copy service name to clipboard"},
				},
			},
			sections[len(sections)-1],
		)
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
