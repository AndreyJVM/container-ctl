package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
)

// ExecModalContext holds information required to render the interactive exec/attach modal.
type ExecModalContext struct {
	ContainerName string
	ContainerID   string
	CursorIdx     int
	IsCustomInput bool
	CustomCmd     string
}

// ExecMenuItem represents a selectable execution target or shell.
type ExecMenuItem struct {
	Key         string
	Title       string
	Description string
}

// GetExecMenuItems returns the list of available shell/exec modes.
func GetExecMenuItems() []ExecMenuItem {
	return []ExecMenuItem{
		{
			Key:         "1",
			Title:       "Embedded Bash / Sh",
			Description: "Recommended: integrated terminal inside Dockpulse",
		},
		{
			Key:         "2",
			Title:       "POSIX Shell (/bin/sh)",
			Description: "Interactive POSIX shell inside Dockpulse",
		},
		{
			Key:         "3",
			Title:       "Zsh Shell (/bin/zsh)",
			Description: "Interactive Z shell inside Dockpulse",
		},
		{
			Key:         "4",
			Title:       "Docker Attach",
			Description: "Attach stdout/stderr/stdin directly",
		},
		{
			Key:         "5",
			Title:       "Root Shell (-u 0)",
			Description: "Embedded root session inside container",
		},
		{
			Key:         "6",
			Title:       "Custom Command...",
			Description: "Execute arbitrary command interactively",
		},
		{
			Key:         "7",
			Title:       "External Bash / Sh",
			Description: "Suspends Dockpulse and runs in your host terminal",
		},
	}
}

// RenderExec renders the interactive exec / attach selection modal.
func RenderExec(ctx ExecModalContext) string {
	var s strings.Builder

	s.WriteString(theme.TitleStyle.Render("EXECUTE INTERACTIVE SHELL") + "\n\n")
	s.WriteString(fmt.Sprintf("  Container: %s (%s)\n\n",
		theme.BoldStyle.Render(ctx.ContainerName),
		theme.DimStyle.Render(ctx.ContainerID),
	))

	if ctx.IsCustomInput {
		s.WriteString(fmt.Sprintf("  %s\n", theme.YellowStyle.Render(fmt.Sprintf("Enter custom command: %s", ctx.CustomCmd+"█"))))
		s.WriteString(fmt.Sprintf("  %s\n\n", theme.DimStyle.Render("(Press Enter to execute, Esc to cancel)")))
		return theme.BoxStyle.Render(s.String())
	}

	items := GetExecMenuItems()
	for i, item := range items {
		cursor := "  "
		itemStyle := theme.BoldStyle
		if i == ctx.CursorIdx {
			cursor = theme.GreenStyle.Render("▸ ")
			itemStyle = theme.ActiveTabStyle
		}

		shortcut := theme.BoldStyle.Render(fmt.Sprintf("[%s]", item.Key))
		s.WriteString(fmt.Sprintf("%s%s %-24s %s\n",
			cursor,
			shortcut,
			itemStyle.Render(item.Title),
			theme.DimStyle.Render(item.Description),
		))
	}

	s.WriteString("\n  " +
		theme.BoldStyle.Render("[1-7 / Enter]") + " Select Action  |  " +
		theme.BoldStyle.Render("[Esc / Q]") + " Cancel\n")

	return theme.BoxStyle.Render(s.String())
}
