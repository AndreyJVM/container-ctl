package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

func createResilienceModel() tui.Model {
	return createComprehensiveTestModel()
}

func TestResilience_ExtremeWindowSizes(t *testing.T) {
	testDimensions := []struct {
		width  int
		height int
	}{
		{-100, -100},
		{-1, -1},
		{0, 0},
		{1, 1},
		{5, 5},
		{10, 5},
		{20, 10},
		{40, 15},
		{80, 24},
		{120, 40},
		{300, 100},
		{1000, 500},
	}

	tabs := []tui.TabID{
		tui.TabDocker,
		tui.TabImages,
		tui.TabCompose,
		tui.TabNetworks,
		tui.TabVolumes,
		tui.TabSwarm,
	}

	modals := []tui.ModalType{
		tui.ModalNone,
		tui.ModalLogs,
		tui.ModalInspect,
		tui.ModalComposeDown,
		tui.ModalRemoveContainer,
		tui.ModalRemoveImage,
		tui.ModalExec,
		tui.ModalPrune,
	}

	for _, dim := range testDimensions {
		for _, tab := range tabs {
			for _, mod := range modals {
				t.Run(fmt.Sprintf("Dim_%dx%d_Tab_%d_Modal_%s", dim.width, dim.height, tab, mod), func(t *testing.T) {
					defer func() {
						if r := recover(); r != nil {
							t.Fatalf("PANIC at dim %dx%d, tab %d, modal %s: %v", dim.width, dim.height, tab, mod, r)
						}
					}()

					m := createResilienceModel()
					m.SetTab(tab)
					m.OpenModal(mod)
					m.SetLogsForTesting("test-container", "12345678", []string{"line 1", "line 2", "line 3"})
					m.SetInspectForTesting(monitor.ContainerDetail{
						ID:     "abcdef123456",
						Name:   "inspect-target",
						State:  "running",
						Status: "Up 2 hours",
						Image:  "alpine:latest",
						Env:    []string{"KEY=val", "SECRET=topsecret"},
						Mounts: []string{"/host:/container"},
					})

					// Send WindowSizeMsg
					updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: dim.width, Height: dim.height})
					resModel := updatedModel.(tui.Model)

					// View() must render without crashing/panicking
					out := resModel.View()
					if dim.width > 20 && dim.height > 10 && len(out) == 0 {
						t.Errorf("View() returned empty string for dimensions %dx%d", dim.width, dim.height)
					}
				})
			}
		}
	}
}

func TestResilience_CursorOutOfBoundsSafety(t *testing.T) {
	tabs := []tui.TabID{tui.TabDocker, tui.TabImages, tui.TabCompose, tui.TabNetworks, tui.TabVolumes, tui.TabSwarm}

	for _, tab := range tabs {
		t.Run(fmt.Sprintf("Tab_%d_BadCursors", tab), func(_ *testing.T) {
			m := tui.NewModel(2*time.Second, 1, "v0.4.1")
			m.SetShowSwarm(true)
			m.SetTab(tab)

			updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
			m = updated.(tui.Model)

			// Completely empty data
			updated, _ = m.Update(actions.DataFetchedMsg{})
			m = updated.(tui.Model)

			badIndices := []int{-999, -1, 0, 1, 999}
			for _, idx := range badIndices {
				m.SetCursorForTesting(idx)

				// Neither View() nor navigation keypresses should panic
				_ = m.View()

				for _, key := range []string{"up", "down", "pgup", "pgdown", "home", "end", "enter"} {
					res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
					m = res.(tui.Model)
					_ = m.View()
				}
			}
		})
	}
}

func TestResilience_CorruptedAndBoundaryContainerData(t *testing.T) {
	boundaryData := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Status:        "running",
			StatusDisplay: "Running",
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithName(strings.Repeat("LongContainerName_", 20)), testutil.WithMetrics(-100.0, -50.0, -10.0)),
				testutil.AContainer(testutil.WithName("HugeMetrics"), testutil.WithMetrics(9999999.9, 9999999.0, 99999999.0)),
				testutil.AContainer(testutil.WithName("EmptyEverything"), testutil.WithState("", "")),
				testutil.AContainer(testutil.WithName("SpecialChars !@#$%^&*()_+-=[]{}|;':\",./<>?"), testutil.WithImage("repo/image:tag@sha256:abcdef")),
			},
		},
	}

	m := tui.NewModel(2*time.Second, 1, "v0.4.1")
	updated, _ := m.Update(boundaryData)
	m = updated.(tui.Model)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(tui.Model)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PANIC on corrupted data: %v", r)
		}
	}()

	view := m.View()
	if len(view) == 0 {
		t.Errorf("View() returned empty on boundary data")
	}
}
