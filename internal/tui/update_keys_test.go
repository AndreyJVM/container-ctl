package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleFilterInput(t *testing.T) {
	t.Parallel()

	t.Run("typing runes and space", func(t *testing.T) {
		m := newFastTestModel()
		m.isFiltering = true

		for _, r := range "alp ha" {
			var msg tea.KeyMsg
			if r == ' ' {
				msg = tea.KeyMsg{Type: tea.KeySpace}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
			}
			res, _ := m.handleFilterInput(msg)
			m = res.(Model)
		}
		if m.filterQuery != "alp ha" {
			t.Errorf("expected filterQuery 'alp ha', got %q", m.filterQuery)
		}
	})

	t.Run("backspace handling and empty safety", func(t *testing.T) {
		m := newFastTestModel()
		m.isFiltering = true
		m.filterQuery = "alp"

		// Backspace deletes last character
		res, _ := m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})
		m = res.(Model)
		if m.filterQuery != "al" {
			t.Errorf("expected filterQuery 'al' after backspace, got %q", m.filterQuery)
		}

		// Unicode backspace
		m.filterQuery = "руна"
		res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})
		m = res.(Model)
		if m.filterQuery != "рун" {
			t.Errorf("expected 'рун' after unicode backspace, got %q", m.filterQuery)
		}

		// Backspace on empty string does not panic
		m.filterQuery = ""
		res, _ = m.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})
		m = res.(Model)
		if m.filterQuery != "" {
			t.Errorf("expected empty filterQuery after backspace on empty, got %q", m.filterQuery)
		}
	})

	t.Run("enter exits filter input preserving query", func(t *testing.T) {
		m := newFastTestModel()
		m.isFiltering = true
		m.filterQuery = "redis"

		res, _ := m.handleFilterInput(tea.KeyMsg{Type: tea.KeyEnter})
		m = res.(Model)
		if m.isFiltering {
			t.Errorf("expected isFiltering=false after enter")
		}
		if m.filterQuery != "redis" {
			t.Errorf("expected filterQuery preserved, got %q", m.filterQuery)
		}
	})

	t.Run("esc cancels filter and clears query", func(t *testing.T) {
		m := newFastTestModel()
		m.isFiltering = true
		m.filterQuery = "postgres"

		res, _ := m.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
		m = res.(Model)
		if m.isFiltering {
			t.Errorf("expected isFiltering=false after esc")
		}
		if m.filterQuery != "" {
			t.Errorf("expected filterQuery cleared after esc, got %q", m.filterQuery)
		}
	})

	t.Run("unhandled keys in filter mode", func(t *testing.T) {
		m := newFastTestModel()
		m.isFiltering = true
		m.filterQuery = "test"

		for _, msg := range []tea.KeyMsg{
			{Type: tea.KeyUp},
			{Type: tea.KeyDown},
			{Type: tea.KeyTab},
		} {
			res, cmd := m.handleFilterInput(msg)
			m = res.(Model)
			if cmd != nil {
				t.Errorf("msg %v: expected nil cmd", msg)
			}
			if m.filterQuery != "test" {
				t.Errorf("msg %v: expected query unchanged, got %q", msg, m.filterQuery)
			}
		}
	})
}

func TestHandleMainKeys_NavigationAndTabs(t *testing.T) {
	t.Parallel()

	t.Run("help toggle and quit keys", func(t *testing.T) {
		m := newFastTestModel()

		// Help toggle with '?'
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		m = res.(Model)
		if !m.showHelp {
			t.Errorf("expected showHelp=true after '?'")
		}

		// Quit with 'q'
		_, quitCmd := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if quitCmd == nil {
			t.Errorf("expected non-nil quit command on 'q'")
		}

		// Quit with ctrl+c
		_, quitCmd = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyCtrlC})
		if quitCmd == nil {
			t.Errorf("expected non-nil quit command on ctrl+c")
		}
	})

	t.Run("tab switching via numbers and arrows", func(t *testing.T) {
		m := newFastTestModel()

		numberTests := []struct {
			key         string
			expectedTab TabID
		}{
			{"2", TabImages},
			{"3", TabCompose},
			{"4", TabNetworks},
			{"5", TabVolumes},
			{"1", TabDocker},
		}

		for _, nt := range numberTests {
			res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(nt.key)})
			m = res.(Model)
			if m.activeTab != nt.expectedTab {
				t.Errorf("key %q: expected tab %d, got %d", nt.key, nt.expectedTab, m.activeTab)
			}
		}

		// Swarm tab when inactive shows toast
		m.setTab(TabDocker)
		m.swarm.IsActive = false
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'6'}})
		m = res.(Model)
		if m.activeTab != TabDocker {
			t.Errorf("expected to stay on TabDocker when swarm inactive")
		}
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

		// Arrow navigation
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
	})

	t.Run("refresh command on 'r' and 'R'", func(t *testing.T) {
		for _, key := range []string{"r", "R"} {
			m := newFastTestModel()
			res, cmd := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			m = res.(Model)
			if !strings.Contains(m.actionMessage, "Refreshing data") {
				t.Errorf("key %q: expected refresh toast, got %q", key, m.actionMessage)
			}
			if cmd == nil {
				t.Errorf("key %q: expected non-nil refresh command", key)
			}
		}
	})

	t.Run("container sort cycling on 'o' and 'O'", func(t *testing.T) {
		m := newFastTestModel()
		m.activeTab = TabDocker
		m.containerSortMode = SortDefault

		expectedModes := []int{SortCPU, SortMem, SortName, SortStatus, SortDefault}
		for _, expected := range expectedModes {
			res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
			m = res.(Model)
			if m.containerSortMode != expected {
				t.Errorf("expected containerSortMode=%d, got %d", expected, m.containerSortMode)
			}
			if !strings.HasPrefix(m.actionMessage, "Sort: ") {
				t.Errorf("expected toast starting with 'Sort: ', got %q", m.actionMessage)
			}
		}

		// Sorting hotkey ignored when on non-Docker tabs
		m.setTab(TabImages)
		m.containerSortMode = SortDefault
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		m = res.(Model)
		if m.containerSortMode != SortDefault {
			t.Errorf("expected sort mode unchanged on Images tab")
		}

		// Sorting hotkey ignored when isFiltering is active
		m.setTab(TabDocker)
		m.isFiltering = true
		res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		m = res.(Model)
		if m.containerSortMode != SortDefault {
			t.Errorf("expected sort mode unchanged when isFiltering=true")
		}
	})
}

func TestHandleMainKeys_FilterActivationAndEsc(t *testing.T) {
	t.Parallel()

	t.Run("slash activates filter on supported tabs", func(t *testing.T) {
		supportedTabs := []TabID{TabDocker, TabImages, TabCompose, TabNetworks, TabVolumes}
		for _, tab := range supportedTabs {
			m := newFastTestModel()
			m.setTab(tab)
			res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
			m = res.(Model)
			if !m.isFiltering {
				t.Errorf("tab %d: expected isFiltering=true after '/'", tab)
			}
		}

		// Slash ignored on TabSwarm
		m := newFastTestModel()
		m.SetShowSwarm(true)
		m.setTab(TabSwarm)
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
		m = res.(Model)
		if m.isFiltering {
			t.Errorf("expected isFiltering=false on TabSwarm after '/'")
		}
	})

	t.Run("esc clears filter query", func(t *testing.T) {
		m := newFastTestModel()
		m.filterQuery = "search-text"
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyEsc})
		m = res.(Model)
		if m.filterQuery != "" {
			t.Errorf("expected filterQuery cleared on esc, got %q", m.filterQuery)
		}
	})

	t.Run("esc resets onlyUnusedImages on Images tab", func(t *testing.T) {
		m := newFastTestModel()
		m.setTab(TabImages)
		m.onlyUnusedImages = true
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyEsc})
		m = res.(Model)
		if m.onlyUnusedImages {
			t.Errorf("expected onlyUnusedImages=false on esc")
		}
		if !strings.Contains(m.actionMessage, "Showing all images") {
			t.Errorf("expected toast 'Showing all images', got %q", m.actionMessage)
		}
	})
}

func TestHandleMainKeys_ModalTriggers(t *testing.T) {
	t.Parallel()

	t.Run("remove modal triggers via 'd' and 'x'", func(t *testing.T) {
		// 1. Docker tab -> ModalRemoveContainer
		m := newFastTestModel()
		m.activeTab = TabDocker
		m.dockerInfo.Containers = []monitor.ContainerItem{
			testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web")),
		}
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		m = res.(Model)
		if m.activeModal != ModalRemoveContainer {
			t.Errorf("expected ModalRemoveContainer with 'd', got %s", m.activeModal)
		}

		m.closeModal()
		res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m = res.(Model)
		if m.activeModal != ModalRemoveContainer {
			t.Errorf("expected ModalRemoveContainer with 'x', got %s", m.activeModal)
		}

		// 2. Images tab -> ModalRemoveImage
		m = newFastTestModel()
		m.activeTab = TabImages
		m.images = []monitor.DockerImage{
			testutil.AnImage(testutil.WithImageID("img1", "sha256:img1")),
		}
		res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		m = res.(Model)
		if m.activeModal != ModalRemoveImage {
			t.Errorf("expected ModalRemoveImage with 'd', got %s", m.activeModal)
		}

		m.closeModal()
		res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m = res.(Model)
		if m.activeModal != ModalRemoveImage {
			t.Errorf("expected ModalRemoveImage with 'x', got %s", m.activeModal)
		}

		// 3. Compose tab -> ModalComposeDown
		m = newFastTestModel()
		m.activeTab = TabCompose
		m.compose = []monitor.ComposeProject{{Name: "stack-1"}}
		res, _ = m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		m = res.(Model)
		if m.activeModal != ModalComposeDown {
			t.Errorf("expected ModalComposeDown with 'd', got %s", m.activeModal)
		}
	})

	t.Run("exec modal trigger via 'e' and 'E'", func(t *testing.T) {
		for _, key := range []string{"e", "E"} {
			m := newFastTestModel()
			m.activeTab = TabDocker
			m.dockerInfo.Containers = []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web")),
			}
			res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			m = res.(Model)
			if m.activeModal != ModalExec {
				t.Errorf("key %q: expected ModalExec, got %s", key, m.activeModal)
			}
		}
	})

	t.Run("prune modal trigger via 'c' and 'C'", func(t *testing.T) {
		origDiskUsage := getDockerDiskUsage
		getDockerDiskUsage = func() (monitor.DockerDiskUsage, error) {
			return monitor.DockerDiskUsage{TotalSize: 1024}, nil
		}
		defer func() { getDockerDiskUsage = origDiskUsage }()

		for _, key := range []string{"c", "C"} {
			m := newFastTestModel()
			m.activeTab = TabDocker
			res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			m = res.(Model)
			if m.activeModal != ModalPrune {
				t.Errorf("key %q: expected ModalPrune, got %s", key, m.activeModal)
			}
			if m.pruneUsage.TotalSize != 1024 {
				t.Errorf("expected pruneUsage.TotalSize 1024, got %d", m.pruneUsage.TotalSize)
			}
		}
	})

	t.Run("prune modal trigger error handling", func(t *testing.T) {
		origDiskUsage := getDockerDiskUsage
		getDockerDiskUsage = func() (monitor.DockerDiskUsage, error) {
			return monitor.DockerDiskUsage{}, errors.New("daemon down")
		}
		defer func() { getDockerDiskUsage = origDiskUsage }()

		m := newFastTestModel()
		m.activeTab = TabDocker
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
		m = res.(Model)
		if m.activeModal != ModalPrune {
			t.Errorf("expected ModalPrune, got %s", m.activeModal)
		}
		if m.pruneUsageErr != "daemon down" {
			t.Errorf("expected pruneUsageErr 'daemon down', got %q", m.pruneUsageErr)
		}
	})
}

func TestHandleMainKeys_FilteringGuards(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	m.activeTab = TabDocker
	m.isFiltering = true
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web")),
	}

	// Hotkeys should be guarded/suppressed while typing in search/filter
	guardedKeys := []string{"d", "x", "u", "p", "c", "w", "y", "o", "K", "home", "end", "g", "G"}
	for _, k := range guardedKeys {
		res, _ := m.handleMainKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		m = res.(Model)
		if m.activeModal != ModalNone {
			t.Errorf("key %q: expected activeModal to remain ModalNone when isFiltering=true, got %s", k, m.activeModal)
		}
	}
}
