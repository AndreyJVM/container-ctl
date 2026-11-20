package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
	"github.com/AndreyJVM/dockpulse/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInspectModalHealthBadge(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.4.1")
	m.activeModal = "inspect"
	m.inspectTarget = monitor.ContainerDetail{
		ID:     "c1",
		Name:   "app-prod",
		Image:  "my-app:v1",
		Status: "Up 2 hours",
		Health: "healthy",
	}

	view := m.View()
	if !strings.Contains(view, "Health:") || !strings.Contains(view, "[healthy]") {
		t.Errorf("expected health badge in inspect modal view, got: %s", view)
	}

	// Test unhealthy
	m.inspectTarget.Health = "unhealthy"
	view = m.View()
	if !strings.Contains(view, "[unhealthy]") {
		t.Errorf("expected [unhealthy] badge in inspect modal view, got: %s", view)
	}

	// Test starting
	m.inspectTarget.Health = "starting"
	view = m.View()
	if !strings.Contains(view, "[starting]") {
		t.Errorf("expected [starting] badge in inspect modal view, got: %s", view)
	}

	// Test no health configured
	m.inspectTarget.Health = ""
	view = m.View()
	if !strings.Contains(view, "Health:") {
		t.Errorf("expected Health line even if health is empty, got: %s", view)
	}
}

func TestPruneModalAndWorkflow(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.5.0")
	m.width = 100
	m.height = 30

	// 1. Open prune modal on tab 2 with "c"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Fatalf("expected activeModal to be 'prune', got %q", m.activeModal)
	}

	// 2. View in menu state
	view := m.View()
	if !strings.Contains(view, "CLEANUP & PRUNE DOCKER RESOURCES") {
		t.Errorf("expected prune modal header in view, got: %s", view)
	}
	if !strings.Contains(view, "Prune Stopped Containers") {
		t.Errorf("expected menu item 1 in view, got: %s", view)
	}
	if !strings.Contains(view, "[1-4]") {
		t.Errorf("expected [1-4] footer hint, got: %s", view)
	}

	// 3. Select option 1 (prune stopped containers)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = res.(Model)
	if m.pruneConfirm != 1 {
		t.Errorf("expected pruneConfirm to be 1, got %d", m.pruneConfirm)
	}
	confirmView := m.View()
	if !strings.Contains(confirmView, "WARNING: Prune Stopped Containers") {
		t.Errorf("expected confirmation prompt in view, got: %s", confirmView)
	}
	if !strings.Contains(confirmView, "[Y]") {
		t.Errorf("expected [Y] confirmation hint, got: %s", confirmView)
	}

	// 4. Cancel confirmation with "n"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = res.(Model)
	if m.pruneConfirm != 0 {
		t.Errorf("expected pruneConfirm to be 0 after 'n', got %d", m.pruneConfirm)
	}
	if m.activeModal != "prune" {
		t.Errorf("expected activeModal to remain 'prune', got %q", m.activeModal)
	}

	// 5. Select option 4 and confirm with "Y"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = res.(Model)
	if m.pruneConfirm != 4 {
		t.Errorf("expected pruneConfirm to be 4, got %d", m.pruneConfirm)
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal to close after confirmation, got %q", m.activeModal)
	}
	if m.pruneConfirm != 0 {
		t.Errorf("expected pruneConfirm to reset to 0, got %d", m.pruneConfirm)
	}

	// 6. Test opening prune modal on Tab 2 (Images) and Tab 3 (Compose)
	m.activeTab = TabImages
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Errorf("expected activeModal 'prune' on tab 2 with 'C', got %q", m.activeModal)
	}

	// 7. Close modal with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal to close on Esc, got %q", m.activeModal)
	}

	// 8. Check footer hints on Docker, Images, and Compose tabs.
	m.activeTab = TabDocker
	m.activeModal = ""
	dockerView := m.View()
	if !strings.Contains(dockerView, "[C]") || !strings.Contains(dockerView, "Prune") {
		t.Errorf("expected [C] Prune in Docker footer, got: %s", dockerView)
	}

	m.activeTab = TabImages
	imagesView := m.View()
	if !strings.Contains(imagesView, "[C]") || !strings.Contains(imagesView, "Prune") {
		t.Errorf("expected [C] Prune in Images footer, got: %s", imagesView)
	}

	m.activeTab = TabCompose
	composeView := m.View()
	if !strings.Contains(composeView, "[C]") || !strings.Contains(composeView, "Prune") {
		t.Errorf("expected [C] Prune in Compose footer, got: %s", composeView)
	}

	// 9. Check help box hint
	m.showHelp = true
	helpView := m.View()
	if !strings.Contains(helpView, "[C]") || !strings.Contains(helpView, "Prune unused Docker resources") {
		t.Errorf("expected [C] hint in help view, got: %s", helpView)
	}
}

func TestExecModalWorkflow(t *testing.T) {
	restore := monitor.SetDockerCLIAvailableCheckerForTesting(func() bool { return true })
	defer restore()

	m := NewModel(2*time.Second, 1, "v0.7.4")
	m.dockerInfo = monitor.DockerSummary{
		Status: "running",
		Containers: []monitor.ContainerItem{
			testutil.AContainer(testutil.WithID("c123"), testutil.WithName("app-test"), testutil.WithState("running", "Up")),
		},
	}
	m.cursorIdx = 0

	// 1. Open exec modal with "e"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(Model)
	if m.activeModal != "exec" {
		t.Fatalf("expected activeModal to be 'exec', got %q", m.activeModal)
	}

	// 2. Navigate down and up in menu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.execCursorIdx != 1 {
		t.Errorf("expected execCursorIdx 1 after down, got %d", m.execCursorIdx)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.execCursorIdx != 0 {
		t.Errorf("expected execCursorIdx 0 after up, got %d", m.execCursorIdx)
	}

	// Wrap around up
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.execCursorIdx != 6 {
		t.Errorf("expected execCursorIdx 6 after wrap up, got %d", m.execCursorIdx)
	}

	// 3. Switch to custom command mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = res.(Model)
	if !m.execIsCustomInput {
		t.Errorf("expected execIsCustomInput to be true")
	}

	// Type custom command "python3"
	for _, r := range "python3" {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}
	if m.execCustomCmd != "python3" {
		t.Errorf("expected execCustomCmd 'python3', got %q", m.execCustomCmd)
	}

	// Backspace
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.execCustomCmd != "python" {
		t.Errorf("expected execCustomCmd 'python', got %q", m.execCustomCmd)
	}

	// Esc in custom command goes back to menu
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = res.(Model)
	if m.execIsCustomInput {
		t.Errorf("expected execIsCustomInput false after Esc")
	}
	if m.activeModal != "exec" {
		t.Errorf("expected activeModal still 'exec'")
	}

	// Direct shortcut "4" launches attach
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected modal to close on launching exec, got %q", m.activeModal)
	}
	if cmd == nil {
		t.Errorf("expected non-nil tea.Cmd for launching attach")
	}
}

func TestNetworksAndVolumesHotkeys(t *testing.T) {
	m := Model{
		version:         "v0.18.1",
		activeTab:       TabNetworks,
		refreshInterval: 2 * time.Second,
		networks: []monitor.DockerNetwork{
			testutil.ANetwork(func(n *monitor.DockerNetwork) {
				n.Name = "production-net"
			}),
		},
		volumes: []monitor.DockerVolume{
			testutil.AVolume(func(v *monitor.DockerVolume) {
				v.Name = "database-data"
			}),
		},
	}

	// 1. Copy network name with "y"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard: production-net") {
		t.Errorf("expected copied network name in toast, got: %s", m.actionMessage)
	}

	// 2. Open prune modal from Networks tab with "c"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Errorf("expected prune modal opened from Networks tab, got %s", m.activeModal)
	}
	m.closeModal()

	// 3. Switch to Volumes tab and copy volume name with "Y"
	m.setTab(TabVolumes)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard: database-data") {
		t.Errorf("expected copied volume name in toast, got: %s", m.actionMessage)
	}

	// 4. Open prune modal from Volumes tab with "C"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Errorf("expected prune modal opened from Volumes tab, got %s", m.activeModal)
	}
}

func TestListNavigationHomeEndPgUpPgDown(t *testing.T) {
	m := Model{
		version:         "v0.18.1",
		activeTab:       TabDocker,
		refreshInterval: 2 * time.Second,
		dockerInfo: monitor.DockerSummary{
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithName("c0")),
				testutil.AContainer(testutil.WithName("c1")),
				testutil.AContainer(testutil.WithName("c2")),
				testutil.AContainer(testutil.WithName("c3")),
				testutil.AContainer(testutil.WithName("c4")),
				testutil.AContainer(testutil.WithName("c5")),
				testutil.AContainer(testutil.WithName("c6")),
				testutil.AContainer(testutil.WithName("c7")),
			},
		},
	}

	// 1. Move to End with "end"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = res.(Model)
	if m.cursorIdx != 7 {
		t.Errorf("expected cursor at 7 after 'end', got %d", m.cursorIdx)
	}

	// 2. Move to Home with "home"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m = res.(Model)
	if m.cursorIdx != 0 {
		t.Errorf("expected cursor at 0 after 'home', got %d", m.cursorIdx)
	}

	// 3. Jump to End with "G"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = res.(Model)
	if m.cursorIdx != 7 {
		t.Errorf("expected cursor at 7 after 'G', got %d", m.cursorIdx)
	}

	// 4. Jump to Start with "g"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = res.(Model)
	if m.cursorIdx != 0 {
		t.Errorf("expected cursor at 0 after 'g', got %d", m.cursorIdx)
	}

	// 5. PageDown moves 5 items down
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = res.(Model)
	if m.cursorIdx != 5 {
		t.Errorf("expected cursor at 5 after 'pgdown', got %d", m.cursorIdx)
	}

	// 6. PageUp moves 5 items up
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = res.(Model)
	if m.cursorIdx != 0 {
		t.Errorf("expected cursor at 0 after 'pgup', got %d", m.cursorIdx)
	}
}

func TestComposeNetworksVolumesSearchFilter(t *testing.T) {
	m := Model{
		version:         "v0.18.1",
		activeTab:       TabCompose,
		refreshInterval: 2 * time.Second,
		compose: []monitor.ComposeProject{
			{Name: "web-stack", Status: "running", Services: []monitor.ComposeService{{Name: "web-srv", Image: "nginx"}}},
			{Name: "db-stack", Status: "running", Services: []monitor.ComposeService{{Name: "db-srv", Image: "postgres"}}},
		},
		networks: []monitor.DockerNetwork{
			{ID: "n1", Name: "bridge-net", Driver: "bridge", Subnet: "172.18.0.0/16"},
			{ID: "n2", Name: "custom-net", Driver: "overlay", Subnet: "10.0.0.0/24"},
		},
		volumes: []monitor.DockerVolume{
			{Name: "db-data", Driver: "local"},
			{Name: "cache-data", Driver: "local"},
		},
	}

	// 1. Open search on TabCompose with "/"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering true on Compose tab")
	}

	// Type "db"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = res.(Model)

	filteredCompose := m.getFilteredCompose()
	if len(filteredCompose) != 1 || filteredCompose[0].Name != "db-stack" {
		t.Errorf("expected db-stack matching 'db', got %+v", filteredCompose)
	}

	// Apply filter with Enter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Errorf("expected isFiltering false after Enter")
	}

	// View rendering contains filter badge
	v := m.View()
	if !strings.Contains(v, "Filter: db") {
		t.Errorf("expected filter badge in compose view, got: %s", v)
	}

	// Reset with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery empty after Esc, got %q", m.filterQuery)
	}
	if len(m.getFilteredCompose()) != 2 {
		t.Errorf("expected all compose projects restored")
	}

	// 2. Search on TabNetworks
	m.activeTab = TabNetworks
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering true on Networks tab")
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o', 'v', 'e', 'r', 'l', 'a', 'y'}})
	m = res.(Model)
	filteredNets := m.getFilteredNetworks()
	if len(filteredNets) != 1 || filteredNets[0].Name != "custom-net" {
		t.Errorf("expected custom-net matching overlay driver, got %+v", filteredNets)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery reset after Esc")
	}

	// 3. Search on TabVolumes
	m.activeTab = TabVolumes
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering true on Volumes tab")
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c', 'a', 'c', 'h', 'e'}})
	m = res.(Model)
	filteredVols := m.getFilteredVolumes()
	if len(filteredVols) != 1 || filteredVols[0].Name != "cache-data" {
		t.Errorf("expected cache-data matching 'cache', got %+v", filteredVols)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery reset after Esc")
	}
}

func TestNetworksAndVolumesInspectModal(t *testing.T) {
	m := Model{
		version:         "v0.18.1",
		activeTab:       TabNetworks,
		refreshInterval: 2 * time.Second,
		networks: []monitor.DockerNetwork{
			{ID: "net-xyz-123", Name: "app-net", Driver: "bridge", Subnet: "172.20.0.0/16"},
		},
		volumes: []monitor.DockerVolume{
			{Name: "data-volume", Driver: "local", Scope: "local"},
		},
	}

	// 1. Inspect on Networks tab with Enter
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.activeModal != ModalInspect {
		t.Fatalf("expected ModalInspect opened for network, got %q", m.activeModal)
	}
	if !strings.Contains(m.inspectCustomTitle, "NETWORK INSPECTION: app-net") {
		t.Errorf("expected network title, got %q", m.inspectCustomTitle)
	}
	if len(m.inspectCustomLines) == 0 {
		t.Errorf("expected inspect lines populated")
	}
	if m.inspectCopyID != "net-xyz-123" {
		t.Errorf("expected copy ID net-xyz-123, got %q", m.inspectCopyID)
	}

	// View renders the network inspection modal
	view := m.View()
	if !strings.Contains(view, "NETWORK INSPECTION: app-net") {
		t.Errorf("expected network inspection header in view, got: %s", view)
	}

	// Test copy hotkey in modal
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard") && !strings.Contains(m.actionMessage, "Clipboard error") {
		t.Errorf("expected clipboard feedback toast, got: %s", m.actionMessage)
	}

	// Close modal with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != ModalNone {
		t.Fatalf("expected modal closed, got %q", m.activeModal)
	}

	// 2. Inspect on Volumes tab with 'i'
	m.activeTab = TabVolumes
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = res.(Model)
	if m.activeModal != ModalInspect {
		t.Fatalf("expected ModalInspect opened for volume, got %q", m.activeModal)
	}
	if !strings.Contains(m.inspectCustomTitle, "VOLUME INSPECTION: data-volume") {
		t.Errorf("expected volume title, got %q", m.inspectCustomTitle)
	}
	if len(m.inspectCustomLines) == 0 {
		t.Errorf("expected inspect lines populated")
	}
	if m.inspectCopyID != "data-volume" {
		t.Errorf("expected copy ID data-volume, got %q", m.inspectCopyID)
	}

	volView := m.View()
	if !strings.Contains(volView, "VOLUME INSPECTION: data-volume") {
		t.Errorf("expected volume inspection header in view, got: %s", volView)
	}

	// Scroll down and up
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)

	// Close modal with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != ModalNone {
		t.Fatalf("expected modal closed, got %q", m.activeModal)
	}
}
