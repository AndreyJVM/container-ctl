package e2e

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

// TestE2E_FullContainerWorkflow tests navigation, sorting, search filter, selection, and inspect/logs modals.
func TestE2E_FullContainerWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Initial State verification
	view := m.View()
	if !strings.Contains(view, "web-gateway") || !strings.Contains(view, "api-backend") {
		t.Fatalf("expected initial containers to be listed in view, got: %s", view)
	}
	if !strings.Contains(view, "Containers: 2 active, 1 stopped") {
		t.Fatalf("expected summary header, got: %s", view)
	}

	// 2. Cursor movement: move down, verify active cursor changed
	m, _ = sendKey(m, "down")
	viewAfterDown := m.View()
	if !strings.Contains(viewAfterDown, "api-backend") {
		t.Errorf("expected view to render api-backend, got: %s", viewAfterDown)
	}

	// Move to bottom and back to top
	m, _ = sendKey(m, "end")
	m, _ = sendKey(m, "home")

	// 3. Cycle Sorting: "o" cycles sorting mode
	// Default -> CPU -> Memory -> Name -> Status -> Default
	for i := 0; i < 5; i++ {
		m, _ = sendKey(m, "o")
	}

	// 4. Search Filter Workflow:
	// Press '/' to enter filter mode
	m, _ = sendKey(m, "/")
	// Type query "gateway"
	m = sendTypeString(m, "gateway")
	// Confirm filter with Enter
	m, _ = sendKey(m, "enter")

	filteredView := m.View()
	if !strings.Contains(filteredView, "web-gateway") {
		t.Errorf("expected filtered view to contain 'web-gateway'")
	}
	if strings.Contains(filteredView, "api-backend") {
		t.Errorf("expected filtered view to exclude 'api-backend'")
	}

	// Clear filter with Esc
	m, _ = sendKey(m, "esc")
	restoredView := m.View()
	if !strings.Contains(restoredView, "api-backend") {
		t.Errorf("expected view after clearing filter to restore all containers")
	}

	// 5. Multi-selection Workflow:
	// Select first container with Space
	m, _ = sendKey(m, "space")
	// Move down and select second container
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "space")

	selectView := m.View()
	if !strings.Contains(selectView, "Remove (2)") && !strings.Contains(selectView, "Start/Stop (2)") {
		t.Errorf("expected view to reflect 2 selected containers in footer, got: %s", selectView)
	}

	// Open Batch Delete Modal with 'x'
	m, _ = sendKey(m, "x")
	if !m.IsModalOpen() {
		t.Fatalf("expected modal to open upon 'x' with active selection")
	}
	if m.ActiveModal() != tui.ModalRemoveContainer {
		t.Errorf("expected ModalRemoveContainer, got: %s", m.ActiveModal())
	}
	batchModalView := m.View()
	if !strings.Contains(batchModalView, "CONFIRM REMOVAL OF 2 CONTAINERS") {
		t.Errorf("expected batch deletion confirmation prompt, got: %s", batchModalView)
	}

	// Cancel modal with Esc
	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected modal to close on Esc")
	}

	// Clear selection with Esc
	m, _ = sendKey(m, "esc")

	// 6. Inspect Modal Workflow:
	m.OpenModal(tui.ModalInspect)
	m.SetInspectForTesting(monitor.ContainerDetail{
		ID:     "c101",
		Name:   "web-gateway",
		State:  "running",
		Status: "Up 4 hours",
		Image:  "nginx:alpine",
		IP:     "172.17.0.2",
	})
	inspectView := m.View()
	if !strings.Contains(inspectView, "CONTAINER INSPECTION") && !strings.Contains(inspectView, "web-gateway") {
		t.Errorf("expected inspect modal to render container details, got: %s", inspectView)
	}

	// Scroll inspect view
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "up")

	// Close inspect modal with Esc
	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected inspect modal to close on Esc")
	}

	// 7. Toggle Details card with 'v'
	m, _ = sendKey(m, "v")
	viewNoDetails := m.View()
	if !strings.Contains(viewNoDetails, "Details card hidden") {
		t.Errorf("expected toast or indicator after toggling details off")
	}
	// Restore details card
	m, _ = sendKey(m, "v")

	// 8. Copy resource ID / command with 'y' (yank)
	m, _ = sendKey(m, "y")
	copyView := m.View()
	if !strings.Contains(copyView, "Copied") && !strings.Contains(copyView, "Clipboard") {
		t.Errorf("expected clipboard notification toast on 'y'")
	}
}

// TestE2E_MultiTabAndSelectionWorkflow tests navigating across all tabs and verifying multi-selection on Images, Networks, and Volumes.
func TestE2E_MultiTabAndSelectionWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Tab switching order: 1: Docker -> 2: Images -> 3: Compose -> 4: Networks -> 5: Volumes -> 6: Swarm
	tabSequence := []struct {
		key         string
		expectedTab tui.TabID
		headerText  string
	}{
		{"2", tui.TabImages, "DOCKER IMAGES"},
		{"3", tui.TabCompose, "DOCKER COMPOSE"},
		{"4", tui.TabNetworks, "DOCKER NETWORKS"},
		{"5", tui.TabVolumes, "DOCKER VOLUMES"},
		{"6", tui.TabSwarm, "SWARM SERVICES"},
		{"1", tui.TabDocker, "DOCKER CONTAINERS"},
	}

	for _, step := range tabSequence {
		m, _ = sendKey(m, step.key)
		if m.ActiveTab() != step.expectedTab {
			t.Fatalf("after key %q: expected active tab %s, got %s", step.key, step.expectedTab, m.ActiveTab())
		}
		view := m.View()
		if !strings.Contains(view, step.headerText) {
			t.Errorf("expected tab view to contain %q, got: %s", step.headerText, view)
		}
	}

	// Test multi-select on Images tab
	m, _ = sendKey(m, "2")
	m, _ = sendKey(m, "a") // Select all images
	viewImages := m.View()
	if !strings.Contains(viewImages, "3 selected") && !strings.Contains(viewImages, "Selected: 3") && !strings.Contains(viewImages, "Delete (3)") {
		t.Errorf("expected images select all to show 3 selected, view: %s", viewImages)
	}

	// Open Batch Remove Images modal with 'x'
	m, _ = sendKey(m, "x")
	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalRemoveImage {
		t.Fatalf("expected ModalRemoveImage upon pressing 'x' with images selected")
	}
	m, _ = sendKey(m, "esc") // Cancel modal
	m, _ = sendKey(m, "esc") // Clear selection

	// Test multi-select on Networks tab
	m, _ = sendKey(m, "4")
	m, _ = sendKey(m, "space") // Select first network
	m, _ = sendKey(m, "x")     // Open Remove modal
	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalRemoveNetwork {
		t.Fatalf("expected ModalRemoveNetwork upon pressing 'x' with network selected")
	}
	m, _ = sendKey(m, "esc")
	m, _ = sendKey(m, "esc")

	// Test multi-select on Volumes tab
	m, _ = sendKey(m, "5")
	m, _ = sendKey(m, "space") // Select first volume
	m, _ = sendKey(m, "x")     // Open Remove modal
	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalRemoveVolume {
		t.Fatalf("expected ModalRemoveVolume upon pressing 'x' with volume selected")
	}
	m, _ = sendKey(m, "esc")
	m, _ = sendKey(m, "esc")
}

// TestE2E_SwarmScalingAndHelpModal tests scaling swarm service replicas and opening/closing the help overlay.
func TestE2E_SwarmScalingAndHelpModal(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Switch to Swarm tab
	m, _ = sendKey(m, "6")
	if m.ActiveTab() != tui.TabSwarm {
		t.Fatalf("expected TabSwarm, got: %s", m.ActiveTab())
	}

	// Open Swarm scale modal with 's' or 'S'
	m, _ = sendKey(m, "s")
	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalSwarmScale {
		t.Fatalf("expected ModalSwarmScale upon pressing 's' on Swarm tab")
	}

	// Cancel scale modal
	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected scale modal to close on Esc")
	}

	// Open Help overlay with '?'
	m, _ = sendKey(m, "?")
	helpView := m.View()
	if !strings.Contains(helpView, "KEYBOARD SHORTCUTS") && !strings.Contains(helpView, "Help") && !strings.Contains(helpView, "Toggle this help window") {
		t.Errorf("expected Help modal to be rendered on '?', got: %s", helpView)
	}

	// Close Help overlay with '?' or Esc
	m, _ = sendKey(m, "esc")
	normalView := m.View()
	if strings.Contains(normalView, "KEYBOARD SHORTCUTS") {
		t.Errorf("expected Help modal to be dismissed on Esc")
	}

	// Graceful Quit on 'q'
	_, quitCmd := sendKey(m, "q")
	if quitCmd == nil {
		t.Errorf("expected quitCmd when pressing 'q'")
	}
}

// TestE2E_SelectionPrunedOnResourceRemoval verifies that when resources are deleted or
// terminated externally on Docker daemon, background data updates (actions.DataFetchedMsg)
// automatically prune stale IDs from selection maps, preventing ghost selections.
func TestE2E_SelectionPrunedOnResourceRemoval(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Containers: Select c101 and c102
	m, _ = sendKey(m, "space") // selects c101, cursor moves to c102
	m, _ = sendKey(m, "space") // selects c102
	if m.SelectedContainersCount() != 2 {
		t.Fatalf("expected 2 selected containers, got: %d", m.SelectedContainersCount())
	}
	view := m.View()
	if !strings.Contains(view, "Remove (2)") {
		t.Fatalf("expected footer to show 'Remove (2)', got:\n%s", view)
	}

	// External event: c102 was stopped and deleted outside container-ctl
	updatedContainers := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c101"), testutil.WithName("web-gateway")),
				testutil.AContainer(testutil.WithID("c103"), testutil.WithName("redis-cache")),
			},
		},
		Images: []monitor.DockerImage{
			testutil.AnImage(testutil.WithImageID("img1", "sha256:img1")),
			testutil.AnImage(testutil.WithImageID("img2", "sha256:img2")),
			testutil.AnImage(testutil.WithImageID("img3", "sha256:img3")),
		},
		Networks: []monitor.DockerNetwork{
			{ID: "net1", Name: "frontend-bridge"},
			{ID: "net2", Name: "backend-internal"},
		},
		Volumes: []monitor.DockerVolume{
			{Name: "pg_data"},
			{Name: "redis_cache"},
		},
		Timestamp: "12:01:00",
	}

	updated, _ := m.Update(updatedContainers)
	m = updated.(tui.Model)

	// Ghost ID c102 must be pruned; only c101 remains
	if m.SelectedContainersCount() != 1 {
		t.Fatalf("expected 1 selected container after pruning, got: %d", m.SelectedContainersCount())
	}
	prunedView := m.View()
	if !strings.Contains(prunedView, "Remove (1)") {
		t.Errorf("expected footer to show 'Remove (1)', got:\n%s", prunedView)
	}

	// 2. Images: Select img1 and img2, then prune img2
	m, _ = sendKey(m, "2") // Switch to TabImages
	m, _ = sendKey(m, "space")
	m, _ = sendKey(m, "space")
	if m.SelectedImagesCount() != 2 {
		t.Fatalf("expected 2 selected images, got: %d", m.SelectedImagesCount())
	}

	updatedImages := updatedContainers
	updatedImages.Images = []monitor.DockerImage{
		testutil.AnImage(testutil.WithImageID("img1", "sha256:img1")),
		testutil.AnImage(testutil.WithImageID("img3", "sha256:img3")),
	}
	updated, _ = m.Update(updatedImages)
	m = updated.(tui.Model)

	if m.SelectedImagesCount() != 1 {
		t.Fatalf("expected 1 selected image after pruning, got: %d", m.SelectedImagesCount())
	}

	// 3. Networks: Select net1 and net2, then prune net1
	m, _ = sendKey(m, "4") // Switch to TabNetworks
	m, _ = sendKey(m, "space")
	m, _ = sendKey(m, "space")
	if m.SelectedNetworksCount() != 2 {
		t.Fatalf("expected 2 selected networks, got: %d", m.SelectedNetworksCount())
	}

	updatedNetworks := updatedImages
	updatedNetworks.Networks = []monitor.DockerNetwork{
		{ID: "net2", Name: "backend-internal"},
	}
	updated, _ = m.Update(updatedNetworks)
	m = updated.(tui.Model)

	if m.SelectedNetworksCount() != 1 {
		t.Fatalf("expected 1 selected network after pruning, got: %d", m.SelectedNetworksCount())
	}

	// 4. Volumes: Select pg_data and redis_cache, then prune pg_data
	m, _ = sendKey(m, "5") // Switch to TabVolumes
	m, _ = sendKey(m, "space")
	m, _ = sendKey(m, "space")
	if m.SelectedVolumesCount() != 2 {
		t.Fatalf("expected 2 selected volumes, got: %d", m.SelectedVolumesCount())
	}

	updatedVolumes := updatedNetworks
	updatedVolumes.Volumes = []monitor.DockerVolume{
		{Name: "redis_cache"},
	}
	updated, _ = m.Update(updatedVolumes)
	m = updated.(tui.Model)

	if m.SelectedVolumesCount() != 1 {
		t.Fatalf("expected 1 selected volume after pruning, got: %d", m.SelectedVolumesCount())
	}
}

// TestE2E_FilterZeroResults_KeyHandling verifies that navigation and selection keys
// behave safely without panic when a filter produces zero results.
func TestE2E_FilterZeroResults_KeyHandling(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Type filter query matching nothing
	m, _ = sendKey(m, "/")
	m = sendTypeString(m, "nonexistent_container_pattern_999")
	m, _ = sendKey(m, "enter")

	// Verify no items match
	view := m.View()
	if strings.Contains(view, "web-gateway") {
		t.Errorf("expected 'web-gateway' to be filtered out")
	}

	// Send various action and navigation keys on TabDocker - none should panic
	testKeys := []string{"space", "a", "enter", "s", "r", "k", "d", "j", "k", "down", "up"}
	for _, k := range testKeys {
		m, _ = sendKey(m, k)
	}

	// Esc should clear filter and restore all containers
	m, _ = sendKey(m, "esc")
	restoredView := m.View()
	if !strings.Contains(restoredView, "web-gateway") {
		t.Errorf("expected 'web-gateway' to be visible after clearing filter with Esc")
	}
}

// TestE2E_BackgroundRefreshDuringModals verifies that background data polling
// (DataFetchedMsg) does not close or corrupt active overlay dialogs.
func TestE2E_BackgroundRefreshDuringModals(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Help overlay open
	m, _ = sendKey(m, "?")
	if !m.IsHelpOpen() {
		t.Fatalf("expected Help overlay to be open")
	}

	// Background data fetch arrives while user is reading Help
	refreshed, _ := m.Update(actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{Status: "running"},
		Timestamp:  "12:02:00",
	})
	m = refreshed.(tui.Model)

	// Help overlay must stay open
	if !m.IsHelpOpen() {
		t.Errorf("expected Help overlay to remain open after background data fetch")
	}

	// Close help
	m, _ = sendKey(m, "esc")
	if m.IsHelpOpen() {
		t.Errorf("expected Help overlay closed on Esc")
	}

	// 2. Inspect Modal open
	m.OpenModal(tui.ModalInspect)
	m.SetInspectForTesting(monitor.ContainerDetail{
		ID:    "c101",
		Name:  "web-gateway",
		Image: "nginx:alpine",
	})

	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalInspect {
		t.Fatalf("expected ModalInspect open")
	}

	// Background data fetch arrives
	refreshed, _ = m.Update(actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{Status: "running"},
		Timestamp:  "12:03:00",
	})
	m = refreshed.(tui.Model)

	if !m.IsModalOpen() || m.ActiveModal() != tui.ModalInspect {
		t.Errorf("expected ModalInspect to remain open across background refresh")
	}

	m, _ = sendKey(m, "esc")
	if m.IsModalOpen() {
		t.Errorf("expected ModalInspect closed on Esc")
	}
}
