package components

import (
	"fmt"
)

// CalculateTableWindow computes the [start, end) index bounds for virtual scrolling in a list or table.
// It guarantees that:
// 1. If totalItems <= maxVisible or maxVisible <= 0, the full range [0, totalItems) is returned.
// 2. cursorIdx is always within [start, end).
// 3. Context rows are preserved above and below the cursor whenever possible.
func CalculateTableWindow(totalItems, cursorIdx, maxVisible int) (int, int) {
	if totalItems <= 0 || maxVisible <= 0 {
		return 0, 0
	}
	if totalItems <= maxVisible {
		return 0, totalItems
	}

	if cursorIdx < 0 {
		cursorIdx = 0
	} else if cursorIdx >= totalItems {
		cursorIdx = totalItems - 1
	}

	// Try to center the cursor in the visible window
	half := maxVisible / 2
	start := cursorIdx - half
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > totalItems {
		end = totalItems
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	return start, end
}

// FormatWindowBadge returns a string like "[1-15 of 54]" when windowing is active,
// or an empty string if all items are currently visible.
func FormatWindowBadge(start, end, total int) string {
	if total <= 0 || (start == 0 && end >= total) {
		return ""
	}
	return fmt.Sprintf("[%d-%d of %d]", start+1, end, total)
}
