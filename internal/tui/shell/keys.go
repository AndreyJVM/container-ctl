package shell

import tea "github.com/charmbracelet/bubbletea"

const (
	WideLayoutMin = 110
	MinWidth      = 20
	MinHeight     = 6
)

// KeyBytes converts a BubbleTea KeyMsg into the raw byte sequence expected by a PTY process.
func KeyBytes(msg tea.KeyMsg) []byte {
	if msg.Type == tea.KeyRunes {
		data := []byte(string(msg.Runes))
		if msg.Alt {
			return append([]byte{0x1b}, data...)
		}
		return data
	}

	if msg.Type == tea.KeySpace {
		data := []byte(" ")
		if msg.Alt {
			return append([]byte{0x1b}, data...)
		}
		return data
	}

	if msg.Type >= 0 && msg.Type <= 31 {
		data := []byte{byte(msg.Type)} // #nosec G115 -- bounded by msg.Type >= 0 && msg.Type <= 31
		if msg.Alt {
			return append([]byte{0x1b}, data...)
		}
		return data
	}
	if msg.Type == 127 {
		data := []byte{127}
		if msg.Alt {
			return append([]byte{0x1b}, data...)
		}
		return data
	}

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
	sequence := sequences[msg.Type]
	if sequence == "" {
		return nil
	}
	if msg.Alt {
		sequence = "\x1b" + sequence
	}
	return []byte(sequence)
}

// PanelSize calculates terminal dimensions given the overall window dimensions.
func PanelSize(width, height int) (int, int) {
	panelWidth := width
	if width >= WideLayoutMin {
		panelWidth = width / 2
	}
	return max(MinWidth, panelWidth-4), max(MinHeight, height-12)
}
