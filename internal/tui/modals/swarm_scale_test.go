package modals

import (
	"strings"
	"testing"
)

func TestRenderSwarmScale(t *testing.T) {
	// Equal replicas (no change)
	out := RenderSwarmScale("web-svc", "123456789abc", 3, 3)
	if !strings.Contains(out, "SCALE DOCKER SWARM SERVICE") {
		t.Errorf("expected title in modal, got %s", out)
	}
	if !strings.Contains(out, "web-svc") || !strings.Contains(out, "123456789abc") {
		t.Errorf("expected service name and ID, got %s", out)
	}
	if !strings.Contains(out, "no change") {
		t.Errorf("expected 'no change' diff indicator, got %s", out)
	}

	// Positive diff (+2)
	outUp := RenderSwarmScale("api-svc", "api-svc", 2, 4)
	if !strings.Contains(outUp, "+2") {
		t.Errorf("expected '+2' diff indicator, got %s", outUp)
	}

	// Negative diff (-1)
	outDown := RenderSwarmScale("worker-svc", "", 5, 4)
	if !strings.Contains(outDown, "-1") {
		t.Errorf("expected '-1' diff indicator, got %s", outDown)
	}
}
