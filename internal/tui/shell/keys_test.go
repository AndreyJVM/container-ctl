package shell

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyBytes_Runes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      tea.KeyMsg
		expected []byte
	}{
		{
			name:     "single ASCII character without alt",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
			expected: []byte("a"),
		},
		{
			name:     "multiple ASCII characters without alt",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("echo")},
			expected: []byte("echo"),
		},
		{
			name:     "unicode multibyte rune without alt",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("привет")},
			expected: []byte("привет"),
		},
		{
			name:     "single ASCII character with alt (esc prefix)",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x"), Alt: true},
			expected: []byte{0x1b, 'x'},
		},
		{
			name:     "unicode character with alt",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ж"), Alt: true},
			expected: append([]byte{0x1b}, []byte("ж")...),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := KeyBytes(tc.msg)
			if !bytes.Equal(got, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestKeyBytes_ControlKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		msg      tea.KeyMsg
		expected []byte
	}{
		{
			name:     "Ctrl+A (code 1)",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlA},
			expected: []byte{1},
		},
		{
			name:     "Ctrl+C (code 3)",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlC},
			expected: []byte{3},
		},
		{
			name:     "Ctrl+D (code 4)",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlD},
			expected: []byte{4},
		},
		{
			name:     "Ctrl+Z (code 26)",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlZ},
			expected: []byte{26},
		},
		{
			name:     "Ctrl+] (code 29)",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlCloseBracket},
			expected: []byte{29},
		},
		{
			name:     "Ctrl+C with Alt (code 3 with esc prefix)",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlC, Alt: true},
			expected: []byte{0x1b, 3},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := KeyBytes(tc.msg)
			if !bytes.Equal(got, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestKeyBytes_DeleteAndBackspace(t *testing.T) {
	t.Parallel()

	t.Run("standard DEL 127 byte", func(t *testing.T) {
		t.Parallel()
		got := KeyBytes(tea.KeyMsg{Type: 127})
		expected := []byte{127}
		if !bytes.Equal(got, expected) {
			t.Errorf("expected %v, got %v", expected, got)
		}
	})

	t.Run("DEL 127 with Alt", func(t *testing.T) {
		t.Parallel()
		got := KeyBytes(tea.KeyMsg{Type: 127, Alt: true})
		expected := []byte{0x1b, 127}
		if !bytes.Equal(got, expected) {
			t.Errorf("expected %v, got %v", expected, got)
		}
	})
}

func TestKeyBytes_EscapeSequences(t *testing.T) {
	t.Parallel()

	sequences := []struct {
		name     string
		keyType  tea.KeyType
		expected string
	}{
		{"KeyUp", tea.KeyUp, "\x1b[A"},
		{"KeyDown", tea.KeyDown, "\x1b[B"},
		{"KeyRight", tea.KeyRight, "\x1b[C"},
		{"KeyLeft", tea.KeyLeft, "\x1b[D"},
		{"KeyHome", tea.KeyHome, "\x1b[H"},
		{"KeyEnd", tea.KeyEnd, "\x1b[F"},
		{"KeyPgUp", tea.KeyPgUp, "\x1b[5~"},
		{"KeyPgDown", tea.KeyPgDown, "\x1b[6~"},
		{"KeyDelete", tea.KeyDelete, "\x1b[3~"},
		{"KeyInsert", tea.KeyInsert, "\x1b[2~"},
		{"KeyShiftTab", tea.KeyShiftTab, "\x1b[Z"},
	}

	for _, tc := range sequences {
		tc := tc
		t.Run(tc.name+" standard", func(t *testing.T) {
			t.Parallel()
			got := KeyBytes(tea.KeyMsg{Type: tc.keyType})
			if string(got) != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, string(got))
			}
		})

		t.Run(tc.name+" with Alt", func(t *testing.T) {
			t.Parallel()
			got := KeyBytes(tea.KeyMsg{Type: tc.keyType, Alt: true})
			expectedWithAlt := "\x1b" + tc.expected
			if string(got) != expectedWithAlt {
				t.Errorf("expected %q, got %q", expectedWithAlt, string(got))
			}
		})
	}
}

func TestKeyBytes_UnmappedKeys(t *testing.T) {
	t.Parallel()

	unmapped := []tea.KeyType{tea.KeyF1, tea.KeyF2, tea.KeyF5, tea.KeyF12}
	for _, k := range unmapped {
		if got := KeyBytes(tea.KeyMsg{Type: k}); got != nil {
			t.Errorf("expected nil for unmapped key %v, got %v", k, got)
		}
	}
}

func TestPanelSize_Calculations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		width      int
		height     int
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "narrow layout width < 110",
			width:      100,
			height:     30,
			wantWidth:  96, // 100 - 4
			wantHeight: 18, // 30 - 12
		},
		{
			name:       "exact wide threshold width == 110",
			width:      110,
			height:     30,
			wantWidth:  51, // (110 / 2) - 4 = 55 - 4 = 51
			wantHeight: 18, // 30 - 12
		},
		{
			name:       "wide layout width > 110",
			width:      160,
			height:     40,
			wantWidth:  76, // (160 / 2) - 4 = 80 - 4 = 76
			wantHeight: 28, // 40 - 12
		},
		{
			name:       "clamping to MinWidth and MinHeight on tiny dimensions",
			width:      15,
			height:     10,
			wantWidth:  MinWidth,  // 20
			wantHeight: MinHeight, // 6
		},
		{
			name:       "clamping to MinWidth and MinHeight on zero and negative dimensions",
			width:      0,
			height:     -5,
			wantWidth:  MinWidth,  // 20
			wantHeight: MinHeight, // 6
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotW, gotH := PanelSize(tc.width, tc.height)
			if gotW != tc.wantWidth || gotH != tc.wantHeight {
				t.Errorf("PanelSize(%d, %d) = (%d, %d); want (%d, %d)",
					tc.width, tc.height, gotW, gotH, tc.wantWidth, tc.wantHeight)
			}
		})
	}
}
