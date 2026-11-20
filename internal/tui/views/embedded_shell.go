package views

import (
	"fmt"

	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// EmbeddedShellContext contains the terminal pane state.
type EmbeddedShellContext struct {
	ContainerName string
	Output        string
	Width         int
	Height        int
	Focused       bool
}

// RenderEmbeddedShell renders an interactive terminal pane inside Dockpulse.
func RenderEmbeddedShell(ctx EmbeddedShellContext) string {
	width := max(20, ctx.Width)
	height := max(6, ctx.Height)
	innerWidth := max(1, width-2)
	innerHeight := max(1, height-2)

	status := "DOCKPULSE FOCUSED"
	statusStyle := theme.DimStyle
	borderColor := lipgloss.Color("#6B7280")
	if ctx.Focused {
		status = "FOCUSED"
		statusStyle = theme.GreenStyle
		borderColor = lipgloss.Color("#22C55E")
	}

	header := theme.BoldStyle.Render(fmt.Sprintf("TERMINAL: %s", ctx.ContainerName)) +
		"  " + statusStyle.Render("["+status+"]")
	output := ctx.Output
	if output == "" {
		output = theme.DimStyle.Render("Starting shell...")
	}

	content := lipgloss.NewStyle().
		Width(innerWidth).
		Height(max(1, innerHeight-1)).
		MaxWidth(innerWidth).
		MaxHeight(max(1, innerHeight-1)).
		Render(header + "\n" + output)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(innerWidth).
		Height(innerHeight).
		MaxWidth(width).
		MaxHeight(height).
		Render(content)
}
