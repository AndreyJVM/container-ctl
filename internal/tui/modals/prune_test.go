package modals

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestRenderPrune(t *testing.T) {
	// Error view
	errView := RenderPrune(monitor.DockerDiskUsage{}, "daemon error", 0)
	if !strings.Contains(errView, "daemon error") {
		t.Errorf("expected error message in prune view, got: %s", errView)
	}

	// Menu view
	usage := monitor.DockerDiskUsage{
		Images:     monitor.DockerDiskItem{TotalCount: 5, ActiveCount: 2, TotalSize: 1024 * 1024 * 500, Reclaimable: 1024 * 1024 * 200},
		Containers: monitor.DockerDiskItem{TotalCount: 3, ActiveCount: 1, TotalSize: 1024 * 1024 * 50, Reclaimable: 1024 * 1024 * 30},
		TotalSize:  1024 * 1024 * 550,
		TotalRecl:  1024 * 1024 * 230,
	}
	menuView := RenderPrune(usage, "", 0)
	if !strings.Contains(menuView, "CLEANUP & PRUNE DOCKER RESOURCES") || !strings.Contains(menuView, "[1]") {
		t.Errorf("expected prune menu items, got: %s", menuView)
	}

	// Confirmation step view
	confirmView := RenderPrune(usage, "", 1)
	if !strings.Contains(confirmView, "WARNING:") || !strings.Contains(confirmView, "[Y]") {
		t.Errorf("expected prune confirmation view, got: %s", confirmView)
	}
}

func TestGetPruneActionName(t *testing.T) {
	if act1 := getPruneActionName(1); act1 != "Prune Stopped Containers" {
		t.Errorf("expected Prune Stopped Containers, got %q", act1)
	}
	if act2 := getPruneActionName(2); act2 != "Prune Dangling & Unused Images" {
		t.Errorf("expected Prune Dangling & Unused Images, got %q", act2)
	}
	if act3 := getPruneActionName(3); act3 != "Prune Unused Volumes" {
		t.Errorf("expected Prune Unused Volumes, got %q", act3)
	}
	if act4 := getPruneActionName(4); act4 != "Prune System (All Unused Resources)" {
		t.Errorf("expected Prune System (All Unused Resources), got %q", act4)
	}
	if actOther := getPruneActionName(99); actOther != "" {
		t.Errorf("expected empty string for unknown step, got %q", actOther)
	}
}
