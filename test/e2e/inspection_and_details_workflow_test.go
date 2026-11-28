package e2e

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

// TestE2E_ContainerSortModesWorkflow verifies that cycling sort mode with 'o'
// correctly orders containers by CPU, Memory, Name, and Status.
func TestE2E_ContainerSortModesWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Initial default mode
	if m.ContainerSortMode() != tui.SortDefault {
		t.Fatalf("expected initial sort mode SortDefault (0), got %d", m.ContainerSortMode())
	}

	// 1. Press 'o' -> SortCPU (1)
	m, _ = sendKey(m, "o")
	if m.ContainerSortMode() != tui.SortCPU {
		t.Fatalf("expected SortCPU (1), got %d", m.ContainerSortMode())
	}
	cpuList := m.FilteredContainers()
	if len(cpuList) < 2 || cpuList[0].Name != "api-backend" {
		t.Errorf("expected highest CPU container 'api-backend' at top, got %s", cpuList[0].Name)
	}

	// 2. Press 'o' -> SortMem (2)
	m, _ = sendKey(m, "o")
	if m.ContainerSortMode() != tui.SortMem {
		t.Fatalf("expected SortMem (2), got %d", m.ContainerSortMode())
	}
	memList := m.FilteredContainers()
	if len(memList) < 2 || memList[0].Name != "api-backend" {
		t.Errorf("expected highest Memory container 'api-backend' at top, got %s", memList[0].Name)
	}

	// 3. Press 'o' -> SortName (3)
	m, _ = sendKey(m, "o")
	if m.ContainerSortMode() != tui.SortName {
		t.Fatalf("expected SortName (3), got %d", m.ContainerSortMode())
	}
	nameList := m.FilteredContainers()
	if nameList[0].Name != "api-backend" || nameList[1].Name != "redis-cache" || nameList[2].Name != "web-gateway" {
		t.Errorf("expected alphabetical ordering [api-backend, redis-cache, web-gateway], got [%s, %s, %s]",
			nameList[0].Name, nameList[1].Name, nameList[2].Name)
	}

	// 4. Press 'o' -> SortStatus (4)
	m, _ = sendKey(m, "o")
	if m.ContainerSortMode() != tui.SortStatus {
		t.Fatalf("expected SortStatus (4), got %d", m.ContainerSortMode())
	}
	statusList := m.FilteredContainers()
	if statusList[2].State != "exited" {
		t.Errorf("expected exited container 'redis-cache' at the bottom in SortStatus, got %s (state: %s)",
			statusList[2].Name, statusList[2].State)
	}

	// 5. Press 'o' -> cycles back to SortDefault (0)
	m, _ = sendKey(m, "o")
	if m.ContainerSortMode() != tui.SortDefault {
		t.Errorf("expected sort mode to cycle back to SortDefault (0), got %d", m.ContainerSortMode())
	}
}

// TestE2E_NetworkAndVolumeInspectionWorkflow verifies opening inspect modals
// for Networks and Volumes with JSON formatting and removal confirmation prompts.
func TestE2E_NetworkAndVolumeInspectionWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Networks Tab (4) Inspect and Remove Modal
	m, _ = sendKey(m, "4")
	assertTab(t, m, tui.TabNetworks)

	// Open Network Inspect with Enter
	m, _ = sendKey(m, "enter")
	assertModal(t, m, tui.ModalInspect)

	inspectView := m.View()
	assertContains(t, inspectView, "NETWORK INSPECTION")
	assertContains(t, inspectView, "frontend-bridge")

	// Test scrolling inspect modal
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "up")
	m, _ = sendKey(m, "pgdown")
	m, _ = sendKey(m, "pgup")

	// Close inspect modal
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// Open Network Remove modal with 'd'
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalRemoveNetwork)
	assertContains(t, m.View(), "frontend-bridge")

	// Cancel remove modal
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// 2. Volumes Tab (5) Inspect and Remove Modal
	m, _ = sendKey(m, "5")
	assertTab(t, m, tui.TabVolumes)

	// Open Volume Inspect with 'i'
	m, _ = sendKey(m, "i")
	assertModal(t, m, tui.ModalInspect)

	volInspectView := m.View()
	assertContains(t, volInspectView, "VOLUME INSPECTION")
	assertContains(t, volInspectView, "pg_data")

	// Close volume inspect
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)

	// Open Volume Remove modal with 'd'
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalRemoveVolume)
	assertContains(t, m.View(), "pg_data")

	// Cancel volume removal
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}

// TestE2E_ImagesFilterAndInUseWarningWorkflow verifies unused image toggling
// and in-use safety warnings when deleting Docker images.
func TestE2E_ImagesFilterAndInUseWarningWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Update images so img1 is in-use, while img2 and img3 are unused
	updatedMsg := actions.DataFetchedMsg{
		Images: []monitor.DockerImage{
			testutil.AnImage(testutil.WithImageID("img1", "sha256:img1"), testutil.WithRepoTag("nginx", "alpine"), testutil.WithImageStatus(true, false, 1)),
			testutil.AnImage(testutil.WithImageID("img2", "sha256:img2"), testutil.WithRepoTag("golang", "1.24"), testutil.WithImageStatus(false, false, 0)),
			testutil.AnImage(testutil.WithImageID("img3", "sha256:img3"), testutil.WithRepoTag("redis", "7-alpine"), testutil.WithImageStatus(false, false, 0)),
		},
		Timestamp: "12:10:00",
	}
	updated, _ := m.Update(updatedMsg)
	m = updated.(tui.Model)

	// Switch to TabImages (2)
	m, _ = sendKey(m, "2")
	assertTab(t, m, tui.TabImages)

	// 1. Toggle unused images with 'u'
	m, _ = sendKey(m, "u")
	if !m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages = true after pressing 'u'")
	}
	unusedImages := m.FilteredImages()
	if len(unusedImages) != 2 {
		t.Fatalf("expected 2 unused images, got %d", len(unusedImages))
	}
	for _, img := range unusedImages {
		if img.ID == "img1" {
			t.Errorf("in-use image img1 should be filtered out when unused-only is enabled")
		}
	}

	// 2. Toggle back to all images
	m, _ = sendKey(m, "u")
	if m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages = false after pressing 'u' again")
	}
	if len(m.FilteredImages()) != 3 {
		t.Fatalf("expected all 3 images visible, got %d", len(m.FilteredImages()))
	}

	// 3. Delete in-use image img1 (cursor is at 0: img1)
	m, _ = sendKey(m, "d")
	assertModal(t, m, tui.ModalRemoveImage)
	assertContains(t, m.View(), "nginx:alpine")

	// Cancel with Esc
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}

// TestE2E_ContainerPortURLExtraction verifies opening and copying container URLs
// with 'w', including handling running vs stopped containers.
func TestE2E_ContainerPortURLExtraction(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Update container c101 to have realistic port mappings
	c101 := testutil.AContainer(
		testutil.WithID("c101"),
		testutil.WithName("web-gateway"),
		testutil.WithState("running", "Up 1 hour"),
		testutil.WithPorts("0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp"),
	)
	c102 := testutil.AContainer(
		testutil.WithID("c102"),
		testutil.WithName("api-backend"),
		testutil.WithState("running", "Up 1 hour"),
		testutil.WithPorts(""), // no mapped ports
	)
	c103 := testutil.AContainer(
		testutil.WithID("c103"),
		testutil.WithName("redis-cache"),
		testutil.WithState("exited", "Exited (0)"),
		testutil.WithPorts(""),
	)

	updatedMsg := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Containers: []monitor.ContainerItem{c101, c102, c103},
		},
		Timestamp: "12:15:00",
	}
	updated, _ := m.Update(updatedMsg)
	m = updated.(tui.Model)

	// 1. Running container with mapped ports (c101 at cursor 0): press 'w'
	m, _ = sendKey(m, "w")
	view := m.View()
	// Should show toast about opening / copying URL
	if !strings.Contains(view, "8080") && !strings.Contains(view, "Copied") && !strings.Contains(view, "Opened") {
		t.Errorf("expected toast mentioning URL extraction on 'w', got:\n%s", view)
	}

	// 2. Running container without mapped ports (c102 at cursor 1): press 'w'
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "w")
	view = m.View()
	if !strings.Contains(view, "No mapped ports") {
		t.Errorf("expected toast 'No mapped ports', got:\n%s", view)
	}

	// 3. Stopped container (c103 at cursor 2): press 'w'
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "w")
	view = m.View()
	if !strings.Contains(view, "is not running") {
		t.Errorf("expected toast 'is not running', got:\n%s", view)
	}
}
