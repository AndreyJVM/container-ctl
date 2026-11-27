package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/module"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	// Defaults when invalid arguments are passed, along with showSwarm variadic flag
	m := NewModel(0, 99, "v1.0.0", true)
	if m.activeTab != TabDocker {
		t.Errorf("expected activeTab to default to Docker, got %d", m.activeTab)
	}
	if m.refreshInterval != 2*time.Second {
		t.Errorf("expected refreshInterval to default to 2s, got %v", m.refreshInterval)
	}
	if m.version != "v1.0.0" {
		t.Errorf("expected version 'v1.0.0', got %q", m.version)
	}
	if !m.showSwarm {
		t.Errorf("expected showSwarm=true via variadic flag")
	}

	// Init command
	cmd := m.Init()
	if cmd == nil {
		t.Errorf("expected non-nil Init cmd")
	}
}

func TestDockerIsFirstTabAndStatusRemainsVisibleAcrossTabs(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.width = 120
	m.dockerInfo = monitor.DockerSummary{
		Status:            "running",
		RunningContainers: 2,
		StoppedContainers: 1,
		ImagesCount:       7,
		Version:           "27.4.0",
	}

	dockerView := m.View()
	if !strings.Contains(dockerView, "[1: Containers]") {
		t.Fatalf("expected Docker to be the first active tab, got: %s", dockerView)
	}
	if strings.Contains(dockerView, "Host Overview") || strings.Contains(dockerView, "SYSTEM INFORMATION") {
		t.Fatalf("expected the Host page to be removed, got: %s", dockerView)
	}
	if !strings.Contains(dockerView, "Containers: 2 active, 1 stopped") {
		t.Fatalf("expected Docker status summary at the top, got: %s", dockerView)
	}

	m.setTab(TabCompose)
	composeView := m.View()
	if !strings.Contains(composeView, "Containers: 2 active, 1 stopped") {
		t.Fatalf("expected Docker status summary to remain visible on Compose, got: %s", composeView)
	}
}

func TestViewIncludesNetworksAndVolumesTabs(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.width = 120
	view := m.View()

	for _, label := range []string{"2: Images", "3: Compose", "4: Networks", "5: Volumes"} {
		if !strings.Contains(view, label) {
			t.Fatalf("expected tab %q in view, got: %s", label, view)
		}
	}
	if strings.Contains(view, "6: Swarm") {
		t.Fatalf("expected inactive Swarm tab to be hidden by default")
	}

	// Show Swarm when forced or active
	m.SetShowSwarm(true)
	viewWithSwarm := m.View()
	if !strings.Contains(viewWithSwarm, "6: Swarm") {
		t.Fatalf("expected tab 6: Swarm in view when showSwarm is enabled")
	}
}

func TestModelNavigationAndUpdates(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.SetShowSwarm(true)

	// Resize
	updatedModel, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Errorf("expected nil cmd on window resize")
	}
	m = updatedModel.(Model)
	if m.width != 120 || m.height != 40 {
		t.Errorf("expected dimensions 120x40, got %dx%d", m.width, m.height)
	}

	// Tab switching
	keyTests := []struct {
		key         string
		expectedTab TabID
	}{
		{"tab", TabImages},
		{"tab", TabCompose},
		{"tab", TabNetworks},
		{"right", TabVolumes},
		{"tab", TabSwarm},
		{"tab", TabDocker},
		{"left", TabSwarm},
		{"shift+tab", TabVolumes},
		{"1", TabDocker},
		{"2", TabImages},
		{"3", TabCompose},
		{"4", TabNetworks},
		{"5", TabVolumes},
		{"6", TabSwarm},
	}

	for _, kt := range keyTests {
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(kt.key)})
		m = res.(Model)
		if m.activeTab != kt.expectedTab {
			t.Errorf("after key %q: expected tab %d, got %d", kt.key, kt.expectedTab, m.activeTab)
		}
	}

	// Help modal toggle
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = res.(Model)
	if !m.showHelp {
		t.Errorf("expected showHelp=true after '?'")
	}

	// Close help
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.showHelp {
		t.Errorf("expected showHelp=false after 'esc'")
	}

	// Quit keys
	_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if quitCmd == nil {
		t.Errorf("expected quitCmd on ctrl+c")
	}
}

func TestRenderAllTabs(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.width = 120
	m.height = 40

	// Render Tab 1: Docker
	m.activeTab = TabDocker
	v1 := m.View()
	if !strings.Contains(v1, "DOCKER CONTAINERS") {
		t.Errorf("expected Docker tab header in View()")
	}

	// Render Tab 2: Images
	m.activeTab = TabImages
	v2 := m.View()
	if !strings.Contains(v2, "DOCKER IMAGES") {
		t.Errorf("expected Images tab header in View()")
	}

	// Render Tab 3: Compose
	m.activeTab = TabCompose
	v3 := m.View()
	if !strings.Contains(v3, "DOCKER COMPOSE") {
		t.Errorf("expected Compose tab header in View()")
	}

	// Render Tab 4: Networks
	m.activeTab = TabNetworks
	v4 := m.View()
	if !strings.Contains(v4, "DOCKER NETWORKS") {
		t.Errorf("expected Networks tab header in View()")
	}

	// Render Tab 5: Volumes
	m.activeTab = TabVolumes
	v5 := m.View()
	if !strings.Contains(v5, "DOCKER VOLUMES") {
		t.Errorf("expected Volumes tab header in View()")
	}

	// Render Tab 6: Swarm
	m.activeTab = TabSwarm
	v6 := m.View()
	if !strings.Contains(v6, "DOCKER SWARM") {
		t.Errorf("expected Swarm tab header in View()")
	}

	// Render Help modal
	m.showHelp = true
	vHelp := m.View()
	if !strings.Contains(vHelp, "HOTKEYS") {
		t.Errorf("expected Help modal in View()")
	}
}

func TestNavigationMethods(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()

	// Inactive Swarm tab wrapping: 1 -> 5
	m.prevTab()
	if m.activeTab != TabVolumes {
		t.Errorf("expected wrap to TabVolumes when Swarm is inactive, got %v", m.activeTab)
	}
	m.nextTab()
	if m.activeTab != TabDocker {
		t.Errorf("expected wrap to TabDocker, got %v", m.activeTab)
	}

	// Active/Forced Swarm tab wrapping: 1 -> 6 -> 1
	m.SetShowSwarm(true)
	m.setTab(TabSwarm)
	m.nextTab()
	if m.activeTab != TabDocker {
		t.Errorf("expected wrap to TabDocker, got %v", m.activeTab)
	}

	// prevTab wrapping to TabSwarm
	m.prevTab()
	if m.activeTab != TabSwarm {
		t.Errorf("expected wrap to TabSwarm, got %v", m.activeTab)
	}

	// setTab valid and invalid
	m.setTab(TabDocker)
	if m.activeTab != TabDocker {
		t.Errorf("expected TabDocker, got %v", m.activeTab)
	}
	m.setTab(99) // Invalid, should remain unchanged
	if m.activeTab != TabDocker {
		t.Errorf("expected TabDocker after invalid tab, got %v", m.activeTab)
	}

	// moveCursor in Docker tab
	m.dockerInfo.Containers = []monitor.ContainerItem{
		{ID: "c1", Name: "app-1"},
		{ID: "c2", Name: "app-2"},
		{ID: "c3", Name: "app-3"},
	}
	m.cursorIdx = 0
	m.moveCursor(1)
	if m.cursorIdx != 1 {
		t.Errorf("expected cursorIdx 1, got %d", m.cursorIdx)
	}
	m.moveCursor(-1)
	if m.cursorIdx != 0 {
		t.Errorf("expected cursorIdx 0, got %d", m.cursorIdx)
	}
	m.moveCursor(-1) // Wrap to end
	if m.cursorIdx != 2 {
		t.Errorf("expected cursorIdx wrapped to 2, got %d", m.cursorIdx)
	}
	m.moveCursor(1) // Wrap back to 0
	if m.cursorIdx != 0 {
		t.Errorf("expected cursorIdx wrapped to 0, got %d", m.cursorIdx)
	}

	// moveCursor in Images tab
	m.setTab(TabImages)
	m.images = []monitor.DockerImage{
		{ID: "img1", Repository: "repo1"},
		{ID: "img2", Repository: "repo2"},
	}
	m.imageCursorIdx = 0
	m.moveCursor(1)
	if m.imageCursorIdx != 1 {
		t.Errorf("expected imageCursorIdx 1, got %d", m.imageCursorIdx)
	}
	m.moveCursor(1) // Wrap
	if m.imageCursorIdx != 0 {
		t.Errorf("expected imageCursorIdx 0, got %d", m.imageCursorIdx)
	}

	// moveCursor in Compose tab
	m.setTab(TabCompose)
	m.compose = []monitor.ComposeProject{
		{Name: "p1"},
		{Name: "p2"},
	}
	m.composeCursorIdx = 0
	m.moveCursor(1)
	if m.composeCursorIdx != 1 {
		t.Errorf("expected composeCursorIdx 1, got %d", m.composeCursorIdx)
	}
	m.moveCursor(1) // Wrap
	if m.composeCursorIdx != 0 {
		t.Errorf("expected composeCursorIdx 0, got %d", m.composeCursorIdx)
	}

	// moveCursor in Networks tab
	m.setTab(TabNetworks)
	m.networks = []monitor.DockerNetwork{{Name: "frontend"}, {Name: "backend"}}
	m.moveCursor(1)
	if m.networkCursorIdx != 1 {
		t.Errorf("expected networkCursorIdx 1, got %d", m.networkCursorIdx)
	}

	// moveCursor in Volumes tab
	m.setTab(TabVolumes)
	m.volumes = []monitor.DockerVolume{{Name: "data"}, {Name: "cache"}}
	m.moveCursor(-1)
	if m.volumeCursorIdx != 1 {
		t.Errorf("expected volumeCursorIdx wrapped to 1, got %d", m.volumeCursorIdx)
	}

	// moveCursor in Swarm tab
	m.setTab(TabSwarm)
	m.swarm.Services = []monitor.SwarmService{
		{ID: "s1", Name: "svc-1"},
		{ID: "s2", Name: "svc-2"},
	}
	m.swarmCursorIdx = 0
	m.moveCursor(1)
	if m.swarmCursorIdx != 1 {
		t.Errorf("expected swarmCursorIdx 1, got %d", m.swarmCursorIdx)
	}
	m.moveCursor(1) // Wrap
	if m.swarmCursorIdx != 0 {
		t.Errorf("expected swarmCursorIdx 0, got %d", m.swarmCursorIdx)
	}
}

func TestModalLifecycle(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	if m.ActiveModal() != ModalNone {
		t.Errorf("expected initial ActiveModal to be ModalNone, got %v", m.ActiveModal())
	}

	// Test ModalExec
	m.OpenModal(ModalExec)
	m.execIsCustomInput = true
	m.execCustomCmd = "top"
	if m.ActiveModal() != ModalExec {
		t.Errorf("expected ModalExec, got %v", m.ActiveModal())
	}

	m.CloseModal()
	if m.ActiveModal() != ModalNone {
		t.Errorf("expected ModalNone after CloseModal, got %v", m.ActiveModal())
	}
	if m.execIsCustomInput {
		t.Errorf("expected execIsCustomInput to be reset to false")
	}
	if m.execCustomCmd != "" {
		t.Errorf("expected execCustomCmd to be reset to empty")
	}

	// Test ModalLogs resets log filtering
	m.logsFiltering = true
	m.logsFilterQuery = "error"
	m.openModal(ModalLogs)
	if m.logsFiltering || m.logsFilterQuery != "" {
		t.Errorf("expected logs filtering state reset on openModal(ModalLogs)")
	}

	// Test ModalInspect resets inspect scroll
	m.inspectScroll = 15
	m.openModal(ModalInspect)
	if m.inspectScroll != 0 {
		t.Errorf("expected inspectScroll reset to 0 on openModal(ModalInspect)")
	}
}

func TestExportedHelpers(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()

	// Tab helpers
	m.SetTab(TabNetworks)
	if m.ActiveTab() != TabNetworks {
		t.Errorf("expected ActiveTab=TabNetworks, got %d", m.ActiveTab())
	}

	// Cursor helper
	m.SetCursorForTesting(4)
	if m.cursorIdx != 4 || m.imageCursorIdx != 4 || m.composeCursorIdx != 4 ||
		m.networkCursorIdx != 4 || m.volumeCursorIdx != 4 || m.swarmCursorIdx != 4 {
		t.Errorf("SetCursorForTesting failed to set all tab cursor indices")
	}

	// Logs helper
	m.SetLogsForTesting("web-app", "c123", []string{"line 1", "line 2"})
	if m.logsTargetName != "web-app" || m.logsTargetID != "c123" || len(m.logsLines) != 2 {
		t.Errorf("SetLogsForTesting failed")
	}

	// Inspect helper
	m.SetInspectForTesting(monitor.ContainerDetail{ID: "c123", Name: "web-app"})
	if m.inspectTarget.ID != "c123" || m.inspectTarget.Name != "web-app" {
		t.Errorf("SetInspectForTesting failed")
	}
}

func TestFormatFetchError(t *testing.T) {
	t.Parallel()

	if errStr := formatFetchError(nil); errStr != "" {
		t.Errorf("expected empty string for nil error, got %q", errStr)
	}

	expected := "connection refused"
	if errStr := formatFetchError(errors.New(expected)); errStr != expected {
		t.Errorf("expected %q, got %q", expected, errStr)
	}
}

func TestContainerStatusScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state  string
		health string
		status string
		score  int
	}{
		{"dead", "", "Dead", 1},
		{"exited", "", "Exited (1) 2m ago", 1},
		{"exited", "", "Exited (137) 10s ago", 1},
		{"running", "unhealthy", "Up 2h (unhealthy)", 1},
		{"running", "starting", "Up 1m (health: starting)", 2},
		{"running", "healthy", "Up 2h (healthy)", 3},
		{"running", "", "Up 5h", 3},
		{"restarting", "", "Restarting", 4},
		{"paused", "", "Paused", 5},
		{"exited", "", "Exited (0) 5m ago", 6},
		{"exited", "", "", 6},
		{"unknown", "", "", 6},
	}

	for _, tt := range tests {
		s := containerStatusScore(monitor.ContainerItem{
			State:  tt.state,
			Health: tt.health,
			Status: tt.status,
		})
		if s != tt.score {
			t.Errorf("for state=%q health=%q status=%q expected score %d, got %d", tt.state, tt.health, tt.status, tt.score, s)
		}
	}
}

func TestDynamicSwarmTabLifecycle(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.swarm.IsActive = false
	m.showSwarm = false

	// 1. Swarm tab hidden, pressing "6" shows toast and stays on TabDocker
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")})
	m = res.(Model)
	if m.activeTab != TabDocker {
		t.Fatalf("expected activeTab to remain TabDocker when Swarm inactive, got %v", m.activeTab)
	}
	if !strings.Contains(m.actionMessage, "Host is not part of an active Swarm cluster.") {
		t.Fatalf("expected toast %q, got %q", "Host is not part of an active Swarm cluster.", m.actionMessage)
	}

	// 2. Swarm becomes active: tab 6 is now navigable
	m.swarm.IsActive = true
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")})
	m = res.(Model)
	if m.activeTab != TabSwarm {
		t.Fatalf("expected activeTab to switch to TabSwarm when active, got %v", m.activeTab)
	}

	// 3. Swarm deactivated while on TabSwarm: applyFetchedData redirects to TabDocker
	m.applyFetchedData(dataFetchedMsg{
		Swarm:     monitor.SwarmSummary{IsActive: false},
		Timestamp: "12:00:00",
	})
	if m.activeTab != TabDocker {
		t.Fatalf("expected fallback to TabDocker after Swarm deactivation, got %v", m.activeTab)
	}
	if !strings.Contains(m.actionMessage, "Host is not part of an active Swarm cluster.") {
		t.Fatalf("expected toast on deactivation fallback, got %q", m.actionMessage)
	}
}

func TestModel_CopyToClipboard(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.copyToClipboard("test-resource-id")
	if !strings.Contains(m.actionMessage, "Copied to clipboard: test-resource-id") && !strings.Contains(m.actionMessage, "Clipboard error:") {
		t.Errorf("unexpected actionMessage after copyToClipboard: %q", m.actionMessage)
	}
}

func TestModel_ApplyActionsData(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.applyActionsData(actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{RunningContainers: 5},
		Timestamp:  "12:34:56",
	})

	if m.dockerInfo.RunningContainers != 5 {
		t.Errorf("expected 5 running containers, got %d", m.dockerInfo.RunningContainers)
	}
	if m.lastUpdate != "12:34:56" {
		t.Errorf("expected timestamp 12:34:56, got %s", m.lastUpdate)
	}
}

func TestModel_FetchAllData_FilteredByRegistry(t *testing.T) {
	m := newFastTestModel()
	r := module.NewRegistry()
	r.Register(module.ContainersModule)
	m.SetRegistry(r)

	// Fetch data via registry filter
	m.fetchAllData()
	if len(m.lastUpdate) != 8 {
		t.Errorf("expected valid timestamp after fetchAllData, got %s", m.lastUpdate)
	}
}

func TestModel_HiddenTabFallback(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	r := module.NewRegistry()
	r.Register(module.ContainersModule)
	r.Register(module.ImagesModule)
	m.SetRegistry(r)

	// Start on TabImages
	m.activeTab = TabImages

	// Disable Images
	disabled := false
	r.SetModuleConfig(module.IDImages, module.Config{Enabled: &disabled})

	// When applying fresh data, activeTab should fall back to TabDocker
	m.applyActionsData(actions.DataFetchedMsg{
		Timestamp: "12:00:00",
	})

	if m.activeTab != TabDocker {
		t.Errorf("expected activeTab to fall back to TabDocker when Images disabled, got %v", m.activeTab)
	}
}

func TestTabID_StringAndValidation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tab      TabID
		expected string
		valid    bool
	}{
		{TabDocker, "Containers", true},
		{TabImages, "Images", true},
		{TabCompose, "Compose", true},
		{TabNetworks, "Networks", true},
		{TabVolumes, "Volumes", true},
		{TabSwarm, "Swarm", true},
		{TabID(0), "Tab(0)", false},
		{TabID(99), "Tab(99)", false},
	}

	for _, tc := range cases {
		if tc.tab.String() != tc.expected {
			t.Errorf("expected TabID(%d).String() = %q, got %q", tc.tab, tc.expected, tc.tab.String())
		}
		if tc.tab.IsValid() != tc.valid {
			t.Errorf("expected TabID(%d).IsValid() = %v, got %v", tc.tab, tc.valid, tc.tab.IsValid())
		}
	}
}

func TestModalType_StringAndIsOpen(t *testing.T) {
	t.Parallel()

	if ModalNone.IsOpen() {
		t.Errorf("expected ModalNone.IsOpen() = false")
	}
	if !ModalLogs.IsOpen() {
		t.Errorf("expected ModalLogs.IsOpen() = true")
	}
	if ModalInspect.String() != "inspect" {
		t.Errorf("expected ModalInspect.String() = 'inspect', got %q", ModalInspect.String())
	}

	var m Model
	if m.IsModalOpen() {
		t.Errorf("expected default model to have no open modal")
	}
	m.OpenModal(ModalPrune)
	if !m.IsModalOpen() {
		t.Errorf("expected IsModalOpen() = true after OpenModal(ModalPrune)")
	}
	m.CloseModal()
	if m.IsModalOpen() {
		t.Errorf("expected IsModalOpen() = false after CloseModal()")
	}
}

func TestNewModelWithOptions(t *testing.T) {
	t.Parallel()

	reg := module.NewRegistry()
	reg.Register(module.ContainersModule)

	m := NewModelWithOptions(
		3*time.Second,
		int(TabCompose),
		"v1.5.0",
		WithShowSwarm(true),
		WithRegistry(reg),
		WithOnlyUnusedImages(true),
		WithSortMode(SortCPU),
		WithHideDetails(true),
	)

	if m.refreshInterval != 3*time.Second {
		t.Errorf("expected interval 3s, got %v", m.refreshInterval)
	}
	if m.activeTab != TabCompose {
		t.Errorf("expected active tab Compose, got %v", m.activeTab)
	}
	if m.version != "v1.5.0" {
		t.Errorf("expected version v1.5.0, got %q", m.version)
	}
	if !m.showSwarm {
		t.Errorf("expected showSwarm = true")
	}
	if m.Registry() != reg {
		t.Errorf("expected custom registry to be set")
	}
	if !m.onlyUnusedImages {
		t.Errorf("expected onlyUnusedImages = true")
	}
	if m.containerSortMode != SortCPU {
		t.Errorf("expected sort mode SortCPU, got %v", m.containerSortMode)
	}
	if !m.hideDetails {
		t.Errorf("expected hideDetails = true")
	}
}

func TestCloseModal_FullCleanup(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.OpenModal(ModalRemoveContainer)

	// Populate various modal fields to verify they are all cleared
	m.execIsCustomInput = true
	m.execCustomCmd = "sh"
	m.execTargetID = "c123"
	m.execTargetName = "my-cont"
	m.logsLines = []string{"line 1", "line 2"}
	m.inspectTarget = monitor.ContainerDetail{ID: "c123"}
	m.inspectCustomLines = []string{"inspect 1"}
	m.removeContainerBatch = []monitor.ContainerItem{{ID: "c1"}}
	m.removeImageBatch = []monitor.DockerImage{{ID: "img1"}}
	m.removeNetworkBatch = []monitor.DockerNetwork{{ID: "net1"}}
	m.removeVolumeBatch = []monitor.DockerVolume{{Name: "vol1"}}
	m.swarmScaleTargetID = "srv1"
	m.swarmScaleDesired = 5

	m.CloseModal()

	if m.activeModal != ModalNone {
		t.Errorf("expected activeModal = ModalNone")
	}
	if m.execIsCustomInput || m.execCustomCmd != "" || m.execTargetID != "" {
		t.Errorf("expected exec state cleared")
	}
	if len(m.logsLines) != 0 {
		t.Errorf("expected logsLines cleared")
	}
	if m.inspectTarget.ID != "" || len(m.inspectCustomLines) != 0 {
		t.Errorf("expected inspect state cleared")
	}
	if len(m.removeContainerBatch) != 0 || len(m.removeImageBatch) != 0 || len(m.removeNetworkBatch) != 0 || len(m.removeVolumeBatch) != 0 {
		t.Errorf("expected remove batches cleared")
	}
	if m.swarmScaleTargetID != "" || m.swarmScaleDesired != 0 {
		t.Errorf("expected swarm scale state cleared")
	}
}

func TestModel_FetchDataCmd(t *testing.T) {
	m := newFastTestModel()
	reg := module.NewRegistry()
	reg.Register(module.ContainersModule)
	m.SetRegistry(reg)

	cmd := m.FetchDataCmd()
	if cmd == nil {
		t.Fatalf("expected non-nil tea.Cmd from FetchDataCmd()")
	}

	msg := cmd()
	fetchedMsg, ok := msg.(actions.DataFetchedMsg)
	if !ok {
		t.Fatalf("expected DataFetchedMsg, got %T", msg)
	}
	if len(fetchedMsg.Timestamp) != 8 {
		t.Errorf("expected valid timestamp in fetched message, got %q", fetchedMsg.Timestamp)
	}
}

func TestModel_ShowToastAndClipboardExported(t *testing.T) {
	m := newFastTestModel()
	m.ShowToast("hello test")
	if m.actionMessage != "hello test" {
		t.Errorf("expected actionMessage = 'hello test', got %q", m.actionMessage)
	}

	m.CopyToClipboard("exported-id")
	if m.actionMessage == "" {
		t.Errorf("expected non-empty actionMessage after CopyToClipboard")
	}
}
