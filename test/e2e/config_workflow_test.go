package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configPkg "github.com/AndreyJVM/container-ctl/internal/config"
	"github.com/AndreyJVM/container-ctl/internal/module"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

func TestE2E_ConfigConstructor_CustomOrderingAndLabels(t *testing.T) {
	t.Parallel()

	yamlContent := `
interval: 1s
initial_tab: 4
show_swarm: false
ui:
  only_unused_images: true
  default_sort: "cpu"
modules:
  networks:
    enabled: true
    title: "Virtual Networks"
    order: 1
  containers:
    enabled: true
    title: "Active Pods"
    order: 2
  images:
    enabled: false
  compose:
    enabled: true
    title: "Services"
    order: 3
  volumes:
    enabled: true
    title: "Storage"
    order: 4
`
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom_config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write custom config: %v", err)
	}

	appCfg, err := configPkg.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	reg := module.DefaultRegistry()
	reg.ApplyConfigs(appCfg.Modules)

	sortMode, _ := configPkg.ParseSortMode(appCfg.UI.DefaultSort)

	m := tui.NewModelWithOptions(
		appCfg.Interval,
		appCfg.InitialTab,
		"v0.22.6",
		tui.WithRegistry(reg),
		tui.WithShowSwarm(appCfg.ShowSwarm),
		tui.WithOnlyUnusedImages(appCfg.UI.OnlyUnusedImages),
		tui.WithSortMode(sortMode),
	)

	// Feed initial test data
	data := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Status:            "running",
			StatusDisplay:     "Running",
			RunningContainers: 1,
			Containers: []monitor.ContainerItem{
				{ID: "c1", Name: "pod-1", State: "running", Status: "Up 1h"},
			},
		},
		Networks: []monitor.DockerNetwork{
			{ID: "net1", Name: "custom-net", Driver: "bridge"},
		},
		Volumes: []monitor.DockerVolume{
			{Name: "data-vol", Driver: "local"},
		},
		Compose: []monitor.ComposeProject{
			{Name: "stack-1", Status: "running", StatusDisplay: "Running"},
		},
		Timestamp: "12:00:00",
	}

	updated, _ := m.Update(data)
	m = updated.(tui.Model)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = updated.(tui.Model)

	// 1. Initial tab should be Networks (TabNetworks)
	if m.ActiveTab() != tui.TabNetworks {
		t.Fatalf("expected initial tab to be TabNetworks (4), got: %s", m.ActiveTab())
	}

	// 2. View should contain custom tab titles
	view := m.View()
	if !strings.Contains(view, "Virtual Networks") {
		t.Errorf("expected view to contain custom title 'Virtual Networks', got: %s", view)
	}
	if !strings.Contains(view, "Active Pods") {
		t.Errorf("expected view to contain custom title 'Active Pods', got: %s", view)
	}
	if !strings.Contains(view, "Services") {
		t.Errorf("expected view to contain custom title 'Services', got: %s", view)
	}
	if !strings.Contains(view, "Storage") {
		t.Errorf("expected view to contain custom title 'Storage', got: %s", view)
	}

	// 3. Disabled module "images" should not be visible in tab list
	if strings.Contains(view, "Images") && !strings.Contains(view, "Images:") {
		// Note: header might contain "Images: 0", but tab title should not be in tab bar
		tabLine := strings.Split(view, "\n")[2]
		if strings.Contains(tabLine, "Images") {
			t.Errorf("expected Images module to be disabled and absent from tab bar: %s", tabLine)
		}
	}

	// 4. Tab cycling: press Tab to cycle through visible modules in configured order
	// Order: Networks (1) -> Containers (2) -> Compose (3) -> Volumes (4)
	m, _ = sendKey(m, "tab")
	if m.ActiveTab() != tui.TabDocker {
		t.Errorf("expected next tab to be TabDocker (Active Pods), got: %s", m.ActiveTab())
	}

	m, _ = sendKey(m, "tab")
	if m.ActiveTab() != tui.TabCompose {
		t.Errorf("expected next tab to be TabCompose (Services), got: %s", m.ActiveTab())
	}

	m, _ = sendKey(m, "tab")
	if m.ActiveTab() != tui.TabVolumes {
		t.Errorf("expected next tab to be TabVolumes (Storage), got: %s", m.ActiveTab())
	}

	// Cycling past the end wraps back to first visible module (Networks)
	m, _ = sendKey(m, "tab")
	if m.ActiveTab() != tui.TabNetworks {
		t.Errorf("expected tab wrap-around back to TabNetworks, got: %s", m.ActiveTab())
	}

	// 5. Direct number navigation
	// Pressing '1' navigates to the 1st visible tab (Virtual Networks)
	m, _ = sendKey(m, "1")
	if m.ActiveTab() != tui.TabNetworks {
		t.Errorf("expected key '1' to activate 1st configured module (Networks), got: %s", m.ActiveTab())
	}

	// Pressing '2' navigates to the 2nd visible tab (Active Pods / TabDocker)
	m, _ = sendKey(m, "2")
	if m.ActiveTab() != tui.TabDocker {
		t.Errorf("expected key '2' to activate 2nd configured module (Containers), got: %s", m.ActiveTab())
	}
}
