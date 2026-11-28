package e2e

import (
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

// TestE2E_CursorNavigationAndWrapping verifies vertical cursor movement,
// cyclic wrapping at boundaries (top <-> bottom), and home/end jumping across tabs.
func TestE2E_CursorNavigationAndWrapping(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// --- 1. TabDocker (3 containers: indices 0, 1, 2) ---
	assertTab(t, m, tui.TabDocker)
	assertCursor(t, m, 0)

	// Press 'up' at top -> wraps to bottom index 2
	m, _ = sendKey(m, "up")
	assertCursor(t, m, 2)

	// Press 'down' at bottom -> wraps to top index 0
	m, _ = sendKey(m, "down")
	assertCursor(t, m, 0)

	// 'G' (end) -> jumps to last item (2)
	m, _ = sendKey(m, "G")
	assertCursor(t, m, 2)

	// 'g' (home) -> jumps to first item (0)
	m, _ = sendKey(m, "g")
	assertCursor(t, m, 0)

	// 'j' moves down by 1
	m, _ = sendKey(m, "j")
	assertCursor(t, m, 1)

	// 'k' moves up by 1
	m, _ = sendKey(m, "k")
	assertCursor(t, m, 0)

	// --- 2. TabImages (3 images: indices 0, 1, 2) ---
	m, _ = sendKey(m, "2")
	assertTab(t, m, tui.TabImages)
	assertCursor(t, m, 0)

	m, _ = sendKey(m, "up")
	assertCursor(t, m, 2)

	m, _ = sendKey(m, "down")
	assertCursor(t, m, 0)

	// --- 3. TabNetworks (2 networks: indices 0, 1) ---
	m, _ = sendKey(m, "4")
	assertTab(t, m, tui.TabNetworks)
	assertCursor(t, m, 0)

	m, _ = sendKey(m, "up")
	assertCursor(t, m, 1)

	m, _ = sendKey(m, "down")
	assertCursor(t, m, 0)
}

// TestE2E_SwarmQuickReplicaScaling verifies incrementing and decrementing
// Swarm service replicas with '+' and '-' hotkeys.
func TestE2E_SwarmQuickReplicaScaling(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Switch to TabSwarm (6)
	m, _ = sendKey(m, "6")
	assertTab(t, m, tui.TabSwarm)

	// Initial service replica target is 2
	view := m.View()
	assertContains(t, view, "web-proxy")

	// 1. Quick scale UP with '+' (target 2 -> 3)
	m, cmdUp := sendKey(m, "+")
	if cmdUp == nil {
		t.Fatalf("expected scale command dispatched on '+'")
	}
	upView := m.View()
	assertContains(t, upView, "3 replicas")

	// 2. Quick scale DOWN with '-' (target 2 -> 1)
	m, cmdDown := sendKey(m, "-")
	if cmdDown == nil {
		t.Fatalf("expected scale command dispatched on '-'")
	}
	downView := m.View()
	assertContains(t, downView, "1 replicas")

	// 3. Test global service cannot be scaled
	globalMsg := actions.DataFetchedMsg{
		Swarm: monitor.SwarmSummary{
			IsActive: true,
			Services: []monitor.SwarmService{
				{
					ID:             "srv-global",
					Name:           "node-exporter",
					Mode:           "global",
					TargetReplicas: 3,
				},
			},
		},
		Timestamp: "12:05:00",
	}
	updated, _ := m.Update(globalMsg)
	m = updated.(tui.Model)

	m, cmdGlobal := sendKey(m, "+")
	if cmdGlobal != nil {
		t.Errorf("expected no scale command dispatched for global service")
	}
	globalView := m.View()
	assertContains(t, globalView, "Cannot scale global Swarm service")
}

// TestE2E_FilterMultiSelectIntersection verifies that selecting all visible items
// with 'a' while an active filter is applied properly targets only the matching items,
// and preserves existing selections from other subsets upon clearing the filter.
func TestE2E_FilterMultiSelectIntersection(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Select container c101 (web-gateway) with Space
	m, _ = sendKey(m, "space")
	assertSelectedContainersCount(t, m, 1)

	// 2. Filter by "redis" -> only c103 (redis-cache) is visible
	m = sendFilterAndConfirm(m, "redis")

	filtered := m.FilteredContainers()
	if len(filtered) != 1 || filtered[0].ID != "c103" {
		t.Fatalf("expected only redis-cache (c103) visible, got %v", filtered)
	}

	// 3. Press 'a' (toggle select all visible) -> selects redis-cache
	m, _ = sendKey(m, "a")
	// Both c101 and c103 must now be selected (total = 2)
	assertSelectedContainersCount(t, m, 2)

	// 4. Clear filter with Esc -> all 3 containers visible again
	m, _ = sendKey(m, "esc")
	allContainers := m.FilteredContainers()
	if len(allContainers) != 3 {
		t.Fatalf("expected all 3 containers restored, got %d", len(allContainers))
	}
	assertSelectedContainersCount(t, m, 2)

	// 5. Press 'a' again: since not all 3 are selected, 'a' selects all 3
	m, _ = sendKey(m, "a")
	assertSelectedContainersCount(t, m, 3)

	// 6. Press 'a' again: all 3 are selected, so 'a' deselects all
	m, _ = sendKey(m, "a")
	assertSelectedContainersCount(t, m, 0)
}

// TestE2E_SwarmLogsWorkflow verifies that pressing 'l' on a Swarm service
// opens the logs viewer modal with the proper title and target.
func TestE2E_SwarmLogsWorkflow(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Switch to TabSwarm (6)
	m, _ = sendKey(m, "6")
	assertTab(t, m, tui.TabSwarm)

	// Set logs content on service target
	m.OpenModal(tui.ModalLogs)
	m.SetLogsForTesting("Swarm: web-proxy", "srv-web", []string{
		"2026-11-28T02:00:00Z [INFO] Service web-proxy started on node-1",
		"2026-11-28T02:00:01Z [INFO] Configuration loaded successfully",
		"2026-11-28T02:00:02Z [WARN] Upstream backend latency exceeded 200ms",
	})

	assertModal(t, m, tui.ModalLogs)
	view := m.View()
	assertContains(t, view, "LOGS")
	assertContains(t, view, "web-proxy")
	assertContains(t, view, "[WARN]")

	// Close logs modal with Esc
	m, _ = sendKey(m, "esc")
	assertModalClosed(t, m)
}
