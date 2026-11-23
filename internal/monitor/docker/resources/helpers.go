package resources

import (
	"strconv"
	"strings"
)

// TruncateResourceID truncates a Docker resource ID to 12 chars.
func TruncateResourceID(id string) string {
	id = strings.TrimSpace(id)
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// ParseBool parses a string boolean safely.
func ParseBool(value string) bool {
	parsed, _ := strconv.ParseBool(strings.TrimSpace(value))
	return parsed
}

// NonEmptyLines filters out empty lines from a string.
func NonEmptyLines(value string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(value), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// LabelValue parses comma-separated key=value labels and returns the value for key.
func LabelValue(labels, key string) string {
	for _, label := range strings.Split(labels, ",") {
		name, value, found := strings.Cut(strings.TrimSpace(label), "=")
		if found && name == key {
			return value
		}
	}
	return ""
}
