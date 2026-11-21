package shell

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyBytes_Runes(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	b := KeyBytes(msg)
	if !bytes.Equal(b, []byte("a")) {
		t.Errorf("expected 'a', got %v", b)
	}

	msgAlt := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a"), Alt: true}
	bAlt := KeyBytes(msgAlt)
	if !bytes.Equal(bAlt, []byte{0x1b, 'a'}) {
		t.Errorf("expected ESC+'a', got %v", bAlt)
	}
}

func TestKeyBytes_ControlAndSpecial(t *testing.T) {
	// Ctrl+C (Type 3)
	msgCtrlC := tea.KeyMsg{Type: tea.KeyCtrlC}
	bCtrlC := KeyBytes(msgCtrlC)
	if len(bCtrlC) != 1 || bCtrlC[0] != 3 {
		t.Errorf("expected byte 3 for Ctrl+C, got %v", bCtrlC)
	}

	// Backspace / DEL (127)
	msgDel := tea.KeyMsg{Type: 127}
	bDel := KeyBytes(msgDel)
	if len(bDel) != 1 || bDel[0] != 127 {
		t.Errorf("expected byte 127, got %v", bDel)
	}

	// Escape sequences: Up, Down, Left, Right, etc.
	sequences := map[tea.KeyType]string{
		tea.KeyUp:       "\x1b[A",
		tea.KeyDown:     "\x1b[B",
		tea.KeyRight:    "\x1b[C",
		tea.KeyLeft:     "\x1b[D",
		tea.KeyHome:     "\x1b[H",
		tea.KeyEnd:      "\x1b[F",
		tea.KeyPgUp:     "\x1b[5~",
		tea.KeyPgDown:   "\x1b[6~",
		tea.KeyDelete:   "\x1b[3~",
		tea.KeyInsert:   "\x1b[2~",
		tea.KeyShiftTab: "\x1b[Z",
	}

	for kType, expected := range sequences {
		b := KeyBytes(tea.KeyMsg{Type: kType})
		if string(b) != expected {
			t.Errorf("for key %v: expected %q, got %q", kType, expected, string(b))
		}
		// With Alt
		bWithAlt := KeyBytes(tea.KeyMsg{Type: kType, Alt: true})
		if string(bWithAlt) != "\x1b"+expected {
			t.Errorf("for key %v with Alt: expected %q, got %q", kType, "\x1b"+expected, string(bWithAlt))
		}
	}

	// Unknown key
	if b := KeyBytes(tea.KeyMsg{Type: tea.KeyF1}); b != nil {
		t.Errorf("expected nil for unmapped key, got %v", b)
	}
}

func TestPanelSize(t *testing.T) {
	// Narrow terminal (< 110)
	w, h := PanelSize(100, 30)
	if w != 96 || h != 18 {
		t.Errorf("expected 96, 18 for 100x30, got %d, %d", w, h)
	}

	// Wide terminal (>= 110)
	w, h = PanelSize(120, 40)
	if w != 56 || h != 28 {
		t.Errorf("expected 56, 28 for 120x40, got %d, %d", w, h)
	}

	// Very small terminal
	w, h = PanelSize(10, 5)
	if w != MinWidth || h != MinHeight {
		t.Errorf("expected min bounds (%d, %d), got (%d, %d)", MinWidth, MinHeight, w, h)
	}
}
