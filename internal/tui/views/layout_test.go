package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/testutil"
)

func TestRenderHeaderAndTabs(t *testing.T) {
	// Header
	hdr := RenderHeader(HeaderContext{
		Version:               "v0.6.3",
		LastUpdate:            "12:00:00",
		DockerInfo:            monitor.DockerSummary{RunningContainers: 2, StoppedContainers: 1, ImagesCount: 7, Version: "27.4.0"},
		FormattedDockerStatus: "Running",
	})
	testutil.AssertContainsAll(t, hdr,
		"Dockpulse",
		"v0.6.3",
		"12:00:00",
		"Containers: 2 active, 1 stopped",
		"27.4.0",
	)

	// Tabs
	tabs := []string{"1: Containers", "2: Images", "3: Compose", "4: Networks", "5: Volumes", "6: Swarm"}
	tabBar := RenderTabs(tabs, 1, 80)
	testutil.AssertContainsInOrder(t, tabBar, "[1: Containers]", "2: Images")
}

func TestRenderFooter(t *testing.T) {
	// Tab 1 footer (compact)
	f1 := RenderFooter(FooterContext{ActiveTab: 1, Width: 80})
	testutil.AssertContainsAll(t, f1, "Kill", "Remove", "Web", "Copy", "Refresh", "Quit", "Restart", "Info")

	// Tab 1 footer (wide)
	f1Wide := RenderFooter(FooterContext{ActiveTab: 1, Width: 180})
	testutil.AssertContainsAll(t, f1Wide, "Kill", "Remove", "Web", "Copy", "Refresh", "Quit", "Search", "Sort")

	// Tab 1 footer filtering
	f1Filter := RenderFooter(FooterContext{ActiveTab: 1, IsFiltering: true, Width: 80})
	if !strings.Contains(f1Filter, "Apply Filter") {
		t.Errorf("expected apply filter hint in footer, got: %s", f1Filter)
	}

	// Tab 2 (Images) footer
	f2 := RenderFooter(FooterContext{ActiveTab: 2, Width: 80})
	if !strings.Contains(f2, "Search") || !strings.Contains(f2, "Remove") || !strings.Contains(f2, "Filter Unused") {
		t.Errorf("expected Search, Remove, and FilterUnused hints in footer 2, got: %s", f2)
	}

	// Modal footers
	fDown := RenderFooter(FooterContext{ActiveModal: "compose_down", Width: 80})
	if !strings.Contains(fDown, "(Down)") {
		t.Errorf("expected compose down modal footer, got: %s", fDown)
	}

	fRemove := RenderFooter(FooterContext{ActiveModal: "remove_container", Width: 80})
	if !strings.Contains(fRemove, "(Remove)") {
		t.Errorf("expected remove container modal footer, got: %s", fRemove)
	}

	fRemoveImg := RenderFooter(FooterContext{ActiveModal: "remove_image", Width: 80})
	if !strings.Contains(fRemoveImg, "(Remove Image)") {
		t.Errorf("expected remove image modal footer, got: %s", fRemoveImg)
	}

	// Exec modal footer
	fExec := RenderFooter(FooterContext{ActiveModal: "exec", Width: 80})
	if !strings.Contains(fExec, "Select Action") {
		t.Errorf("expected select action in exec footer, got: %s", fExec)
	}

	// Prune modal footers
	fPrune := RenderFooter(FooterContext{ActiveModal: "prune", PruneConfirm: 0, Width: 80})
	if !strings.Contains(fPrune, "Select Action") {
		t.Errorf("expected select action in prune footer, got: %s", fPrune)
	}
	fPruneConf := RenderFooter(FooterContext{ActiveModal: "prune", PruneConfirm: 1, Width: 80})
	if !strings.Contains(fPruneConf, "Confirm Prune") {
		t.Errorf("expected confirm prune in footer, got: %s", fPruneConf)
	}

	// Shell active footers
	fShellUnf := RenderFooter(FooterContext{ShellActive: true, ShellFocused: false, Width: 80})
	if !strings.Contains(fShellUnf, "Focus terminal") {
		t.Errorf("expected shell focus hint, got: %s", fShellUnf)
	}
	fShellFoc := RenderFooter(FooterContext{ShellActive: true, ShellFocused: true, Width: 80})
	if !strings.Contains(fShellFoc, "Return to Dockpulse") {
		t.Errorf("expected release focus hint, got: %s", fShellFoc)
	}

	// Other tab footers (3, 4, 6, default)
	f3 := RenderFooter(FooterContext{ActiveTab: 3, Width: 80})
	if !strings.Contains(f3, "Select Stack") {
		t.Errorf("expected select stack in tab 3, got: %s", f3)
	}
	f4 := RenderFooter(FooterContext{ActiveTab: 4, Width: 80})
	if !strings.Contains(f4, "Refresh") {
		t.Errorf("expected refresh in tab 4, got: %s", f4)
	}
	f6 := RenderFooter(FooterContext{ActiveTab: 6, Width: 80})
	if !strings.Contains(f6, "Service Logs") {
		t.Errorf("expected service logs in tab 6, got: %s", f6)
	}
	fDef := RenderFooter(FooterContext{ActiveTab: 99, Width: 80})
	if !strings.Contains(fDef, "Tabs") {
		t.Errorf("expected tabs hint in default footer, got: %s", fDef)
	}
}
