package components

import (
	"fmt"
	"math"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// GaugeStyle returns the appropriate Lip Gloss style for a given percentage threshold.
// < 50%: Green
// 50% - 79%: Yellow
// >= 80%: Red
func GaugeStyle(percent float64) lipgloss.Style {
	if percent >= 80.0 {
		return theme.RedStyle
	} else if percent >= 50.0 {
		return theme.YellowStyle
	}
	return theme.GreenStyle
}

// RenderGauge renders a colored progress bar gauge of the given width (in character blocks)
// enclosed in brackets, e.g. [██████░░░░░░].
// percent is clamped between 0.0 and 100.0.
// If isRunning is false, a dimmed placeholder [────────────────] is returned.
func RenderGauge(percent float64, width int, isRunning bool) string {
	if width <= 0 {
		return ""
	}

	if !isRunning {
		return theme.DimStyle.Render("[" + strings.Repeat("─", width) + "]")
	}

	clamped := math.Max(0.0, math.Min(100.0, percent))
	filledCount := int(math.Round((clamped / 100.0) * float64(width)))
	if clamped > 0 && filledCount == 0 {
		filledCount = 1
	}
	if filledCount > width {
		filledCount = width
	}
	emptyCount := width - filledCount

	style := GaugeStyle(clamped)
	filledStr := style.Render(strings.Repeat("█", filledCount))
	emptyStr := theme.DimStyle.Render(strings.Repeat("░", emptyCount))

	return "[" + filledStr + emptyStr + "]"
}

// RenderPlainGauge renders an unstyled progress bar string [████░░░░] suitable for
// contexts where the outer row has its own background styling (e.g. selected table row).
func RenderPlainGauge(percent float64, width int, isRunning bool) string {
	if width <= 0 {
		return ""
	}

	if !isRunning {
		return "[" + strings.Repeat("─", width) + "]"
	}

	clamped := math.Max(0.0, math.Min(100.0, percent))
	filledCount := int(math.Round((clamped / 100.0) * float64(width)))
	if clamped > 0 && filledCount == 0 {
		filledCount = 1
	}
	if filledCount > width {
		filledCount = width
	}
	emptyCount := width - filledCount

	return "[" + strings.Repeat("█", filledCount) + strings.Repeat("░", emptyCount) + "]"
}

// FormatMB formats megabytes into a human-readable string (MB or GB).
func FormatMB(mb float64) string {
	if mb <= 0 {
		return "0 MB"
	}
	if mb >= 1024.0 {
		return fmt.Sprintf("%.2f GB", mb/1024.0)
	}
	return fmt.Sprintf("%.1f MB", mb)
}
