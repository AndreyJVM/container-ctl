package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func createResilienceModel() Model {
	m := Model{
		version:         "v0.4.1",
		activeTab:       TabDocker,
		refreshInterval: 2 * time.Second,
		showSwarm:       true,
	}
	m.dockerInfo = monitor.DockerSummary{
		Status:            "running",
		StatusDisplay:     "Running",
		RunningContainers: 2,
		StoppedContainers: 1,
		ImagesCount:       3,
		Version:           "27.4.0",
		Containers: []monitor.ContainerItem{
			testutil.AContainer(testutil.WithName("app-1"), testutil.WithState("running", "Up 2 hours")),
			testutil.AContainer(testutil.WithName("app-2"), testutil.WithState("exited", "Exited (0) 10m ago")),
		},
	}
	m.images = []monitor.DockerImage{
		testutil.AnImage(testutil.WithRepoTag("redis", "alpine")),
		testutil.AnImage(testutil.WithRepoTag("postgres", "16")),
	}
	m.compose = []monitor.ComposeProject{
		{
			Name:            "web-stack",
			Status:          "running",
			StatusDisplay:   "Running",
			TotalServices:   2,
			RunningServices: 2,
			Services: []monitor.ComposeService{
				{Name: "api", ContainerName: "web-stack_api_1", State: "running", Status: "Up"},
				{Name: "db", ContainerName: "web-stack_db_1", State: "running", Status: "Up"},
			},
		},
	}
	m.networks = []monitor.DockerNetwork{
		testutil.ANetwork(),
	}
	m.volumes = []monitor.DockerVolume{
		testutil.AVolume(),
	}
	m.swarm = monitor.SwarmSummary{
		IsActive:      true,
		StatusDisplay: "Active",
		IsManager:     true,
		NodesCount:    3,
		ManagersCount: 1,
		Services: []monitor.SwarmService{
			{
				ID:              "srv1",
				Name:            "frontend",
				Image:           "nginx:alpine",
				Mode:            "replicated",
				Replicas:        "3/3",
				RunningReplicas: 3,
				TargetReplicas:  3,
			},
		},
	}
	return m
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

	tabs := []TabID{
		TabDocker,
		TabImages,
		TabCompose,
		TabNetworks,
		TabVolumes,
		TabSwarm,
	}

	modals := []ModalType{
		ModalNone,
		ModalLogs,
		ModalInspect,
		ModalComposeDown,
		ModalRemoveContainer,
		ModalRemoveImage,
		ModalExec,
		ModalPrune,
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
					m.activeTab = tab
					m.activeModal = mod

					m.logsLines = []string{"line 1", "line 2", "line 3"}
					m.logsTargetName = "test-container"
					m.logsTargetID = "12345678"

					m.inspectTarget = monitor.ContainerDetail{
						ID:     "abcdef123456",
						Name:   "inspect-target",
						State:  "running",
						Status: "Up 2 hours",
						Image:  "alpine:latest",
						Env:    []string{"KEY=val", "SECRET=topsecret"},
						Mounts: []string{"/host:/container"},
					}

					// Send WindowSizeMsg
					updatedModel, _ := m.Update(tea.WindowSizeMsg{Width: dim.width, Height: dim.height})
					resModel := updatedModel.(Model)

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

func TestResilience_LogsModalStressAndUnicode(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.4.1")
	m.width = 100
	m.height = 30
	m.activeModal = ModalLogs
	m.logsTargetName = "stress-app"
	m.logsTargetID = "stress123"

	// 1. Stress: 50,000 log lines
	largeLogs := make([]string, 50000)
	for i := range largeLogs {
		largeLogs[i] = fmt.Sprintf("2026-11-20T18:00:%02d.000Z [INFO] Processing request #%d with metadata payload key=value", i%60, i)
	}
	m.logsLines = largeLogs

	// Test boundary scrolls
	scrolls := []int{-100, -1, 0, 10, 25000, 49999, 50000, 999999}
	for _, scr := range scrolls {
		m.logsScroll = scr
		out := m.renderLogsModal()
		if len(out) == 0 {
			t.Fatalf("renderLogsModal returned empty string for scroll %d", scr)
		}
	}

	// 2. Filter with special regex / glob / Unicode chars
	filterQueries := []string{
		".*",
		"[a-z]+",
		"(?i)error",
		"\\d{4}",
		"***",
		"[[[",
		"🚀 Unicode Emoji 💥",
		"Привет мир! Тестирование логов",
		"日本語のログメッセージ",
		"\x1b[31mRed Ansi\x1b[0m",
		strings.Repeat("A", 2000),
	}

	for _, q := range filterQueries {
		m.logsFilterQuery = q
		m.logsFiltering = false
		out := m.renderLogsModal()
		if len(out) == 0 {
			t.Fatalf("renderLogsModal returned empty for query %q", q)
		}
	}
}

func TestResilience_CursorOutOfBoundsSafety(t *testing.T) {
	tabs := []TabID{TabDocker, TabImages, TabCompose, TabNetworks, TabVolumes, TabSwarm}

	for _, tab := range tabs {
		t.Run(fmt.Sprintf("Tab_%d_BadCursors", tab), func(_ *testing.T) {
			m := NewModel(2*time.Second, 1, "v0.4.1")
			m.SetShowSwarm(true)
			m.activeTab = tab
			m.width = 120
			m.height = 40

			// Case A: Completely empty data
			m.dockerInfo = monitor.DockerSummary{}
			m.images = nil
			m.compose = nil
			m.networks = nil
			m.volumes = nil
			m.swarm = monitor.SwarmSummary{}

			badIndices := []int{-999, -1, 0, 1, 999}
			for _, idx := range badIndices {
				m.cursorIdx = idx
				m.imageCursorIdx = idx
				m.composeCursorIdx = idx
				m.networkCursorIdx = idx
				m.volumeCursorIdx = idx
				m.swarmCursorIdx = idx

				// Neither View() nor navigation keypresses should panic
				_ = m.View()

				for _, key := range []string{"up", "down", "pgup", "pgdown", "home", "end", "enter"} {
					updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
					m = updated.(Model)
				}
			}

			// Case B: Populated data with out of bounds cursor
			m = createResilienceModel()
			m.activeTab = tab
			m.width = 120
			m.height = 40

			for _, idx := range badIndices {
				m.cursorIdx = idx
				m.imageCursorIdx = idx
				m.composeCursorIdx = idx
				m.networkCursorIdx = idx
				m.volumeCursorIdx = idx
				m.swarmCursorIdx = idx

				_ = m.View()
			}
		})
	}
}

func TestResilience_CorruptedAndBoundaryContainerData(t *testing.T) {
	m := NewModel(2*time.Second, 1, "v0.4.1")
	m.width = 120
	m.height = 40
	m.activeTab = TabDocker

	// Corrupted container items
	m.dockerInfo = monitor.DockerSummary{
		Status: "running",
		Containers: []monitor.ContainerItem{
			{
				ID:         "",
				Name:       "",
				Image:      "",
				Status:     "",
				State:      "",
				Ports:      "",
				Mounts:     nil,
				CPUPerc:    -50.0,
				MemPerc:    -10.0,
				MemUsageMB: -100.0,
				MemLimitMB: -500.0,
				NetRxBytes: 0,
				NetTxBytes: 0,
				Health:     "unknown_weird_health",
			},
			{
				ID:              strings.Repeat("9", 256),
				Name:            "🔥 " + strings.Repeat("ОченьДлинноеИмя", 50) + " 🚀",
				Image:           strings.Repeat("repo/image:tag@", 30),
				Status:          strings.Repeat("Up 999 days ", 10),
				State:           "running",
				Ports:           "0.0.0.0:80->80/tcp, :::8080->8080/tcp, 127.0.0.1:443->443/tcp, invalid_port_mapping",
				Mounts:          []string{strings.Repeat("/very/long/path/", 20)},
				CPUPerc:         99999.9,
				MemPerc:         1000.0,
				MemUsageMB:      9999999.0,
				MemLimitMB:      9999999.0,
				NetRxBytes:      ^uint64(0), // MaxUint64
				NetTxBytes:      ^uint64(0),
				BlockReadBytes:  ^uint64(0),
				BlockWriteBytes: ^uint64(0),
				Health:          "healthy",
				ErrorSnippet:    strings.Repeat("Critical Error Occurred\n", 50),
				ComposeProject:  strings.Repeat("project_", 10),
				ComposeService:  strings.Repeat("service_", 10),
			},
		},
	}

	m.cursorIdx = 0
	out0 := m.View()
	if len(out0) == 0 {
		t.Errorf("View() returned empty string for empty container item")
	}

	m.cursorIdx = 1
	out1 := m.View()
	if len(out1) == 0 {
		t.Errorf("View() returned empty string for extreme boundary container item")
	}
}
