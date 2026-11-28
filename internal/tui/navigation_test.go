package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/module"
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

func TestDynamicModuleNavigation_ReorderingAndDisabling(t *testing.T) {
	t.Parallel()

	// 1. Test TabIDFromModuleID and ModuleIDFromTabID mapping
	tabMappings := []struct {
		tab TabID
		mod module.ID
	}{
		{TabDocker, module.IDContainers},
		{TabImages, module.IDImages},
		{TabCompose, module.IDCompose},
		{TabNetworks, module.IDNetworks},
		{TabVolumes, module.IDVolumes},
		{TabSwarm, module.IDSwarm},
	}
	for _, m := range tabMappings {
		modID, ok := ModuleIDFromTabID(m.tab)
		if !ok || modID != m.mod {
			t.Errorf("ModuleIDFromTabID(%v) = (%v, %v), want (%v, true)", m.tab, modID, ok, m.mod)
		}
		tabID, ok := TabIDFromModuleID(m.mod)
		if !ok || tabID != m.tab {
			t.Errorf("TabIDFromModuleID(%v) = (%v, %v), want (%v, true)", m.mod, tabID, ok, m.tab)
		}
	}
	if _, ok := TabIDFromModuleID(module.ID("unknown")); ok {
		t.Errorf("expected false for unknown module ID")
	}
	if _, ok := ModuleIDFromTabID(TabID(999)); ok {
		t.Errorf("expected false for invalid TabID")
	}

	// 2. Custom registry: disable images, order Swarm first (order 1), Containers second (order 2)
	m := newFastTestModel()
	r := module.NewRegistry()
	r.Register(module.ContainersModule)
	r.Register(module.ImagesModule)
	r.Register(module.SwarmModule)

	disabled := false
	r.SetModuleConfig(module.IDImages, module.Config{Enabled: &disabled})
	r.SetModuleConfig(module.IDSwarm, module.Config{Order: 1})
	r.SetModuleConfig(module.IDContainers, module.Config{Order: 2})

	m.SetRegistry(r)
	m.swarm.IsActive = true // make swarm active so it is visible

	visible := m.visibleModules()
	if len(visible) != 2 {
		t.Fatalf("expected exactly 2 visible modules, got %d", len(visible))
	}
	if visible[0].ID != module.IDSwarm {
		t.Errorf("expected first visible module to be Swarm, got %v", visible[0].ID)
	}
	if visible[1].ID != module.IDContainers {
		t.Errorf("expected second visible module to be Containers, got %v", visible[1].ID)
	}

	// 3. Test setTabByIndex
	if !m.setTabByIndex(1) {
		t.Fatalf("expected setTabByIndex(1) to succeed")
	}
	if m.activeTab != TabSwarm {
		t.Errorf("expected activeTab to be TabSwarm after setTabByIndex(1), got %v", m.activeTab)
	}

	if !m.setTabByIndex(2) {
		t.Fatalf("expected setTabByIndex(2) to succeed")
	}
	if m.activeTab != TabDocker {
		t.Errorf("expected activeTab to be TabDocker after setTabByIndex(2), got %v", m.activeTab)
	}

	// Out of bounds index
	if m.setTabByIndex(3) {
		t.Errorf("expected setTabByIndex(3) to fail when only 2 modules visible")
	}

	// 4. Test nextTab and prevTab cycling with custom ordering
	m.setTab(TabSwarm)
	m.nextTab()
	if m.activeTab != TabDocker {
		t.Errorf("expected nextTab from Swarm to go to Containers, got %v", m.activeTab)
	}
	m.nextTab()
	if m.activeTab != TabSwarm {
		t.Errorf("expected nextTab from Containers to wrap back to Swarm, got %v", m.activeTab)
	}

	m.prevTab()
	if m.activeTab != TabDocker {
		t.Errorf("expected prevTab from Swarm to wrap back to Containers, got %v", m.activeTab)
	}

	// 5. Test numeric key presses via Update
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = res.(Model)
	if m.activeTab != TabSwarm {
		t.Errorf("expected pressing '1' to switch to first visible module (Swarm), got %v", m.activeTab)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = res.(Model)
	if m.activeTab != TabDocker {
		t.Errorf("expected pressing '2' to switch to second visible module (Containers), got %v", m.activeTab)
	}
}

func TestNavigation_EmptyVisibleAndActiveNotInVisible(t *testing.T) {
	t.Parallel()

	// 1. Empty visible modules (all modules disabled or empty registry)
	m := newFastTestModel()
	emptyReg := module.NewRegistry()
	m.SetRegistry(emptyReg)

	m.activeTab = TabVolumes
	m.nextTab()
	if m.activeTab != TabDocker {
		t.Errorf("expected nextTab with empty visible modules to wrap around to TabDocker, got %v", m.activeTab)
	}

	m.activeTab = TabDocker
	m.prevTab()
	if m.activeTab != TabVolumes {
		t.Errorf("expected prevTab with empty visible modules to wrap around to TabVolumes, got %v", m.activeTab)
	}

	// 2. Active tab not in visible list (currIdx == -1 fallback)
	regWithOnlyImages := module.NewRegistry()
	regWithOnlyImages.Register(module.ImagesModule)
	m.SetRegistry(regWithOnlyImages)

	m.activeTab = TabDocker // not visible in registry
	m.nextTab()
	if m.activeTab != TabImages {
		t.Errorf("expected nextTab when current tab not visible to jump to first visible (TabImages), got %v", m.activeTab)
	}

	m.activeTab = TabDocker // not visible in registry
	m.prevTab()
	if m.activeTab != TabImages {
		t.Errorf("expected prevTab when current tab not visible to jump to last visible (TabImages), got %v", m.activeTab)
	}
}

// =============================================================================
// Filter tests merged from filter_test.go
// =============================================================================

func TestFilterContainers(t *testing.T) {
	t.Parallel()

	containers := []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web-gateway"), testutil.WithImage("nginx:alpine"), testutil.WithCompose("infra", "")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db-postgres"), testutil.WithImage("postgres:16"), testutil.WithCompose("backend", "")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("redis-cache"), testutil.WithImage("redis:7-alpine"), testutil.WithCompose("cache", "")),
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "empty query returns all containers",
			query:         "",
			expectedCount: 3,
			expectedNames: []string{"web-gateway", "db-postgres", "redis-cache"},
		},
		{
			name:          "whitespace only query returns all containers",
			query:         "   ",
			expectedCount: 3,
			expectedNames: []string{"web-gateway", "db-postgres", "redis-cache"},
		},
		{
			name:          "filter by container name",
			query:         "web",
			expectedCount: 1,
			expectedNames: []string{"web-gateway"},
		},
		{
			name:          "case-insensitive match on image",
			query:         "POSTGRES",
			expectedCount: 1,
			expectedNames: []string{"db-postgres"},
		},
		{
			name:          "match by container ID",
			query:         "c3",
			expectedCount: 1,
			expectedNames: []string{"redis-cache"},
		},
		{
			name:          "match by compose project name",
			query:         "infra",
			expectedCount: 1,
			expectedNames: []string{"web-gateway"},
		},
		{
			name:          "no match returns empty slice",
			query:         "non-existent-service",
			expectedCount: 0,
			expectedNames: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newFastTestModel()
			m.dockerInfo.Containers = containers
			m.filterQuery = tc.query

			filtered := m.getFilteredContainers()
			if len(filtered) != tc.expectedCount {
				t.Fatalf("expected %d containers, got %d", tc.expectedCount, len(filtered))
			}
			for i, name := range tc.expectedNames {
				if filtered[i].Name != name {
					t.Errorf("expected container %d to be %q, got %q", i, name, filtered[i].Name)
				}
			}
		})
	}
}

func TestFilterContainersByHealth(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web"), testutil.WithImage("nginx"), testutil.WithHealth("healthy")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db"), testutil.WithImage("postgres"), testutil.WithHealth("unhealthy")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("cache"), testutil.WithImage("redis"), testutil.WithHealth("starting")),
	}

	t.Run("filter by unhealthy", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.filterQuery = "unhealthy"
		filtered := subM.getFilteredContainers()
		if len(filtered) != 1 || filtered[0].Name != "db" {
			t.Errorf("expected 1 result matching 'unhealthy' (db), got %v", filtered)
		}
	})

	t.Run("filter by healthy matches substring in unhealthy too", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.filterQuery = "healthy"
		filtered := subM.getFilteredContainers()
		if len(filtered) != 2 {
			t.Errorf("expected 2 results matching 'healthy', got %d", len(filtered))
		}
	})

	t.Run("filter by starting", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.filterQuery = "starting"
		filtered := subM.getFilteredContainers()
		if len(filtered) != 1 || filtered[0].Name != "cache" {
			t.Errorf("expected 1 result matching 'starting', got %v", filtered)
		}
	})
}

func TestFilterModeKeys(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web server"), testutil.WithImage("nginx")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("worker"), testutil.WithImage("app")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("database"), testutil.WithImage("postgres")),
	}
	m.cursorIdx = 2 // Point to last container

	// 1. Activate search mode with '/'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering=true after '/'")
	}

	// 2. Type "web" with Space and "server"
	for _, r := range []rune("web server") {
		if r == ' ' {
			res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
		} else {
			res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		m = res.(Model)
	}
	if m.filterQuery != "web server" {
		t.Errorf("expected filterQuery='web server', got %q", m.filterQuery)
	}
	if len(m.getFilteredContainers()) != 1 {
		t.Errorf("expected 1 container matching 'web server', got %d", len(m.getFilteredContainers()))
	}
	// Cursor must clamp to 0 since only 1 container matches
	if m.cursorIdx != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.cursorIdx)
	}

	// 3. Backspace deletes character
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.filterQuery != "web serve" {
		t.Errorf("expected filterQuery='web serve' after backspace, got %q", m.filterQuery)
	}

	// 4. Enter applies filter and exits typing mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Errorf("expected isFiltering=false after Enter")
	}
	if m.filterQuery != "web serve" {
		t.Errorf("expected filterQuery to persist after Enter, got %q", m.filterQuery)
	}

	// 5. Esc in normal mode clears filter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery='' after Esc, got %q", m.filterQuery)
	}

	// 6. Backspace on empty filterQuery does not panic
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery to remain empty after backspace on empty string")
	}

	// 7. Esc during active typing clears query and exits filtering
	m.isFiltering = true
	m.filterQuery = "search"
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.isFiltering || m.filterQuery != "" {
		t.Errorf("expected isFiltering=false and filterQuery='' after Esc during typing")
	}
}

func TestFilterModelDelegation(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()

	m.images = []monitor.DockerImage{
		{ID: "img1", Repository: "nginx", Tag: "latest", InUse: true},
		{ID: "img2", Repository: "redis", Tag: "alpine", InUse: false},
		{ID: "img3", Repository: "<none>", Tag: "<none>", InUse: false, Dangling: true},
	}
	m.compose = []monitor.ComposeProject{
		{Name: "app-prod"},
		{Name: "app-dev"},
	}
	m.networks = []monitor.DockerNetwork{
		{ID: "n1", Name: "bridge-net", Driver: "bridge"},
		{ID: "n2", Name: "custom-net", Driver: "overlay"},
	}
	m.volumes = []monitor.DockerVolume{
		{Name: "db-data", Driver: "local"},
		{Name: "cache-data", Driver: "local"},
	}

	// Compose delegation
	m.filterQuery = "prod"
	if len(m.getFilteredCompose()) != 1 || m.getFilteredCompose()[0].Name != "app-prod" {
		t.Errorf("expected 1 compose project matching 'prod'")
	}

	// Networks delegation
	m.filterQuery = "overlay"
	if len(m.getFilteredNetworks()) != 1 || m.getFilteredNetworks()[0].Name != "custom-net" {
		t.Errorf("expected 1 network matching 'overlay'")
	}

	// Volumes delegation
	m.filterQuery = "cache"
	if len(m.getFilteredVolumes()) != 1 || m.getFilteredVolumes()[0].Name != "cache-data" {
		t.Errorf("expected 1 volume matching 'cache'")
	}

	// Images delegation with query and onlyUnused
	m.filterQuery = "redis"
	if len(m.getFilteredImages()) != 1 || m.getFilteredImages()[0].Repository != "redis" {
		t.Errorf("expected 1 image matching 'redis'")
	}

	m.filterQuery = ""
	m.SetOnlyUnusedImages(true)
	if !m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be true")
	}
	if len(m.getFilteredImages()) != 2 {
		t.Errorf("expected 2 unused images (redis and dangling), got %d", len(m.getFilteredImages()))
	}

	m.SetOnlyUnusedImages(false)
	if m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be false")
	}
	if len(m.getFilteredImages()) != 3 {
		t.Errorf("expected all 3 images when onlyUnused is false")
	}

	// Sort mode setters and boundary checks
	m.SetContainerSortMode(SortName)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode SortName, got %d", m.containerSortMode)
	}
	// Invalid modes out of range should be ignored
	m.SetContainerSortMode(999)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode to remain SortName when 999 passed, got %d", m.containerSortMode)
	}
	m.SetContainerSortMode(-1)
	if m.containerSortMode != SortName {
		t.Errorf("expected sort mode to remain SortName when -1 passed, got %d", m.containerSortMode)
	}

	// Container status score evaluation
	runningHealthy := containerStatusScore(monitor.ContainerItem{State: "running", Health: "healthy"})
	runningUnhealthy := containerStatusScore(monitor.ContainerItem{State: "running", Health: "unhealthy"})
	if runningUnhealthy >= runningHealthy {
		t.Errorf("expected unhealthy containers to have higher priority (lower score) than healthy")
	}
}

func TestContainerSortingViaModel(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithName("beta"), testutil.WithMetrics(10.0, 500.0, 1024.0)),
		testutil.AContainer(testutil.WithName("alpha"), testutil.WithMetrics(85.0, 100.0, 1024.0)),
		testutil.AContainer(testutil.WithName("gamma"), testutil.WithMetrics(45.0, 900.0, 1024.0)),
	}

	t.Run("sort by CPU descending", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.SetContainerSortMode(SortCPU)
		sorted := subM.getFilteredContainers()
		if sorted[0].Name != "alpha" || sorted[1].Name != "gamma" || sorted[2].Name != "beta" {
			t.Errorf("unexpected CPU sort order: %v, %v, %v", sorted[0].Name, sorted[1].Name, sorted[2].Name)
		}
	})

	t.Run("sort by memory descending", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.SetContainerSortMode(SortMem)
		sorted := subM.getFilteredContainers()
		if sorted[0].Name != "gamma" || sorted[1].Name != "beta" || sorted[2].Name != "alpha" {
			t.Errorf("unexpected memory sort order: %v, %v, %v", sorted[0].Name, sorted[1].Name, sorted[2].Name)
		}
	})

	t.Run("sort by name ascending", func(t *testing.T) {
		t.Parallel()
		subM := m
		subM.SetContainerSortMode(SortName)
		sorted := subM.getFilteredContainers()
		if sorted[0].Name != "alpha" || sorted[1].Name != "beta" || sorted[2].Name != "gamma" {
			t.Errorf("unexpected name sort order: %v, %v, %v", sorted[0].Name, sorted[1].Name, sorted[2].Name)
		}
	})
}

func TestComposeNetworksVolumesSearchFilter(t *testing.T) {
	t.Parallel()

	var m Model
	m.version = "v0.18.1"
	m.activeTab = TabCompose
	m.refreshInterval = 2 * time.Second
	m.compose = []monitor.ComposeProject{
		{Name: "web-stack", Status: "running", Services: []monitor.ComposeService{{Name: "web-srv", Image: "nginx"}}},
		{Name: "db-stack", Status: "running", Services: []monitor.ComposeService{{Name: "db-srv", Image: "postgres"}}},
	}
	m.networks = []monitor.DockerNetwork{
		{ID: "n1", Name: "bridge-net", Driver: "bridge", Subnet: "172.18.0.0/16"},
		{ID: "n2", Name: "custom-net", Driver: "overlay", Subnet: "10.0.0.0/24"},
	}
	m.volumes = []monitor.DockerVolume{
		{Name: "db-data", Driver: "local"},
		{Name: "cache-data", Driver: "local"},
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

func TestNavigation_ExportedAccessors(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("c1")),
	}
	m.images = []monitor.DockerImage{
		{ID: "img1", Repository: "alpine", Tag: "latest"},
	}
	m.compose = []monitor.ComposeProject{
		{Name: "p1", Status: "running"},
	}
	m.networks = []monitor.DockerNetwork{
		{ID: "n1", Name: "net1"},
	}
	m.volumes = []monitor.DockerVolume{
		{Name: "vol1"},
	}

	// 1. Cursor accessors
	m.activeTab = TabDocker
	m.cursorIdx = 0
	if m.ActiveCursor() != 0 {
		t.Errorf("expected ActiveCursor 0, got %d", m.ActiveCursor())
	}

	// 2. Filtered list accessors
	if len(m.FilteredContainers()) != 1 {
		t.Errorf("expected 1 container, got %d", len(m.FilteredContainers()))
	}
	if len(m.FilteredImages()) != 1 {
		t.Errorf("expected 1 image, got %d", len(m.FilteredImages()))
	}
	if len(m.FilteredCompose()) != 1 {
		t.Errorf("expected 1 compose project, got %d", len(m.FilteredCompose()))
	}
	if len(m.FilteredNetworks()) != 1 {
		t.Errorf("expected 1 network, got %d", len(m.FilteredNetworks()))
	}
	if len(m.FilteredVolumes()) != 1 {
		t.Errorf("expected 1 volume, got %d", len(m.FilteredVolumes()))
	}

	// 3. Filter query and status
	m.filterQuery = "test-query"
	m.isFiltering = true
	if m.FilterQuery() != "test-query" {
		t.Errorf("expected FilterQuery 'test-query', got %q", m.FilterQuery())
	}
	if !m.IsFiltering() {
		t.Errorf("expected IsFiltering true")
	}

	// 4. Sort mode
	m.containerSortMode = SortCPU
	if m.ContainerSortMode() != SortCPU {
		t.Errorf("expected ContainerSortMode SortCPU, got %d", m.ContainerSortMode())
	}

	// 5. ToastMessage
	m.actionMessage = "Operation completed"
	if m.ToastMessage() != "Operation completed" {
		t.Errorf("expected ToastMessage 'Operation completed', got %q", m.ToastMessage())
	}

	// 6. Shell state accessors
	if m.IsShellActive() {
		t.Errorf("expected IsShellActive false initially")
	}
	if m.IsShellFocused() {
		t.Errorf("expected IsShellFocused false initially")
	}
	if m.ShellContainerName() != "" {
		t.Errorf("expected ShellContainerName empty initially, got %q", m.ShellContainerName())
	}

	m.embeddedShell = &fakeEmbeddedShell{}
	m.shellFocused = true
	m.shellContainerName = "api-service"

	if !m.IsShellActive() {
		t.Errorf("expected IsShellActive true")
	}
	if !m.IsShellFocused() {
		t.Errorf("expected IsShellFocused true")
	}
	if m.ShellContainerName() != "api-service" {
		t.Errorf("expected ShellContainerName 'api-service', got %q", m.ShellContainerName())
	}
}
