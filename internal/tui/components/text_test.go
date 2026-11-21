package components

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hello world", 8, "hello..."},
		{"ab😀cdef", 3, "ab😀"},
		{"a😀b", 2, "a😀"},
		{"", 5, ""},
		{"short", 0, ""},
		{"short", -1, ""},
		{"short", -10, ""},
	}

	for _, c := range cases {
		got := Truncate(c.input, c.maxLen)
		if got != c.expected {
			t.Errorf("Truncate(%q, %d) = %q; want %q", c.input, c.maxLen, got, c.expected)
		}
	}
}

func TestPadRight(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"shorter than width", "test", 8, "test    "},
		{"exact width", "test", 4, "test"},
		{"longer than width", "longer-text", 4, "long"},
		{"zero width", "test", 0, ""},
		{"negative width", "test", -5, ""},
		{"empty string", "", 3, "   "},
		{"unicode runes", "привет", 8, "привет  "},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PadRight(c.input, c.width)
			if got != c.expected {
				t.Errorf("PadRight(%q, %d) = %q; want %q", c.input, c.width, got, c.expected)
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"shorter than width", "test", 8, "    test"},
		{"exact width", "test", 4, "test"},
		{"longer than width", "longer-text", 4, "long"},
		{"zero width", "test", 0, ""},
		{"negative width", "test", -5, ""},
		{"empty string", "", 3, "   "},
		{"unicode runes", "мир", 6, "   мир"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := PadLeft(c.input, c.width)
			if got != c.expected {
				t.Errorf("PadLeft(%q, %d) = %q; want %q", c.input, c.width, got, c.expected)
			}
		})
	}
}

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
