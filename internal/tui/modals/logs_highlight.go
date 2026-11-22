package modals

import (
	"regexp"
	"strings"

	"github.com/AndreyJVM/container-ctl/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

var (
	// SearchHighlightStyle highlights search/regex matches with black text on amber background.
	SearchHighlightStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(lipgloss.Color("#FBBF24"))

	// TimestampStyle renders timestamps in logs with dim cyan styling.
	TimestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// HTTPMethodStyle renders HTTP request methods with bold magenta styling.
	HTTPMethodStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#D946EF"))

	// Regex for detecting ISO8601 or bracketed timestamps at line start.
	timestampRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?|\[\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[^\]]*\])`)

	// Regex for detecting common HTTP methods with word boundaries.
	httpMethodRegex = regexp.MustCompile(`\b(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b`)

	// Regex for detecting standard log levels.
	errorLevelRegex = regexp.MustCompile(`(?i)\b(ERROR|FATAL|PANIC|CRITICAL|FAIL|FAILURE)\b`)
	warnLevelRegex  = regexp.MustCompile(`(?i)\b(WARN|WARNING)\b`)
	infoLevelRegex  = regexp.MustCompile(`(?i)\b(INFO|NOTICE)\b`)
	debugLevelRegex = regexp.MustCompile(`(?i)\b(DEBUG|TRACE)\b`)
)

// CompileLogMatcher returns a matcher function and compiled regexp for searching logs.
// If query is valid regex, it uses case-insensitive regex matching.
// If query contains invalid regex syntax, it gracefully falls back to case-insensitive literal substring.
func CompileLogMatcher(query string) (func(string) bool, *regexp.Regexp, bool) {
	q := strings.TrimSpace(query)
	if q == "" {
		return func(string) bool { return true }, nil, false
	}

	// Try compiling as regex
	if re, err := regexp.Compile("(?i)" + q); err == nil {
		return func(line string) bool {
			return re.MatchString(line)
		}, re, true
	}

	// Fallback to literal case-insensitive substring match
	lowerQ := strings.ToLower(q)
	return func(line string) bool {
		return strings.Contains(strings.ToLower(line), lowerQ)
	}, nil, false
}

// HighlightLogLine applies syntax highlighting (timestamps, log levels, HTTP methods)
// and overlays vibrant search match highlighting.
func HighlightLogLine(line string, re *regexp.Regexp, rawQuery string) string {
	if line == "" {
		return ""
	}

	// 1. If there's an active search query, find all match ranges first
	var matchSpans [][]int
	if re != nil {
		matchSpans = re.FindAllStringIndex(line, -1)
	} else if strings.TrimSpace(rawQuery) != "" {
		q := strings.ToLower(strings.TrimSpace(rawQuery))
		lowerLine := strings.ToLower(line)
		start := 0
		for {
			idx := strings.Index(lowerLine[start:], q)
			if idx == -1 {
				break
			}
			matchSpans = append(matchSpans, []int{start + idx, start + idx + len(q)})
			start += idx + len(q)
		}
	}

	// If there are search matches, highlight them
	if len(matchSpans) > 0 {
		return renderWithHighlightSpans(line, matchSpans)
	}

	// 2. Syntax highlighting for standard log tokens when not overshadowed by search
	return applySyntaxHighlighting(line)
}

func renderWithHighlightSpans(line string, spans [][]int) string {
	var sb strings.Builder
	lastIdx := 0

	for _, span := range spans {
		start, end := span[0], span[1]
		if start < lastIdx {
			start = lastIdx
		}
		if start >= len(line) || end <= start {
			continue
		}
		if start > lastIdx {
			sb.WriteString(applySyntaxHighlighting(line[lastIdx:start]))
		}
		sb.WriteString(SearchHighlightStyle.Render(line[start:end]))
		lastIdx = end
	}

	if lastIdx < len(line) {
		sb.WriteString(applySyntaxHighlighting(line[lastIdx:]))
	}

	return sb.String()
}

func applySyntaxHighlighting(segment string) string {
	if segment == "" {
		return ""
	}

	// Check if this segment contains a timestamp at the start
	res := segment
	if loc := timestampRegex.FindStringIndex(res); loc != nil && loc[0] == 0 {
		ts := res[loc[0]:loc[1]]
		rest := res[loc[1]:]
		res = TimestampStyle.Render(ts) + rest
	}

	// Highlight log levels
	res = errorLevelRegex.ReplaceAllStringFunc(res, func(m string) string {
		return theme.RedStyle.Render(m)
	})
	res = warnLevelRegex.ReplaceAllStringFunc(res, func(m string) string {
		return theme.YellowStyle.Render(m)
	})
	res = infoLevelRegex.ReplaceAllStringFunc(res, func(m string) string {
		return theme.GreenStyle.Render(m)
	})
	res = debugLevelRegex.ReplaceAllStringFunc(res, func(m string) string {
		return theme.DimStyle.Render(m)
	})

	// Highlight HTTP methods
	res = httpMethodRegex.ReplaceAllStringFunc(res, func(m string) string {
		return HTTPMethodStyle.Render(m)
	})

	return res
}
