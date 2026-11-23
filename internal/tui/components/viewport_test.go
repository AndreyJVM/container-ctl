package components

import (
	"testing"
)

func TestCalculateTableWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		total      int
		cursor     int
		maxVisible int
		wantStart  int
		wantEnd    int
	}{
		{"empty list", 0, 0, 10, 0, 0},
		{"negative total", -5, 0, 10, 0, 0},
		{"zero maxVisible", 10, 2, 0, 0, 0},
		{"negative maxVisible", 10, 2, -1, 0, 0},
		{"fits entirely", 5, 2, 10, 0, 5},
		{"exact match", 10, 5, 10, 0, 10},
		{"cursor at start", 50, 0, 10, 0, 10},
		{"cursor near start", 50, 3, 10, 0, 10},
		{"cursor in middle", 50, 20, 10, 15, 25},
		{"cursor at end", 50, 49, 10, 40, 50},
		{"cursor near end", 50, 47, 10, 40, 50},
		{"negative cursor clamped", 50, -5, 10, 0, 10},
		{"excessive cursor clamped", 50, 999, 10, 40, 50},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotStart, gotEnd := CalculateTableWindow(tt.total, tt.cursor, tt.maxVisible)
			if gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("CalculateTableWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.total, tt.cursor, tt.maxVisible, gotStart, gotEnd, tt.wantStart, tt.wantEnd)
			}

			// Verify invariant: if valid window, cursor is in [start, end)
			if tt.total > 0 && tt.maxVisible > 0 {
				effectiveCursor := tt.cursor
				if effectiveCursor < 0 {
					effectiveCursor = 0
				} else if effectiveCursor >= tt.total {
					effectiveCursor = tt.total - 1
				}
				if effectiveCursor < gotStart || effectiveCursor >= gotEnd {
					t.Errorf("cursor %d not in window [%d, %d)", effectiveCursor, gotStart, gotEnd)
				}
			}
		})
	}
}

func TestFormatWindowBadge(t *testing.T) {
	t.Parallel()

	if b := FormatWindowBadge(0, 10, 10); b != "" {
		t.Errorf("expected empty string when all visible, got %q", b)
	}
	if b := FormatWindowBadge(0, 0, 0); b != "" {
		t.Errorf("expected empty string for zero total, got %q", b)
	}
	if b := FormatWindowBadge(0, 15, 50); b != "[1-15 of 50]" {
		t.Errorf("expected '[1-15 of 50]', got %q", b)
	}
	if b := FormatWindowBadge(10, 25, 50); b != "[11-25 of 50]" {
		t.Errorf("expected '[11-25 of 50]', got %q", b)
	}
}
