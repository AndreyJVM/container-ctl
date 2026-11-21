package tui

import (
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNavigationTabAndCyclicMovement(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.19.0")
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("c1")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("c2")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("c3")),
	}

	// Initial tab is TabDocker
	if m.activeTab != TabDocker {
		t.Fatalf("expected TabDocker, got %v", m.activeTab)
	}

	// Move cursor down cyclically
	m.moveCursor(1)
	if m.cursorIdx != 1 {
		t.Errorf("expected cursorIdx=1, got %d", m.cursorIdx)
	}
	m.moveCursor(1)
	if m.cursorIdx != 2 {
		t.Errorf("expected cursorIdx=2, got %d", m.cursorIdx)
	}
	m.moveCursor(1)
	if m.cursorIdx != 0 {
		t.Errorf("expected cursorIdx wrapped to 0, got %d", m.cursorIdx)
	}
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

	// Tabs cycle forward and backward
	m.SetShowSwarm(false)
	m.swarm.IsActive = false
	if m.IsSwarmVisible() {
		t.Errorf("expected swarm not visible")
	}
	if m.maxTab() != TabVolumes {
		t.Errorf("expected maxTab to be TabVolumes (%d), got %d", TabVolumes, m.maxTab())
	}

	// Next tab cycling
	m.setTab(TabVolumes)
	m.nextTab()
	if m.activeTab != TabDocker {
		t.Errorf("expected wrap to TabDocker, got %d", m.activeTab)
	}
	m.prevTab()
	if m.activeTab != TabVolumes {
		t.Errorf("expected wrap to TabVolumes, got %d", m.activeTab)
	}

	// Show swarm
	m.SetShowSwarm(true)
	if !m.IsSwarmVisible() {
		t.Errorf("expected swarm visible when showSwarm=true")
	}
	if m.maxTab() != TabSwarm {
		t.Errorf("expected maxTab to be TabSwarm (%d), got %d", TabSwarm, m.maxTab())
	}
}

func TestNavigationAllTabsCursor(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.19.0")
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

	// TabImages
	m.setTab(TabImages)
	m.moveCursor(1)
	if m.imageCursorIdx != 1 {
		t.Errorf("expected imageCursorIdx=1, got %d", m.imageCursorIdx)
	}
	m.cursorToStart()
	if m.imageCursorIdx != 0 {
		t.Errorf("expected imageCursorIdx=0, got %d", m.imageCursorIdx)
	}
	m.cursorToEnd()
	if m.imageCursorIdx != 1 {
		t.Errorf("expected imageCursorIdx=1, got %d", m.imageCursorIdx)
	}

	// TabCompose
	m.setTab(TabCompose)
	m.moveCursor(1)
	if m.composeCursorIdx != 1 {
		t.Errorf("expected composeCursorIdx=1, got %d", m.composeCursorIdx)
	}
	m.cursorToStart()
	if m.composeCursorIdx != 0 {
		t.Errorf("expected composeCursorIdx=0, got %d", m.composeCursorIdx)
	}
	m.cursorToEnd()
	if m.composeCursorIdx != 1 {
		t.Errorf("expected composeCursorIdx=1, got %d", m.composeCursorIdx)
	}

	// TabNetworks
	m.setTab(TabNetworks)
	m.moveCursor(1)
	if m.networkCursorIdx != 1 {
		t.Errorf("expected networkCursorIdx=1, got %d", m.networkCursorIdx)
	}
	m.cursorToStart()
	if m.networkCursorIdx != 0 {
		t.Errorf("expected networkCursorIdx=0, got %d", m.networkCursorIdx)
	}
	m.cursorToEnd()
	if m.networkCursorIdx != 1 {
		t.Errorf("expected networkCursorIdx=1, got %d", m.networkCursorIdx)
	}

	// TabVolumes
	m.setTab(TabVolumes)
	m.moveCursor(1)
	if m.volumeCursorIdx != 1 {
		t.Errorf("expected volumeCursorIdx=1, got %d", m.volumeCursorIdx)
	}
	m.cursorToStart()
	if m.volumeCursorIdx != 0 {
		t.Errorf("expected volumeCursorIdx=0, got %d", m.volumeCursorIdx)
	}
	m.cursorToEnd()
	if m.volumeCursorIdx != 1 {
		t.Errorf("expected volumeCursorIdx=1, got %d", m.volumeCursorIdx)
	}

	// TabSwarm
	m.setTab(TabSwarm)
	m.moveCursor(1)
	if m.swarmCursorIdx != 1 {
		t.Errorf("expected swarmCursorIdx=1, got %d", m.swarmCursorIdx)
	}
	m.cursorToStart()
	if m.swarmCursorIdx != 0 {
		t.Errorf("expected swarmCursorIdx=0, got %d", m.swarmCursorIdx)
	}
	m.cursorToEnd()
	if m.swarmCursorIdx != 1 {
		t.Errorf("expected swarmCursorIdx=1, got %d", m.swarmCursorIdx)
	}

	// clampCursor when lists shrink or empty
	m.cursorIdx = 10
	m.imageCursorIdx = 10
	m.composeCursorIdx = 10
	m.networkCursorIdx = 10
	m.volumeCursorIdx = 10
	m.swarmCursorIdx = 10
	m.clampCursor()
	if m.cursorIdx != 0 || m.imageCursorIdx != 1 || m.composeCursorIdx != 1 ||
		m.networkCursorIdx != 1 || m.volumeCursorIdx != 1 || m.swarmCursorIdx != 1 {
		t.Errorf("clampCursor failed to clamp out-of-bound indices")
	}

	// Negative index clamping
	m.imageCursorIdx = -5
	m.composeCursorIdx = -5
	m.networkCursorIdx = -5
	m.volumeCursorIdx = -5
	m.swarmCursorIdx = -5
	m.clampCursor()
	if m.imageCursorIdx != 0 || m.composeCursorIdx != 0 ||
		m.networkCursorIdx != 0 || m.volumeCursorIdx != 0 || m.swarmCursorIdx != 0 {
		t.Errorf("clampCursor failed to clamp negative indices")
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

