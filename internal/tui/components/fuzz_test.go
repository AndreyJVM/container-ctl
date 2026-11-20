package components

import (
	"testing"
)

func FuzzTruncate(f *testing.F) {
	f.Add("hello world", 5)
	f.Add("short", 10)
	f.Add("", 0)
	f.Add("emoji 🚀 test", 3)
	f.Add("unicode Привет", 4)
	f.Add("test", -1)

	f.Fuzz(func(t *testing.T, s string, maxLen int) {
		res := Truncate(s, maxLen)
		if maxLen <= 0 && res != "" {
			t.Fatalf("expected empty string for maxLen <= 0, got %q", res)
		}
	})
}

func FuzzPadRightAndLeft(f *testing.F) {
	f.Add("hello", 10)
	f.Add("already long enough", 5)
	f.Add("", 0)
	f.Add("🚀", 4)
	f.Add("test", -5)

	f.Fuzz(func(t *testing.T, s string, width int) {
		right := PadRight(s, width)
		left := PadLeft(s, width)
		if width <= 0 {
			if right != "" || left != "" {
				t.Fatalf("expected empty for width <= 0, got right=%q left=%q", right, left)
			}
		}
	})
}
