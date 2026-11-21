package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

// newFastTestModel creates an initialized Model for testing without making
// slow and blocking calls to the Docker daemon.
func newFastTestModel() Model {
	return Model{
		version:         "v0.5.0",
		activeTab:       TabDocker,
		refreshInterval: 2 * time.Second,
	}
}

func TestAsyncCommandTriggers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		containers    []monitor.ContainerItem
		cursor        int
		key           string
		expectedToast string
	}{
		{
			name: "press 's' on running container -> stops asynchronously",
			containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web"), testutil.WithState("running", "Up")),
			},
			cursor:        0,
			key:           "s",
			expectedToast: "Stopping container web",
		},
		{
			name: "press 's' on stopped container -> starts asynchronously",
			containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db"), testutil.WithState("exited", "Exited")),
			},
			cursor:        0,
			key:           "s",
			expectedToast: "Starting container db",
		},
		{
			name: "press 'p' on container -> restarts asynchronously",
			containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web"), testutil.WithState("running", "Up")),
			},
			cursor:        0,
			key:           "p",
			expectedToast: "Restarting web",
		},
		{
			name:          "press 'r' -> refresh command",
			containers:    nil,
			cursor:        0,
			key:           "r",
			expectedToast: "Refreshing data",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newFastTestModel()
			m.dockerInfo.Containers = tc.containers
			m.cursorIdx = tc.cursor

			res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)})
			updated := res.(Model)

			if cmd == nil {
				t.Fatalf("expected non-nil tea.Cmd for key %q", tc.key)
			}
			if !strings.Contains(updated.actionMessage, tc.expectedToast) {
				t.Errorf("expected toast containing %q, got %q", tc.expectedToast, updated.actionMessage)
			}
		})
	}
}

func TestActionFinishedMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		msg           actionFinishedMsg
		expectedToast string
		expectCmd     bool
	}{
		{
			name:          "success message without refresh",
			msg:           actionFinishedMsg{message: "Operation completed successfully", refresh: false},
			expectedToast: "Operation completed successfully",
			expectCmd:     false,
		},
		{
			name:          "error message",
			msg:           actionFinishedMsg{err: errors.New("command execution failed"), refresh: false},
			expectedToast: "command execution failed",
			expectCmd:     false,
		},
		{
			name:          "success message with refresh -> triggers fetchDataCmd",
			msg:           actionFinishedMsg{message: "Container removed", refresh: true},
			expectedToast: "Container removed",
			expectCmd:     true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newFastTestModel()
			res, cmd := m.Update(tc.msg)
			updated := res.(Model)

			if (cmd != nil) != tc.expectCmd {
				t.Errorf("expected cmd: %v, got: %v", tc.expectCmd, cmd != nil)
			}
			if !strings.Contains(updated.actionMessage, tc.expectedToast) {
				t.Errorf("expected toast containing %q, got: %q", tc.expectedToast, updated.actionMessage)
			}
		})
	}
}

func TestDataFetchedMsg(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	data := dataFetchedMsg{
		timestamp: "14:25:36",
		dockerInfo: monitor.DockerSummary{
			Status:            monitor.DockerStatusRunning,
			RunningContainers: 2,
		},
	}

	res, cmd := m.Update(data)
	updated := res.(Model)

	if cmd != nil {
		t.Errorf("expected nil cmd after dataFetchedMsg")
	}
	if updated.lastUpdate != "14:25:36" {
		t.Errorf("expected lastUpdate='14:25:36', got: %s", updated.lastUpdate)
	}
	if updated.dockerInfo.RunningContainers != 2 {
		t.Errorf("expected 2 running containers, got: %d", updated.dockerInfo.RunningContainers)
	}
}

func TestWrapAction(t *testing.T) {
	t.Parallel()

	t.Run("converts actions.FinishedMsg to actionFinishedMsg", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("underlying error")
		cmd := wrapAction(func() tea.Msg {
			return actions.FinishedMsg{
				Message: "action done",
				Err:     testErr,
				Refresh: true,
			}
		})
		res := cmd()
		afm, ok := res.(actionFinishedMsg)
		if !ok {
			t.Fatalf("expected actionFinishedMsg, got %T", res)
		}
		if afm.message != "action done" {
			t.Errorf("expected message 'action done', got %q", afm.message)
		}
		if !errors.Is(afm.err, testErr) {
			t.Errorf("expected error %v, got %v", testErr, afm.err)
		}
		if !afm.refresh {
			t.Errorf("expected refresh=true")
		}
	})

	t.Run("passes through non-FinishedMsg types untouched", func(t *testing.T) {
		t.Parallel()
		cmd := wrapAction(func() tea.Msg {
			return "custom string msg"
		})
		res := cmd()
		if s, ok := res.(string); !ok || s != "custom string msg" {
			t.Errorf("expected 'custom string msg', got %v", res)
		}
	})

	t.Run("passes through nil message", func(t *testing.T) {
		t.Parallel()
		cmd := wrapAction(func() tea.Msg {
			return nil
		})
		if res := cmd(); res != nil {
			t.Errorf("expected nil msg, got %v", res)
		}
	})
}

func TestMakeActionCmd(t *testing.T) {
	t.Parallel()

	t.Run("success case", func(t *testing.T) {
		t.Parallel()
		cmd := makeActionCmd(
			func() error { return nil },
			"prefix: ",
			"done successfully",
		)
		msg := cmd()
		afm, ok := msg.(actionFinishedMsg)
		if !ok {
			t.Fatalf("expected actionFinishedMsg, got %T", msg)
		}
		if afm.err != nil || afm.message != "done successfully" || !afm.refresh {
			t.Errorf("unexpected success actionFinishedMsg: %+v", afm)
		}
	})

	t.Run("error case with error wrapping", func(t *testing.T) {
		t.Parallel()
		testErr := errors.New("network failure")
		cmd := makeActionCmd(
			func() error { return testErr },
			"action error: ",
			"should not be used",
		)
		msg := cmd()
		afm, ok := msg.(actionFinishedMsg)
		if !ok {
			t.Fatalf("expected actionFinishedMsg, got %T", msg)
		}
		if afm.err == nil || !strings.Contains(afm.err.Error(), "action error: network failure") {
			t.Errorf("expected wrapped error with prefix, got: %+v", afm.err)
		}
		if !errors.Is(afm.err, testErr) {
			t.Errorf("expected wrapped error to unwrap to testErr via %%w")
		}
	})
}

func TestExecutePruneAndPruneCmd(t *testing.T) {
	t.Parallel()

	t.Run("invalid option returns error", func(t *testing.T) {
		t.Parallel()
		_, err := executePrune(99)
		if err == nil || !strings.Contains(err.Error(), "unknown prune option: 99") {
			t.Errorf("expected error for unknown option, got: %v", err)
		}

		badCmd := pruneCmd(99)
		badMsg := badCmd()
		afm, ok := badMsg.(actionFinishedMsg)
		if !ok {
			t.Fatalf("expected actionFinishedMsg, got %T", badMsg)
		}
		if afm.err == nil || !strings.Contains(afm.err.Error(), "Prune error:") {
			t.Errorf("expected prune error toast, got: %+v", afm)
		}
	})

	t.Run("valid options return non-nil tea.Cmd", func(t *testing.T) {
		t.Parallel()
		opts := []int{PruneOptionContainers, PruneOptionImages, PruneOptionVolumes, PruneOptionAll}
		for _, opt := range opts {
			cmd := pruneCmd(opt)
			if cmd == nil {
				t.Errorf("expected non-nil cmd for option %d", opt)
			}
		}
	})
}

func TestKillAndRemoveCommandsExecution(t *testing.T) {
	t.Parallel()

	killCmd := killContainerCmd("c-test", "test-app")
	if killCmd == nil {
		t.Fatal("expected non-nil tea.Cmd for killContainerCmd")
	}

	removeCmd := removeContainerCmd("c-test", "test-app")
	if removeCmd == nil {
		t.Fatal("expected non-nil tea.Cmd for removeContainerCmd")
	}
}

func TestRemoveImageCmdExecution(t *testing.T) {
	t.Parallel()

	cmd := removeImageCmd("img-id", "nginx:alpine", false)
	if cmd == nil {
		t.Fatal("expected non-nil tea.Cmd for removeImageCmd")
	}
}

func TestContainerAndComposeCommandsExecution(t *testing.T) {
	t.Parallel()

	cmds := []struct {
		name string
		cmd  tea.Cmd
	}{
		{"startContainer", startContainerCmd("c1", "web")},
		{"stopContainer", stopContainerCmd("c1", "web")},
		{"restartContainer", restartContainerCmd("c1", "web")},
		{"killContainer", killContainerCmd("c1", "web")},
		{"removeContainer", removeContainerCmd("c1", "web")},
		{"removeImage", removeImageCmd("img1", "alpine:latest", false)},
		{"startCompose", startComposeCmd("stack", nil)},
		{"stopCompose", stopComposeCmd("stack", nil)},
		{"restartCompose", restartComposeCmd("stack", nil)},
		{"downCompose", downComposeCmd("stack", nil)},
	}

	for _, tc := range cmds {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.cmd == nil {
				t.Fatalf("expected non-nil cmd for %s", tc.name)
			}
		})
	}
}

func TestFetchAllDataAndFetchDataCmd(t *testing.T) {
	t.Parallel()

	cmd := fetchDataCmd()
	if cmd == nil {
		t.Fatal("expected non-nil cmd from fetchDataCmd()")
	}

	data := fetchAllData()
	if len(data.timestamp) != 8 || !strings.Contains(data.timestamp, ":") {
		t.Errorf("expected timestamp in HH:MM:SS format, got %q", data.timestamp)
	}
}

func TestFilterModesAndSortSetters(t *testing.T) {
	t.Parallel()

	m := newFastTestModel()
	if m.IsOnlyUnusedImages() {
		t.Errorf("expected default IsOnlyUnusedImages to be false")
	}

	m.SetOnlyUnusedImages(true)
	if !m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be true after setter")
	}

	m.SetOnlyUnusedImages(false)
	if m.IsOnlyUnusedImages() {
		t.Errorf("expected IsOnlyUnusedImages to be false after setter")
	}

	m.SetContainerSortMode(SortCPU)
	if m.containerSortMode != SortCPU {
		t.Errorf("expected SortCPU, got %d", m.containerSortMode)
	}
	m.SetContainerSortMode(SortMem)
	if m.containerSortMode != SortMem {
		t.Errorf("expected SortMem, got %d", m.containerSortMode)
	}
	m.SetContainerSortMode(999) // Invalid mode out of bounds
	if m.containerSortMode != SortMem {
		t.Errorf("expected mode to remain SortMem when invalid passed, got %d", m.containerSortMode)
	}
}
