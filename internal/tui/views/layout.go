package views

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
)

// HeaderContext contains the application and Docker status shown above every tab.
type HeaderContext struct {
	Version               string
	LastUpdate            string
	DockerInfo            monitor.DockerSummary
	FormattedDockerStatus string
}

// RenderHeader renders the application banner and persistent Docker status.
func RenderHeader(ctx HeaderContext) string {
	verStr := ctx.Version
	if verStr == "" {
		verStr = "v0.4.0"
	}
	appLine := theme.HeaderStyle.Render("Dockpulse") +
		theme.DimStyle.Render(fmt.Sprintf("  %s  ", verStr)) +
		theme.GreenStyle.Render("["+ctx.LastUpdate+"]")
	engineLabel := "Docker:"
	if ctx.DockerInfo.IsPodman {
		if ctx.DockerInfo.IsRootless {
			engineLabel = "Podman (rootless):"
		} else {
			engineLabel = "Podman:"
		}
	} else if ctx.DockerInfo.IsRootless {
		engineLabel = "Docker (rootless):"
	}

	statusLine := fmt.Sprintf("%s %s  |  %s  |  %s %d  |  %s %s",
		theme.DimStyle.Render(engineLabel), ctx.FormattedDockerStatus,
		theme.BoldStyle.Render(fmt.Sprintf("Containers: %d active, %d stopped", ctx.DockerInfo.RunningContainers, ctx.DockerInfo.StoppedContainers)),
		theme.DimStyle.Render("Images:"), ctx.DockerInfo.ImagesCount,
		theme.DimStyle.Render("Engine:"), ctx.DockerInfo.Version,
	)
	return appLine + "\n" + statusLine
}

// RenderTabs renders the tab navigation bar.
func RenderTabs(tabs []string, activeTab int, width int) string {
	var renderedTabs []string
	for i, tab := range tabs {
		tabIdx := i + 1
		if activeTab == tabIdx {
			renderedTabs = append(renderedTabs, theme.ActiveTabStyle.Render(fmt.Sprintf("[%s]", tab)))
		} else {
			renderedTabs = append(renderedTabs, theme.InactiveTabStyle.Render(fmt.Sprintf(" %s ", tab)))
		}
	}
	bar := strings.Join(renderedTabs, " ") + "\n" +
		theme.DimStyle.Render(strings.Repeat("─", max(width, 70)))
	return bar
}

// FooterContext contains parameters needed to render context-aware bottom hotkey hints.
type FooterContext struct {
	ActiveTab    int
	ActiveModal  string
	PruneConfirm int
	IsFiltering  bool
	FilterQuery  string
	Width        int
	ShellActive  bool
	ShellFocused bool
	SwarmVisible bool
}

// RenderFooter renders the bottom hotkeys guide.
func RenderFooter(ctx FooterContext) string {
	var content string

	if ctx.ActiveModal != "" {
		if ctx.ActiveModal == "compose_down" {
			content = theme.BoldStyle.Render("[Y]") + " Confirm (Down)  |  " +
				theme.BoldStyle.Render("[N / Esc]") + " Cancel"
		} else if ctx.ActiveModal == "remove_container" {
			content = theme.RedStyle.Render("[Y]") + " Confirm (Remove)  |  " +
				theme.BoldStyle.Render("[N / Esc]") + " Cancel"
		} else if ctx.ActiveModal == "remove_image" {
			content = theme.RedStyle.Render("[Y]") + " Confirm (Remove Image)  |  " +
				theme.BoldStyle.Render("[N / Esc]") + " Cancel"
		} else if ctx.ActiveModal == "remove_network" {
			content = theme.RedStyle.Render("[Y]") + " Confirm (Remove Network)  |  " +
				theme.BoldStyle.Render("[N / Esc]") + " Cancel"
		} else if ctx.ActiveModal == "remove_volume" {
			content = theme.RedStyle.Render("[Y]") + " Confirm (Remove Volume)  |  " +
				theme.BoldStyle.Render("[N / Esc]") + " Cancel"
		} else if ctx.ActiveModal == "exec" {
			content = theme.BoldStyle.Render("[1-7 / Enter]") + " Select Action  |  " +
				theme.BoldStyle.Render("[Esc / Q]") + " Cancel"
		} else if ctx.ActiveModal == "swarm_scale" {
			content = theme.BoldStyle.Render("[+ / - / ↑ / ↓]") + " Adjust  |  " +
				theme.BoldStyle.Render("[0-9]") + " Type  |  " +
				theme.BoldStyle.Render("[Enter]") + " Apply  |  " +
				theme.BoldStyle.Render("[Esc / Q]") + " Cancel"
		} else if ctx.ActiveModal == "prune" {
			if ctx.PruneConfirm > 0 {
				content = theme.BoldStyle.Render("[Y]") + " Confirm Prune  |  " +
					theme.BoldStyle.Render("[N / Esc]") + " Cancel"
			} else {
				content = theme.BoldStyle.Render("[1-4]") + " Select Action  |  " +
					theme.BoldStyle.Render("[Esc / Q]") + " Close"
			}
		} else if ctx.ActiveModal == "logs" {
			content = theme.BoldStyle.Render("[/]") + " Search (Regex)  |  " +
				theme.BoldStyle.Render("[Esc / Q]") + " Close  |  " +
				theme.BoldStyle.Render("[↑ / ↓ / PgUp / PgDn]") + " Scroll"
		} else {
			content = theme.BoldStyle.Render("[Esc / Q]") + " Close  |  " +
				theme.BoldStyle.Render("[↑ / ↓ / PgUp / PgDn]") + " Scroll"
		}
	} else if ctx.ShellActive {
		if ctx.ShellFocused {
			content = theme.BoldStyle.Render("[Ctrl+X]") + " Close terminal  |  " +
				theme.BoldStyle.Render("[Ctrl+]]") + " Return to Dockpulse  |  " +
				theme.BoldStyle.Render("[Ctrl+D]") + " Exit shell"
		} else {
			content = theme.BoldStyle.Render("[E]") + " Focus terminal  |  " +
				theme.BoldStyle.Render("[Ctrl+X]") + " Close terminal  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		}
	} else if ctx.ActiveTab == 1 {
		if ctx.IsFiltering {
			content = theme.BoldStyle.Render("[Enter]") + " Apply Filter  |  " +
				theme.BoldStyle.Render("[Esc]") + " Cancel"
		} else if ctx.FilterQuery != "" {
			row1 := theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[Esc]") + " Clear Filter  |  " +
				theme.BoldStyle.Render("[Enter/I]") + " Info  |  " +
				theme.BoldStyle.Render("[L]") + " Logs  |  " +
				theme.BoldStyle.Render("[E]") + " Shell  |  " +
				theme.BoldStyle.Render("[S]") + " Start/Stop  |  " +
				theme.BoldStyle.Render("[P]") + " Restart  |  " +
				theme.BoldStyle.Render("[K]") + " Kill  |  " +
				theme.BoldStyle.Render("[x]") + " Remove"
			row2 := theme.BoldStyle.Render("[O]") + " Sort  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[w]") + " Web  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
			content = row1 + "\n" + row2
		} else if ctx.Width > 0 && ctx.Width < 95 {
			row1 := theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[Enter/I]") + " Info  |  " +
				theme.BoldStyle.Render("[L]") + " Logs  |  " +
				theme.BoldStyle.Render("[E]") + " Shell  |  " +
				theme.BoldStyle.Render("[S]") + " Start/Stop  |  " +
				theme.BoldStyle.Render("[K]") + " Kill  |  " +
				theme.BoldStyle.Render("[x]") + " Remove"
			row2 := theme.BoldStyle.Render("[/]") + " Search  |  " +
				theme.BoldStyle.Render("[O]") + " Sort  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[P]") + " Restart  |  " +
				theme.BoldStyle.Render("[w]") + " Web  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
			content = row1 + "\n" + row2
		} else if ctx.Width >= 160 {
			content = theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[/]") + " Search  |  " +
				theme.BoldStyle.Render("[O]") + " Sort  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[Enter/I]") + " Info  |  " +
				theme.BoldStyle.Render("[L]") + " Logs  |  " +
				theme.BoldStyle.Render("[E]") + " Shell  |  " +
				theme.BoldStyle.Render("[S]") + " Start/Stop  |  " +
				theme.BoldStyle.Render("[P]") + " Restart  |  " +
				theme.BoldStyle.Render("[w]") + " Web  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[K]") + " Kill  |  " +
				theme.BoldStyle.Render("[x]") + " Remove  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		} else {
			row1 := theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[Enter/I]") + " Info  |  " +
				theme.BoldStyle.Render("[L]") + " Logs  |  " +
				theme.BoldStyle.Render("[E]") + " Shell  |  " +
				theme.BoldStyle.Render("[S]") + " Start/Stop  |  " +
				theme.BoldStyle.Render("[P]") + " Restart  |  " +
				theme.BoldStyle.Render("[K]") + " Kill  |  " +
				theme.BoldStyle.Render("[x]") + " Remove"
			row2 := theme.BoldStyle.Render("[/]") + " Search  |  " +
				theme.BoldStyle.Render("[O]") + " Sort  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[w]") + " Web  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
			content = row1 + "\n" + row2
		}
	} else if ctx.ActiveTab == 2 {
		if ctx.IsFiltering {
			content = theme.BoldStyle.Render("[Enter]") + " Apply Filter  |  " +
				theme.BoldStyle.Render("[Esc]") + " Cancel"
		} else if ctx.FilterQuery != "" {
			content = theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[Esc]") + " Reset Filter  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[x]") + " Remove  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		} else {
			content = theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[/]") + " Search  |  " +
				theme.BoldStyle.Render("[u]") + " Filter Unused  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[x]") + " Remove  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		}
	} else if ctx.ActiveTab == 3 {
		if ctx.IsFiltering {
			content = theme.BoldStyle.Render("[Enter]") + " Apply Filter  |  " +
				theme.BoldStyle.Render("[Esc]") + " Cancel"
		} else if ctx.FilterQuery != "" {
			content = theme.BoldStyle.Render("[↑ / ↓]") + " Select Stack  |  " +
				theme.BoldStyle.Render("[Esc]") + " Reset Filter  |  " +
				theme.BoldStyle.Render("[L]") + " Logs  |  " +
				theme.BoldStyle.Render("[S]") + " Stop  |  " +
				theme.BoldStyle.Render("[U]") + " Start  |  " +
				theme.BoldStyle.Render("[P]") + " Restart  |  " +
				theme.BoldStyle.Render("[D]") + " Down  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		} else {
			content = theme.BoldStyle.Render("[↑ / ↓]") + " Select Stack  |  " +
				theme.BoldStyle.Render("[/]") + " Search  |  " +
				theme.BoldStyle.Render("[L]") + " Logs  |  " +
				theme.BoldStyle.Render("[S]") + " Stop  |  " +
				theme.BoldStyle.Render("[U]") + " Start  |  " +
				theme.BoldStyle.Render("[P]") + " Restart  |  " +
				theme.BoldStyle.Render("[D]") + " Down  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		}
	} else if ctx.ActiveTab == 4 || ctx.ActiveTab == 5 {
		if ctx.IsFiltering {
			content = theme.BoldStyle.Render("[Enter]") + " Apply Filter  |  " +
				theme.BoldStyle.Render("[Esc]") + " Cancel"
		} else if ctx.FilterQuery != "" {
			content = theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[Enter/I]") + " Inspect  |  " +
				theme.BoldStyle.Render("[Esc]") + " Reset Filter  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[x]") + " Remove  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		} else {
			content = theme.BoldStyle.Render("[↑ / ↓]") + " Select  |  " +
				theme.BoldStyle.Render("[Enter/I]") + " Inspect  |  " +
				theme.BoldStyle.Render("[/]") + " Search  |  " +
				theme.BoldStyle.Render("[V]") + " Details  |  " +
				theme.BoldStyle.Render("[y]") + " Copy  |  " +
				theme.BoldStyle.Render("[x]") + " Remove  |  " +
				theme.BoldStyle.Render("[C]") + " Prune  |  " +
				theme.BoldStyle.Render("[R]") + " Refresh  |  " +
				theme.BoldStyle.Render("[?]") + " Help  |  " +
				theme.BoldStyle.Render("[Q]") + " Quit"
		}
	} else if ctx.ActiveTab == 6 {
		content = theme.BoldStyle.Render("[↑ / ↓]") + " Select Service  |  " +
			theme.BoldStyle.Render("[s]") + " Scale  |  " +
			theme.BoldStyle.Render("[+ / -]") + " Replicas  |  " +
			theme.BoldStyle.Render("[L]") + " Service Logs  |  " +
			theme.BoldStyle.Render("[y]") + " Copy  |  " +
			theme.BoldStyle.Render("[R]") + " Refresh  |  " +
			theme.BoldStyle.Render("[?]") + " Help  |  " +
			theme.BoldStyle.Render("[Q]") + " Quit"
	} else {
		tabRange := "1-5"
		if ctx.SwarmVisible {
			tabRange = "1-6"
		}
		content = theme.BoldStyle.Render("["+tabRange+" / Tab / ← →]") + " Tabs  |  " +
			theme.BoldStyle.Render("[R]") + " Refresh  |  " +
			theme.BoldStyle.Render("[?]") + " Help  |  " +
			theme.BoldStyle.Render("[Q]") + " Quit"
	}

	return "\n" + theme.DimStyle.Render(strings.Repeat("─", max(ctx.Width, 70))) + "\n" +
		theme.FooterStyle.Render(content)
}
