package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleFilterInput(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v1.0.0")
	m.isFiltering = true
	m.filterQuery = "alp"

	// 1. Backspace removes last rune
	res, _ := m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.filterQuery != "al" {
		t.Errorf("expected filterQuery 'al' after backspace, got %q", m.filterQuery)
	}

	// 2. Typing appends rune
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = res.(Model)
	if m.filterQuery != "alp" {
		t.Errorf("expected filterQuery 'alp', got %q", m.filterQuery)
	}

	// Typing space
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m = res.(Model)
	if m.filterQuery != "alp " {
		t.Errorf("expected filterQuery 'alp ', got %q", m.filterQuery)
	}

	// 3. Enter exits typing mode and preserves query
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Errorf("expected isFiltering=false after enter")
	}
	if m.filterQuery != "alp " {
		t.Errorf("expected filterQuery preserved, got %q", m.filterQuery)
	}

	// 4. Esc cancels filter and clears query
	m.isFiltering = true
	res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.isFiltering {
		t.Errorf("expected isFiltering=false after esc")
	}
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery='' after esc, got %q", m.filterQuery)
	}
}

func TestHandleMainKeys_NavigationAndTabs(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v1.0.0")

	// 1. Help toggle
	res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = res.(Model)
	if !m.showHelp {
		t.Errorf("expected showHelp=true after '?'")
	}

	// 2. Tab switching with numbers
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = res.(Model)
	if m.activeTab != TabImages {
		t.Errorf("expected TabImages, got %d", m.activeTab)
	}

	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = res.(Model)
	if m.activeTab != TabCompose {
		t.Errorf("expected TabCompose, got %d", m.activeTab)
	}

	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	m = res.(Model)
	if m.activeTab != TabNetworks {
		t.Errorf("expected TabNetworks, got %d", m.activeTab)
	}

	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = res.(Model)
	if m.activeTab != TabVolumes {
		t.Errorf("expected TabVolumes, got %d", m.activeTab)
	}

	// Swarm tab when inactive shows toast
	m.swarm.IsActive = false
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "not part of an active Swarm") {
		t.Errorf("expected swarm inactive toast, got %s", m.actionMessage)
	}

	// Swarm tab when active switches to TabSwarm
	m.swarm.IsActive = true
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
	m = res.(Model)
	if m.activeTab != TabSwarm {
		t.Errorf("expected TabSwarm, got %d", m.activeTab)
	}

	// 3. Arrow keys tab navigation
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyLeft})
	m = res.(Model)
	if m.activeTab != TabVolumes {
		t.Errorf("expected TabVolumes after left arrow, got %d", m.activeTab)
	}

	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRight})
	m = res.(Model)
	if m.activeTab != TabSwarm {
		t.Errorf("expected TabSwarm after right arrow, got %d", m.activeTab)
	}

	// 4. Refresh key 'r'
	res, cmd := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Refreshing data") || cmd == nil {
		t.Errorf("expected refresh toast and non-nil cmd")
	}

	// 5. Container sort mode 'o' on Docker tab
	m.activeTab = TabDocker
	m.containerSortMode = SortDefault
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = res.(Model)
	if m.containerSortMode != SortCPU {
		t.Errorf("expected SortCPU after 'o', got %d", m.containerSortMode)
	}

	// 6. Quit key 'q'
	_, quitCmd := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if quitCmd == nil {
		t.Errorf("expected non-nil quit command")
	}
}

func TestHandleMainKeys_DomainDispatch(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v1.0.0")
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web"), testutil.WithState("running", "Up 1 hour")),
	}
	m.activeTab = TabDocker
	m.cursorIdx = 0

	// 1. Inspect on Docker tab with 'enter'
	res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.activeModal != ModalInspect && !strings.Contains(m.actionMessage, "Inspect error") {
		t.Errorf("expected ModalInspect opened or inspect error, got modal=%q msg=%q", m.activeModal, m.actionMessage)
	}
	m.closeModal()

	// 2. Logs on Docker tab with 'l'
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = res.(Model)
	if m.activeModal != ModalLogs && !strings.Contains(m.actionMessage, "Logs error") {
		t.Errorf("expected ModalLogs opened or logs error, got modal=%q msg=%q", m.activeModal, m.actionMessage)
	}
	m.closeModal()

	// 3. Remove container on Docker tab with 'd'
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = res.(Model)
	if m.activeModal != ModalRemoveContainer {
		t.Errorf("expected ModalRemoveContainer opened with 'd', got %s", m.activeModal)
	}
	m.closeModal()

	// 4. Prune modal with 'c'
	res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = res.(Model)
	if m.activeModal != ModalPrune {
		t.Errorf("expected ModalPrune opened with 'c', got %s", m.activeModal)
	}
	m.closeModal()
}
