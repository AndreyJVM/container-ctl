package modals

import (
	"strings"
	"testing"
	"time"
)

func TestRenderHelp(t *testing.T) {
	// Default (Swarm visible)
	view := RenderHelp(2 * time.Second)
	if !strings.Contains(view, "HELP & KEYBOARD SHORTCUTS") || !strings.Contains(view, "[K]") || !strings.Contains(view, "[x]") {
		t.Errorf("expected help title and shortcut keys [K] and [x], got: %s", view)
	}
	if !strings.Contains(view, "NAVIGATION") || !strings.Contains(view, "CONTAINER ACTIONS") {
		t.Errorf("expected section titles in help view, got: %s", view)
	}
	if !strings.Contains(view, "SWARM ACTIONS") || !strings.Contains(view, "Scale service replicas") {
		t.Errorf("expected SWARM ACTIONS section in help view when swarm is visible, got: %s", view)
	}

	// Swarm hidden
	viewNoSwarm := RenderHelp(2*time.Second, false)
	if strings.Contains(viewNoSwarm, "SWARM ACTIONS") {
		t.Errorf("expected SWARM ACTIONS absent when swarm is hidden, got: %s", viewNoSwarm)
	}
}
