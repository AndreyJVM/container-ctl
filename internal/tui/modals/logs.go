package modals

import (
	"fmt"
	"strings"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/tui/components"
	"github.com/AndreyJVM/dockpulse/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Constants for log modal dimensions and constraints.
const (
	DefaultMaxVisibleLogs = 18
	MinLogLineWidth       = 40
	LogModalBorderMargin  = 8
)

// RenderLogs renders the interactive container/service log viewer modal with search and filter support.
func RenderLogs(lines []string, scroll int, targetName, targetID string, width int, dict *i18n.Dictionary, filterOpts ...string) string {
	var s strings.Builder
	d := getModalDictionary(dict)

	title := fmt.Sprintf(d.LogsTitle, theme.BoldStyle.Render(targetName), theme.DimStyle.Render(targetID))
	s.WriteString(theme.TitleStyle.Render(title) + "\n\n")

	filterQuery := ""
	isInputActive := false
	if len(filterOpts) > 0 {
		filterQuery = strings.TrimSpace(filterOpts[0])
	}
	if len(filterOpts) > 1 && filterOpts[1] == "true" {
		isInputActive = true
	}

	displayLines := lines
	if filterQuery != "" {
		lowerQ := strings.ToLower(filterQuery)
		var matched []string
		for _, l := range lines {
			if strings.Contains(strings.ToLower(l), lowerQ) {
				matched = append(matched, l)
			}
		}
		displayLines = matched
	}

	if isInputActive {
		s.WriteString(fmt.Sprintf("  %s %s█  %s\n\n",
			theme.BoldStyle.Render("Search:"),
			theme.YellowStyle.Render(filterQuery),
			theme.DimStyle.Render("(Enter: confirm, Esc: clear)"),
		))
	} else if filterQuery != "" {
		s.WriteString(fmt.Sprintf("  %s %s (%d matches)  %s\n\n",
			theme.BoldStyle.Render("Filter:"),
			theme.CyanStyle.Render(filterQuery),
			len(displayLines),
			theme.DimStyle.Render("[/] to edit, [Esc] to clear"),
		))
	}

	if len(displayLines) == 0 {
		if filterQuery != "" {
			s.WriteString(fmt.Sprintf("  %s %q\n", theme.DimStyle.Render("No log lines match filter:"), filterQuery))
		} else {
			s.WriteString("  " + theme.DimStyle.Render(d.NoLogsOutput) + "\n")
		}
		return theme.BoxStyle.Render(s.String())
	}

	start, end := calculateLogRange(len(displayLines), scroll, DefaultMaxVisibleLogs)
	maxLineWidth := max(MinLogLineWidth, width-LogModalBorderMargin)

	for i := start; i < end; i++ {
		line := displayLines[i]
		gutter := getLogGutterStyle(line).Render("│")
		s.WriteString(fmt.Sprintf(" %s %s\n", gutter, components.Truncate(line, maxLineWidth)))
	}

	if filterQuery != "" {
		s.WriteString(fmt.Sprintf("\n  %s\n", theme.DimStyle.Render(fmt.Sprintf("[Lines %d-%d of %d matches (total %d lines)]", start+1, end, len(displayLines), len(lines)))))
	} else {
		s.WriteString(fmt.Sprintf("\n  %s\n", theme.DimStyle.Render(fmt.Sprintf(d.LogsRangeCount, start+1, end, len(lines)))))
	}

	return theme.BoxStyle.Render(s.String())
}

func getModalDictionary(dict *i18n.Dictionary) *i18n.Dictionary {
	if dict != nil {
		return dict
	}
	d := i18n.Get()
	return &d
}

func calculateLogRange(totalLines, scroll, maxVisible int) (int, int) {
	if totalLines <= 0 {
		return 0, 0
	}
	maxScroll := max(0, totalLines-1)
	start := scroll
	if start < 0 {
		start = 0
	}
	if start > maxScroll {
		start = maxScroll
	}
	end := min(start+maxVisible, totalLines)
	return start, end
}

func getLogGutterStyle(line string) lipgloss.Style {
	upper := strings.ToUpper(line)
	if strings.Contains(upper, "ERROR") || strings.Contains(upper, "FATAL") || strings.Contains(upper, "PANIC") {
		return theme.RedStyle
	}
	if strings.Contains(upper, "WARN") {
		return theme.YellowStyle
	}
	return theme.DimStyle
}
