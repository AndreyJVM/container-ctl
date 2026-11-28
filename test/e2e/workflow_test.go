package e2e

import (
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
	assertContains(t, view, "web-gateway")
	assertContains(t, view, "api-backend")
	assertContains(t, view, "Containers: 2 active, 1 stopped")

	// 2. Cursor movement: move down, verify active cursor changed
	m, _ = sendKey(m, "down")
	assertContains(t, m.View(), "api-backend")

	// Move to bottom and back to top
	m, _ = sendKey(m, "end")
	m, _ = sendKey(m, "home")

	// 3. Cycle Sorting: "o" cycles sorting mode
	// Default -> CPU -> Memory -> Name -> Status -> Default
	for i := 0; i < 5; i++ {
		m, _ = sendKey(m, "o")
	}

	// 4. Search Filter Workflow:
	m = sendFilterAndConfirm(m, "gateway")
	filteredView := m.View()
	assertContains(t, filteredView, "web-gateway")
	assertNotContains(t, filteredView, "api-backend")

	// Clear filter with Esc
	m, _ = sendKey(m, "esc")
	assertContains(t, m.View(), "api-backend")

	// 5. Multi-selection Workflow:
	// Select first container with Space
	m, _ = sendKey(m, "space")
	// Move down and select second container
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "space")
	assertSelectedContainersCount(t, m, 2)

	selectView := m.View()
	assertContains(t, selectView, "Remove (2)")
	assertContains(t, selectView, "Start/Stop (2)")

	// Open Batch Delete Modal with 'x'
	m, _ = sendKey(m, "x")
	assertModal(t, m, tui.ModalRemoveContainer)
	assertContains(t, m.View(), "CONFIRM REMOVAL OF 2 CONTAINERS")

	// Cancel modal with Esc
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// Clear selection with Esc
	m, _ = sendKey(m, "esc")
	assertSelectedContainersCount(t, m, 0)

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
	assertContains(t, inspectView, "CONTAINER INSPECTION")
	assertContains(t, inspectView, "web-gateway")

	// Scroll inspect view
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "up")

	// Close inspect modal with Esc
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// 7. Toggle Details card with 'v'
	m, _ = sendKey(m, "v")
	assertContains(t, m.View(), "Details card hidden")
	// Restore details card
	m, _ = sendKey(m, "v")

	// 8. Copy resource ID / command with 'y' (yank)
	m, _ = sendKey(m, "y")
	assertContainsAny(t, m.View(), "Copied", "Clipboard")
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
		assertTab(t, m, step.expectedTab)
		assertContains(t, m.View(), step.headerText)
	}

	// Test multi-select on Images tab
	m, _ = sendKey(m, "2")
	m, _ = sendKey(m, "a") // Select all images
	assertSelectedImagesCount(t, m, 3)

	// Open Batch Remove Images modal with 'x'
	m, _ = sendKey(m, "x")
	assertModal(t, m, tui.ModalRemoveImage)
	m, _ = sendKey(m, "esc") // Cancel modal
	assertModalClosed(t, m)
	m, _ = sendKey(m, "esc") // Clear selection
	assertSelectedImagesCount(t, m, 0)

	// Test multi-select on Networks tab
	m, _ = sendKey(m, "4")
	m, _ = sendKey(m, "space") // Select first network
	assertSelectedNetworksCount(t, m, 1)
	m, _ = sendKey(m, "x") // Open Remove modal
	assertModal(t, m, tui.ModalRemoveNetwork)
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
	m, _ = sendKey(m, "esc")
	assertSelectedNetworksCount(t, m, 0)

	// Test multi-select on Volumes tab
	m, _ = sendKey(m, "5")
	m, _ = sendKey(m, "space") // Select first volume
	assertSelectedVolumesCount(t, m, 1)
	m, _ = sendKey(m, "x") // Open Remove modal
	assertModal(t, m, tui.ModalRemoveVolume)
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
	m, _ = sendKey(m, "esc")
	assertSelectedVolumesCount(t, m, 0)
}

// TestE2E_SwarmScalingAndHelpModal tests scaling swarm service replicas and opening/closing the help overlay.
func TestE2E_SwarmScalingAndHelpModal(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Switch to Swarm tab
	m, _ = sendKey(m, "6")
	assertTab(t, m, tui.TabSwarm)

	// Open Swarm scale modal with 's' or 'S'
	m, _ = sendKey(m, "s")
	assertModal(t, m, tui.ModalSwarmScale)

	// Cancel scale modal
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// Open Help overlay with '?'
	m, _ = sendKey(m, "?")
	assertHelpOpen(t, m)
	assertContains(t, m.View(), "KEYBOARD SHORTCUTS")

	// Close Help overlay with '?' or Esc
	m, _ = sendKey(m, "esc")
	assertHelpClosed(t, m)
	assertNotContains(t, m.View(), "KEYBOARD SHORTCUTS")

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
	assertSelectedContainersCount(t, m, 2)
	assertContains(t, m.View(), "Remove (2)")

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
	assertSelectedContainersCount(t, m, 1)
	assertContains(t, m.View(), "Remove (1)")

	// 2. Images: Select img1 and img2, then prune img2
	m, _ = sendKey(m, "2") // Switch to TabImages
	m, _ = sendKey(m, "space")
	m, _ = sendKey(m, "space")
	assertSelectedImagesCount(t, m, 2)

	updatedImages := updatedContainers
	updatedImages.Images = []monitor.DockerImage{
		testutil.AnImage(testutil.WithImageID("img1", "sha256:img1")),
		testutil.AnImage(testutil.WithImageID("img3", "sha256:img3")),
	}
	updated, _ = m.Update(updatedImages)
	m = updated.(tui.Model)
	assertSelectedImagesCount(t, m, 1)

	// 3. Networks: Select net1 and net2, then prune net1
	m, _ = sendKey(m, "4") // Switch to TabNetworks
	m, _ = sendKey(m, "space")
	m, _ = sendKey(m, "space")
	assertSelectedNetworksCount(t, m, 2)

	updatedNetworks := updatedImages
	updatedNetworks.Networks = []monitor.DockerNetwork{
		{ID: "net2", Name: "backend-internal"},
	}
	updated, _ = m.Update(updatedNetworks)
	m = updated.(tui.Model)
	assertSelectedNetworksCount(t, m, 1)

	// 4. Volumes: Select pg_data and redis_cache, then prune pg_data
	m, _ = sendKey(m, "5") // Switch to TabVolumes
	m, _ = sendKey(m, "space")
	m, _ = sendKey(m, "space")
	assertSelectedVolumesCount(t, m, 2)

	updatedVolumes := updatedNetworks
	updatedVolumes.Volumes = []monitor.DockerVolume{
		{Name: "redis_cache"},
	}
	updated, _ = m.Update(updatedVolumes)
	m = updated.(tui.Model)
	assertSelectedVolumesCount(t, m, 1)
}

// TestE2E_FilterZeroResults_KeyHandling verifies that navigation and selection keys
// behave safely without panic when a filter produces zero results.
func TestE2E_FilterZeroResults_KeyHandling(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Type filter query matching nothing
	m = sendFilterAndConfirm(m, "nonexistent_container_pattern_999")
	assertNotContains(t, m.View(), "web-gateway")

	// Send various action and navigation keys on TabDocker - none should panic
	testKeys := []string{"space", "a", "enter", "s", "r", "k", "d", "j", "k", "down", "up"}
	for _, k := range testKeys {
		m, _ = sendKey(m, k)
	}

	// Esc should clear filter and restore all containers
	m, _ = sendKey(m, "esc")
	assertContains(t, m.View(), "web-gateway")
}

// TestE2E_BackgroundRefreshDuringModals verifies that background data polling
// (DataFetchedMsg) does not close or corrupt active overlay dialogs.
func TestE2E_BackgroundRefreshDuringModals(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Help overlay open
	m, _ = sendKey(m, "?")
	assertHelpOpen(t, m)

	// Background data fetch arrives while user is reading Help
	refreshed, _ := m.Update(actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{Status: "running"},
		Timestamp:  "12:02:00",
	})
	m = refreshed.(tui.Model)
	assertHelpOpen(t, m)

	// Close help
	m, _ = sendKey(m, "esc")
	assertHelpClosed(t, m)

	// 2. Inspect Modal open
	m.OpenModal(tui.ModalInspect)
	m.SetInspectForTesting(monitor.ContainerDetail{
		ID:    "c101",
		Name:  "web-gateway",
		Image: "nginx:alpine",
	})
	assertModal(t, m, tui.ModalInspect)

	// Background data fetch arrives
	refreshed, _ = m.Update(actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{Status: "running"},
		Timestamp:  "12:03:00",
	})
	m = refreshed.(tui.Model)
	assertModal(t, m, tui.ModalInspect)

	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}
