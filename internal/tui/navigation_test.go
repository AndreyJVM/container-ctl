package tui

import (
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNavigationTabAndCyclicMovement(t *testing.T) {
	t.Parallel()

	t.Run("containers list cyclic cursor movement", func(t *testing.T) {
		m := newFastTestModel()
		m.dockerInfo.Containers = []monitor.ContainerItem{
			testutil.AContainer(testutil.WithID("c1"), testutil.WithName("c1")),
			testutil.AContainer(testutil.WithID("c2"), testutil.WithName("c2")),
			testutil.AContainer(testutil.WithID("c3"), testutil.WithName("c3")),
		}

		if m.activeTab != TabDocker {
			t.Fatalf("expected initial tab TabDocker, got %v", m.activeTab)
		}

		// Step forward
		m.moveCursor(1)
		if m.cursorIdx != 1 {
			t.Errorf("expected cursorIdx=1, got %d", m.cursorIdx)
		}
		m.moveCursor(1)
		if m.cursorIdx != 2 {
			t.Errorf("expected cursorIdx=2, got %d", m.cursorIdx)
		}

		// Cyclic wrap forward (2 -> 0)
		m.moveCursor(1)
		if m.cursorIdx != 0 {
			t.Errorf("expected cursorIdx wrapped to 0, got %d", m.cursorIdx)
		}

		// Cyclic wrap backward (0 -> 2)
		m.moveCursor(-1)
		if m.cursorIdx != 2 {
			t.Errorf("expected cursorIdx wrapped to 2 on -1, got %d", m.cursorIdx)
		}

		// cursorToStart and cursorToEnd
		m.cursorToStart()
		if m.cursorIdx != 0 {
			t.Errorf("expected cursorIdx=0 after cursorToStart, got %d", m.cursorIdx)
		}
		m.cursorToEnd()
		if m.cursorIdx != 2 {
			t.Errorf("expected cursorIdx=2 after cursorToEnd, got %d", m.cursorIdx)
		}
	})

	t.Run("swarm visibility and tab limits", func(t *testing.T) {
		m := newFastTestModel()

		// 1. Swarm inactive and not forced
		m.SetShowSwarm(false)
		m.swarm.IsActive = false
		if m.IsSwarmVisible() {
			t.Errorf("expected swarm not visible when showSwarm=false and swarm.IsActive=false")
		}
		if m.maxTab() != TabVolumes {
			t.Errorf("expected maxTab to be TabVolumes (%d), got %d", TabVolumes, m.maxTab())
		}

		// Next tab cycling with inactive Swarm (wraps TabVolumes -> TabDocker)
		m.setTab(TabVolumes)
		m.nextTab()
		if m.activeTab != TabDocker {
			t.Errorf("expected wrap to TabDocker, got %d", m.activeTab)
		}
		// Prev tab cycling with inactive Swarm (wraps TabDocker -> TabVolumes)
		m.prevTab()
		if m.activeTab != TabVolumes {
			t.Errorf("expected wrap to TabVolumes, got %d", m.activeTab)
		}

		// 2. Swarm active on host
		m.swarm.IsActive = true
		if !m.IsSwarmVisible() {
			t.Errorf("expected swarm visible when swarm.IsActive=true")
		}
		if m.maxTab() != TabSwarm {
			t.Errorf("expected maxTab to be TabSwarm (%d), got %d", TabSwarm, m.maxTab())
		}

		// Next tab cycling with active Swarm (TabVolumes -> TabSwarm -> TabDocker)
		m.setTab(TabVolumes)
		m.nextTab()
		if m.activeTab != TabSwarm {
			t.Errorf("expected nextTab from Volumes to be TabSwarm, got %d", m.activeTab)
		}
		m.nextTab()
		if m.activeTab != TabDocker {
			t.Errorf("expected nextTab from Swarm to wrap to TabDocker, got %d", m.activeTab)
		}

		// Prev tab cycling with active Swarm (TabDocker -> TabSwarm)
		m.prevTab()
		if m.activeTab != TabSwarm {
			t.Errorf("expected prevTab from Docker to wrap to TabSwarm, got %d", m.activeTab)
		}

		// 3. Forced swarm visibility via SetShowSwarm(true)
		m.swarm.IsActive = false
		m.SetShowSwarm(true)
		if !m.IsSwarmVisible() {
			t.Errorf("expected swarm visible when showSwarm=true")
		}
		if m.maxTab() != TabSwarm {
			t.Errorf("expected maxTab TabSwarm, got %d", m.maxTab())
		}
	})

	t.Run("setTab boundary validation", func(t *testing.T) {
		m := newFastTestModel()
		m.setTab(TabImages)
		if m.activeTab != TabImages {
			t.Errorf("expected TabImages, got %d", m.activeTab)
		}

		// Out-of-bounds tab ID should be ignored
		m.setTab(TabID(999))
		if m.activeTab != TabImages {
			t.Errorf("expected activeTab to remain TabImages on out-of-bounds tab ID, got %d", m.activeTab)
		}

		m.setTab(TabID(-1))
		if m.activeTab != TabImages {
			t.Errorf("expected activeTab to remain TabImages on negative tab ID, got %d", m.activeTab)
		}
	})
}

func TestNavigationAllTabsCursor(t *testing.T) {
	t.Parallel()

	initPopulatedModel := func() Model {
		m := newFastTestModel()
		m.images = []monitor.DockerImage{
			testutil.AnImage(testutil.WithImageID("img1", "sha256:img1")),
			testutil.AnImage(testutil.WithImageID("img2", "sha256:img2")),
		}
		m.compose = []monitor.ComposeProject{
			{Name: "proj1"},
			{Name: "proj2"},
		}
		m.networks = []monitor.DockerNetwork{
			testutil.ANetwork(),
			testutil.ANetwork(),
		}
		m.volumes = []monitor.DockerVolume{
			testutil.AVolume(),
			testutil.AVolume(),
		}
		m.swarm.Services = []monitor.SwarmService{
			{ID: "s1"},
			{ID: "s2"},
		}
		return m
	}

	t.Run("per-tab cursor movement and boundaries", func(t *testing.T) {
		m := initPopulatedModel()

		tabTests := []struct {
			name      string
			tab       TabID
			getCursor func(m Model) int
		}{
			{name: "TabImages", tab: TabImages, getCursor: func(m Model) int { return m.imageCursorIdx }},
			{name: "TabCompose", tab: TabCompose, getCursor: func(m Model) int { return m.composeCursorIdx }},
			{name: "TabNetworks", tab: TabNetworks, getCursor: func(m Model) int { return m.networkCursorIdx }},
			{name: "TabVolumes", tab: TabVolumes, getCursor: func(m Model) int { return m.volumeCursorIdx }},
			{name: "TabSwarm", tab: TabSwarm, getCursor: func(m Model) int { return m.swarmCursorIdx }},
		}

		for _, tt := range tabTests {
			m.setTab(tt.tab)
			m.cursorToStart()
			if c := tt.getCursor(m); c != 0 {
				t.Errorf("%s: expected cursor 0 after cursorToStart, got %d", tt.name, c)
			}

			m.moveCursor(1)
			if c := tt.getCursor(m); c != 1 {
				t.Errorf("%s: expected cursor 1 after moveCursor(1), got %d", tt.name, c)
			}

			// Wrap around
			m.moveCursor(1)
			if c := tt.getCursor(m); c != 0 {
				t.Errorf("%s: expected wrap to 0, got %d", tt.name, c)
			}

			m.cursorToEnd()
			if c := tt.getCursor(m); c != 1 {
				t.Errorf("%s: expected cursor 1 after cursorToEnd, got %d", tt.name, c)
			}
		}
	})

	t.Run("clampCursor out-of-bound and negative indices", func(t *testing.T) {
		m := initPopulatedModel()

		// Positive overflow clamp
		m.cursorIdx = 10
		m.imageCursorIdx = 10
		m.composeCursorIdx = 10
		m.networkCursorIdx = 10
		m.volumeCursorIdx = 10
		m.swarmCursorIdx = 10
		m.clampCursor()

		if m.cursorIdx != 0 {
			t.Errorf("expected cursorIdx clamped to 0 (empty containers), got %d", m.cursorIdx)
		}
		if m.imageCursorIdx != 1 || m.composeCursorIdx != 1 ||
			m.networkCursorIdx != 1 || m.volumeCursorIdx != 1 || m.swarmCursorIdx != 1 {
			t.Errorf("clampCursor failed to clamp out-of-bound indices to len-1")
		}

		// Negative index clamp
		m.cursorIdx = -5
		m.imageCursorIdx = -5
		m.composeCursorIdx = -5
		m.networkCursorIdx = -5
		m.volumeCursorIdx = -5
		m.swarmCursorIdx = -5
		m.clampCursor()

		if m.cursorIdx != 0 || m.imageCursorIdx != 0 || m.composeCursorIdx != 0 ||
			m.networkCursorIdx != 0 || m.volumeCursorIdx != 0 || m.swarmCursorIdx != 0 {
			t.Errorf("clampCursor failed to clamp negative indices to 0")
		}
	})

	t.Run("clampCursor with search filter", func(t *testing.T) {
		m := newFastTestModel()
		m.dockerInfo.Containers = []monitor.ContainerItem{
			testutil.AContainer(testutil.WithName("alpha-api")),
			testutil.AContainer(testutil.WithName("beta-db")),
			testutil.AContainer(testutil.WithName("gamma-cache")),
		}
		m.cursorIdx = 2
		m.filterQuery = "alpha"
		m.clampCursor()

		// Filter reduces matching containers to 1 ("alpha-api"), cursor must clamp to 0
		if m.cursorIdx != 0 {
			t.Errorf("expected cursorIdx clamped to 0 for filtered list, got %d", m.cursorIdx)
		}
	})
}

func TestNavigationEmptyListSafety(t *testing.T) {
	t.Parallel()

	tabs := []TabID{TabDocker, TabImages, TabCompose, TabNetworks, TabVolumes, TabSwarm}

	for _, tab := range tabs {
		m := newFastTestModel()
		m.setTab(tab)

		// None of these operations should panic or leave negative cursor
		m.moveCursor(1)
		m.moveCursor(-1)
		m.cursorToStart()
		m.cursorToEnd()
		m.clampCursor()

		if m.cursorIdx < 0 || m.imageCursorIdx < 0 || m.composeCursorIdx < 0 ||
			m.networkCursorIdx < 0 || m.volumeCursorIdx < 0 || m.swarmCursorIdx < 0 {
			t.Errorf("tab %d: unexpected negative cursor on empty list operations", tab)
		}
	}
}

func TestNavigationKeysViaUpdate(t *testing.T) {
	t.Parallel()

	t.Run("tab number hotkeys and cycling", func(t *testing.T) {
		m := newFastTestModel()

		// Jump directly with keys "1" through "5"
		tabKeys := []struct {
			key         string
			expectedTab TabID
		}{
			{key: "2", expectedTab: TabImages},
			{key: "3", expectedTab: TabCompose},
			{key: "4", expectedTab: TabNetworks},
			{key: "5", expectedTab: TabVolumes},
			{key: "1", expectedTab: TabDocker},
		}

		for _, tk := range tabKeys {
			res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tk.key)})
			m = res.(Model)
			if m.activeTab != tk.expectedTab {
				t.Errorf("key %q: expected activeTab=%d, got %d", tk.key, tk.expectedTab, m.activeTab)
			}
		}

		// Key "6" when swarm is inactive shows toast and does not change tab
		m.setTab(TabDocker)
		m.swarm.IsActive = false
		m.SetShowSwarm(false)
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")})
		m = res.(Model)
		if m.activeTab != TabDocker {
			t.Errorf("key '6': expected to remain on TabDocker when swarm inactive, got %d", m.activeTab)
		}
		if m.actionMessage != "Host is not part of an active Swarm cluster." {
			t.Errorf("expected swarm inactive toast message, got %q", m.actionMessage)
		}

		// Key "6" when swarm is active switches to TabSwarm
		m.swarm.IsActive = true
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("6")})
		m = res.(Model)
		if m.activeTab != TabSwarm {
			t.Errorf("key '6': expected activeTab=TabSwarm when swarm active, got %d", m.activeTab)
		}

		// Tab cycling keys: tab / right and shift+tab / left
		m.setTab(TabDocker)
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = res.(Model)
		if m.activeTab != TabImages {
			t.Errorf("expected Tab after Docker to switch to Images, got %d", m.activeTab)
		}

		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
		m = res.(Model)
		if m.activeTab != TabCompose {
			t.Errorf("expected Right after Images to switch to Compose, got %d", m.activeTab)
		}

		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
		m = res.(Model)
		if m.activeTab != TabImages {
			t.Errorf("expected ShiftTab to go back to Images, got %d", m.activeTab)
		}

		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
		m = res.(Model)
		if m.activeTab != TabDocker {
			t.Errorf("expected Left to go back to Docker, got %d", m.activeTab)
		}
	})

	t.Run("home end pgup pgdown and vim keys", func(t *testing.T) {
		m := newFastTestModel()
		m.dockerInfo.Containers = []monitor.ContainerItem{
			testutil.AContainer(testutil.WithName("c0")),
			testutil.AContainer(testutil.WithName("c1")),
			testutil.AContainer(testutil.WithName("c2")),
			testutil.AContainer(testutil.WithName("c3")),
			testutil.AContainer(testutil.WithName("c4")),
			testutil.AContainer(testutil.WithName("c5")),
			testutil.AContainer(testutil.WithName("c6")),
			testutil.AContainer(testutil.WithName("c7")),
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

		// 7. Vim navigation keys j/k
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = res.(Model)
		if m.cursorIdx != 1 {
			t.Errorf("expected cursor at 1 after 'j', got %d", m.cursorIdx)
		}

		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = res.(Model)
		if m.cursorIdx != 0 {
			t.Errorf("expected cursor at 0 after 'k', got %d", m.cursorIdx)
		}

		// 8. While filtering, "g" and "G" do NOT trigger cursorToStart / cursorToEnd
		m.isFiltering = true
		m.cursorIdx = 3
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		m = res.(Model)
		if m.cursorIdx != 3 {
			t.Errorf("expected cursor to remain at 3 when filtering and pressing 'g', got %d", m.cursorIdx)
		}
	})
}
