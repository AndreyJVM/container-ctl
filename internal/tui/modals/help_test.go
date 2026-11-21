package modals

import (
	"strings"
	"testing"
	"time"
)

func TestRenderHelp(t *testing.T) {
	view := RenderHelp(2 * time.Second)
	if !strings.Contains(view, "HELP & KEYBOARD SHORTCUTS") || !strings.Contains(view, "[K]") || !strings.Contains(view, "[x]") {
		t.Errorf("expected help title and new shortcut keys [K] and [x], got: %s", view)
	}
	if !strings.Contains(view, "NAVIGATION") || !strings.Contains(view, "CONTAINER ACTIONS") {
		t.Errorf("expected section titles in help view, got: %s", view)
	}
}
