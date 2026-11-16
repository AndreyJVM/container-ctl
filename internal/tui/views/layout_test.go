package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/dockpulse/internal/i18n"
	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestRenderHeaderAndTabs(t *testing.T) {
	dict := i18n.Get()

	// Header
	hdr := RenderHeader(HeaderContext{
		Version:               "v0.6.3",
		LastUpdate:            "12:00:00",
		DockerInfo:            monitor.DockerSummary{RunningContainers: 2, StoppedContainers: 1, ImagesCount: 7, Version: "27.4.0"},
		FormattedDockerStatus: "Running",
		I18n:                  &dict,
	})
	if !strings.Contains(hdr, "Dockpulse") || !strings.Contains(hdr, "v0.6.3") || !strings.Contains(hdr, "12:00:00") {
		t.Errorf("unexpected header output: %s", hdr)
	}
	if !strings.Contains(hdr, "Containers: 2 active, 1 stopped") || !strings.Contains(hdr, "27.4.0") {
		t.Errorf("expected persistent Docker summary, got: %s", hdr)
	}

	// Tabs
	tabs := []string{dict.TabDocker, dict.TabImages, dict.TabCompose, dict.TabNetworks, dict.TabVolumes, dict.TabSwarm}
	tabBar := RenderTabs(tabs, 1, 80)
	if !strings.Contains(tabBar, "[1: Containers]") {
		t.Errorf("expected active tab bracketed, got: %s", tabBar)
	}
	if !strings.Contains(tabBar, "2: Images") {
		t.Errorf("expected 2: Images tab in tab bar, got: %s", tabBar)
	}
}

func TestRenderFooter(t *testing.T) {
	dict := i18n.Get()

	// Tab 1 footer
	f1 := RenderFooter(FooterContext{ActiveTab: 1, I18n: &dict, Width: 80})
	if !strings.Contains(f1, dict.FooterKill) || !strings.Contains(f1, dict.FooterRemove) {
		t.Errorf("expected Kill and Remove hints in footer 1, got: %s", f1)
	}

	// Tab 1 footer filtering
	f1Filter := RenderFooter(FooterContext{ActiveTab: 1, IsFiltering: true, I18n: &dict, Width: 80})
	if !strings.Contains(f1Filter, dict.FooterApplyFilter) {
		t.Errorf("expected apply filter hint in footer, got: %s", f1Filter)
	}

	// Tab 2 (Images) footer
	f2 := RenderFooter(FooterContext{ActiveTab: 2, I18n: &dict, Width: 80})
	if !strings.Contains(f2, dict.FooterSearch) || !strings.Contains(f2, dict.FooterRemove) {
		t.Errorf("expected Search and Remove hints in footer 2, got: %s", f2)
	}

	// Modal footers
	fDown := RenderFooter(FooterContext{ActiveModal: "compose_down", I18n: &dict, Width: 80})
	if !strings.Contains(fDown, "(Down)") {
		t.Errorf("expected compose down modal footer, got: %s", fDown)
	}

	fRemove := RenderFooter(FooterContext{ActiveModal: "remove_container", I18n: &dict, Width: 80})
	if !strings.Contains(fRemove, "(Remove)") {
		t.Errorf("expected remove container modal footer, got: %s", fRemove)
	}

	fRemoveImg := RenderFooter(FooterContext{ActiveModal: "remove_image", I18n: &dict, Width: 80})
	if !strings.Contains(fRemoveImg, "(Remove Image)") {
		t.Errorf("expected remove image modal footer, got: %s", fRemoveImg)
	}

	// Exec modal footer
	fExec := RenderFooter(FooterContext{ActiveModal: "exec", I18n: &dict, Width: 80})
	if !strings.Contains(fExec, dict.FooterSelectAction) {
		t.Errorf("expected select action in exec footer, got: %s", fExec)
	}

	// Prune modal footers
	fPrune := RenderFooter(FooterContext{ActiveModal: "prune", PruneConfirm: 0, I18n: &dict, Width: 80})
	if !strings.Contains(fPrune, dict.FooterSelectAction) {
		t.Errorf("expected select action in prune footer, got: %s", fPrune)
	}
	fPruneConf := RenderFooter(FooterContext{ActiveModal: "prune", PruneConfirm: 1, I18n: &dict, Width: 80})
	if !strings.Contains(fPruneConf, dict.FooterConfirmPrune) {
		t.Errorf("expected confirm prune in footer, got: %s", fPruneConf)
	}

	// Shell active footers
	fShellUnf := RenderFooter(FooterContext{ShellActive: true, ShellFocused: false, I18n: &dict, Width: 80})
	if !strings.Contains(fShellUnf, dict.FooterShellFocus) {
		t.Errorf("expected shell focus hint, got: %s", fShellUnf)
	}
	fShellFoc := RenderFooter(FooterContext{ShellActive: true, ShellFocused: true, I18n: &dict, Width: 80})
	if !strings.Contains(fShellFoc, dict.FooterShellReleaseFocus) {
		t.Errorf("expected release focus hint, got: %s", fShellFoc)
	}

	// Other tab footers (3, 4, 6, default)
	f3 := RenderFooter(FooterContext{ActiveTab: 3, I18n: &dict, Width: 80})
	if !strings.Contains(f3, dict.FooterSelectStack) {
		t.Errorf("expected select stack in tab 3, got: %s", f3)
	}
	f4 := RenderFooter(FooterContext{ActiveTab: 4, I18n: &dict, Width: 80})
	if !strings.Contains(f4, dict.FooterRefresh) {
		t.Errorf("expected refresh in tab 4, got: %s", f4)
	}
	f6 := RenderFooter(FooterContext{ActiveTab: 6, I18n: &dict, Width: 80})
	if !strings.Contains(f6, dict.FooterServiceLogs) {
		t.Errorf("expected service logs in tab 6, got: %s", f6)
	}
	fDef := RenderFooter(FooterContext{ActiveTab: 99, I18n: &dict, Width: 80})
	if !strings.Contains(fDef, dict.FooterTabs) {
		t.Errorf("expected tabs hint in default footer, got: %s", fDef)
	}
}
