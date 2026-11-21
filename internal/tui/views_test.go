package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
)

func TestViewModel_HeaderAndTabs(t *testing.T) {
	t.Parallel()

	t.Run("header renders version, status display and update time", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.version = "v0.9.9"
		m.lastUpdate = "14:25:36"
		m.dockerInfo = monitor.DockerSummary{
			Status:        "running",
			StatusDisplay: "Running",
		}

		v := m.View()
		if !strings.Contains(v, "v0.9.9") {
			t.Errorf("expected version 'v0.9.9' in View, got: %s", v)
		}
		if !strings.Contains(v, "Docker: Running") {
			t.Errorf("expected 'Docker: Running' in View, got: %s", v)
		}
		if !strings.Contains(v, "14:25:36") {
			t.Errorf("expected timestamp '14:25:36' in View, got: %s", v)
		}
	})

	t.Run("action message toast renders while active and disappears when expired", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.actionMessage = "Container restarted successfully"

		// Active toast
		m.actionMessageExpiry = time.Now().Add(5 * time.Second)
		vActive := m.View()
		if !strings.Contains(vActive, "[!] Container restarted successfully") {
			t.Errorf("expected toast in active View, got: %s", vActive)
		}

		// Expired toast
		m.actionMessageExpiry = time.Now().Add(-1 * time.Second)
		vExpired := m.View()
		if strings.Contains(vExpired, "[!] Container restarted successfully") {
			t.Errorf("expected expired toast to be absent in View, got: %s", vExpired)
		}
	})

	t.Run("swarm visibility in tabs bar", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()

		// Swarm inactive
		m.swarm = monitor.SwarmSummary{IsActive: false}
		vInactive := m.View()
		if strings.Contains(vInactive, "6: Swarm") {
			t.Errorf("expected '6: Swarm' absent when swarm is inactive, got: %s", vInactive)
		}
		if !strings.Contains(vInactive, "1: Containers") || !strings.Contains(vInactive, "5: Volumes") {
			t.Errorf("expected standard tabs 1..5 in View, got: %s", vInactive)
		}

		// Swarm active
		m.swarm = monitor.SwarmSummary{IsActive: true, StatusDisplay: "Active"}
		vActive := m.View()
		if !strings.Contains(vActive, "6: Swarm") {
			t.Errorf("expected '6: Swarm' present when swarm is active, got: %s", vActive)
		}
	})
}

func TestViewModel_TabsRendering(t *testing.T) {
	t.Parallel()

	t.Run("TabDocker rendering with containers, sort modes and search filter", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeTab = TabDocker
		m.dockerInfo = monitor.DockerSummary{
			Status:        "running",
			StatusDisplay: "Running",
			Containers: []monitor.ContainerItem{
				{
					ID:         "c1234567890",
					Name:       "dockpulse-api",
					Image:      "api:latest",
					State:      "running",
					Status:     "Up 3 hours",
					CPUPerc:    12.5,
					MemUsageMB: 256.0,
					MemLimitMB: 1024.0,
				},
			},
		}
		m.containerSortMode = SortCPU

		v := m.View()
		if !strings.Contains(v, "DOCKER CONTAINERS") {
			t.Errorf("expected 'DOCKER CONTAINERS' in View, got: %s", v)
		}
		if !strings.Contains(v, "dockpulse-api") {
			t.Errorf("expected container name 'dockpulse-api' in View, got: %s", v)
		}
		if !strings.Contains(v, "CPU ↓") {
			t.Errorf("expected sort badge 'CPU ↓' in View, got: %s", v)
		}

		// Search filter active
		m.isFiltering = true
		m.filterQuery = "api"
		vFilter := m.View()
		if !strings.Contains(vFilter, "[Search: api█]") {
			t.Errorf("expected search prompt in View, got: %s", vFilter)
		}
	})

	t.Run("TabImages rendering with images, unused filter and daemon error", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeTab = TabImages
		m.images = []monitor.DockerImage{
			{
				ID:         "sha256:11223344",
				Repository: "nginx",
				Tag:        "alpine",
				Size:       23 * 1024 * 1024,
				InUse:      false,
			},
		}

		v := m.View()
		if !strings.Contains(v, "DOCKER IMAGES") {
			t.Errorf("expected 'DOCKER IMAGES' in View, got: %s", v)
		}
		if !strings.Contains(v, "nginx") {
			t.Errorf("expected image repository 'nginx' in View, got: %s", v)
		}

		// Filter unused images
		m.onlyUnusedImages = true
		vUnused := m.View()
		if !strings.Contains(vUnused, "Unused Images Only") {
			t.Errorf("expected 'Unused Images Only' indicator in View, got: %s", vUnused)
		}

		// Error banner
		m.imageErr = "cannot connect to docker daemon"
		vErr := m.View()
		if !strings.Contains(vErr, "Docker images are unavailable: cannot connect to docker daemon") {
			t.Errorf("expected image error banner in View, got: %s", vErr)
		}
	})

	t.Run("TabCompose rendering with projects", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeTab = TabCompose
		m.compose = []monitor.ComposeProject{
			{
				Name:            "monitoring-stack",
				Status:          "running",
				StatusDisplay:   "Running",
				TotalServices:   3,
				RunningServices: 3,
			},
		}

		v := m.View()
		if !strings.Contains(v, "DOCKER COMPOSE PROJECTS") {
			t.Errorf("expected 'DOCKER COMPOSE PROJECTS' in View, got: %s", v)
		}
		if !strings.Contains(v, "monitoring-stack") {
			t.Errorf("expected compose project name in View, got: %s", v)
		}
		if !strings.Contains(v, "[Running]") {
			t.Errorf("expected '[Running]' status in View, got: %s", v)
		}
	})

	t.Run("TabNetworks rendering with networks and error state", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeTab = TabNetworks
		m.networks = []monitor.DockerNetwork{
			{
				ID:     "net12345",
				Name:   "app-tier",
				Driver: "bridge",
				Scope:  "local",
			},
		}

		v := m.View()
		if !strings.Contains(v, "DOCKER NETWORKS") {
			t.Errorf("expected 'DOCKER NETWORKS' in View, got: %s", v)
		}
		if !strings.Contains(v, "app-tier") {
			t.Errorf("expected network name 'app-tier' in View, got: %s", v)
		}

		// Error state
		m.networkErr = "networks inspect failed"
		vErr := m.View()
		if !strings.Contains(vErr, "Docker networks are unavailable: networks inspect failed") {
			t.Errorf("expected network error in View, got: %s", vErr)
		}
	})

	t.Run("TabVolumes rendering with volumes and error state", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeTab = TabVolumes
		m.volumes = []monitor.DockerVolume{
			{
				Name:       "postgres_data",
				Driver:     "local",
				Scope:      "local",
				Mountpoint: "/var/lib/docker/volumes/postgres_data/_data",
			},
		}

		v := m.View()
		if !strings.Contains(v, "DOCKER VOLUMES") {
			t.Errorf("expected 'DOCKER VOLUMES' in View, got: %s", v)
		}
		if !strings.Contains(v, "postgres_data") {
			t.Errorf("expected volume name 'postgres_data' in View, got: %s", v)
		}

		// Error state
		m.volumeErr = "volume daemon connection refused"
		vErr := m.View()
		if !strings.Contains(vErr, "Docker volumes are unavailable: volume daemon connection refused") {
			t.Errorf("expected volume error in View, got: %s", vErr)
		}
	})

	t.Run("TabSwarm rendering active cluster and inactive cluster", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeTab = TabSwarm

		// Inactive
		m.swarm = monitor.SwarmSummary{
			IsActive:      false,
			StatusDisplay: "Inactive",
		}
		vInactive := m.View()
		if !strings.Contains(vInactive, "DOCKER SWARM CLUSTER") {
			t.Errorf("expected 'DOCKER SWARM CLUSTER' in View, got: %s", vInactive)
		}
		if !strings.Contains(vInactive, "Status: Inactive") {
			t.Errorf("expected 'Status: Inactive' in View, got: %s", vInactive)
		}

		// Active with services
		m.swarm = monitor.SwarmSummary{
			IsActive:      true,
			StatusDisplay: "Active",
			NodesCount:    3,
			ManagersCount: 1,
			Services: []monitor.SwarmService{
				{
					ID:       "srv123",
					Name:     "web-service",
					Replicas: "3/3",
					Image:    "nginx:latest",
				},
			},
		}
		vActive := m.View()
		if !strings.Contains(vActive, "web-service") {
			t.Errorf("expected 'web-service' in View, got: %s", vActive)
		}
	})
}

func TestViewModel_ModalsRendering(t *testing.T) {
	t.Parallel()

	t.Run("ModalLogs rendering with target, lines and filter", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width = 120
		m.height = 30
		m.activeModal = ModalLogs
		m.logsTargetName = "production-backend"
		m.logsTargetID = "backend123"
		m.logsLines = []string{
			"2026-11-21 20:00:01 INFO Starting application",
			"2026-11-21 20:00:02 WARN Cache miss",
			"2026-11-21 20:00:03 ERROR Connection refused",
		}

		v := m.View()
		if !strings.Contains(v, "production-backend") {
			t.Errorf("expected container name in logs modal, got: %s", v)
		}
		if !strings.Contains(v, "Starting application") {
			t.Errorf("expected log line in logs modal, got: %s", v)
		}

		// Filter active in logs
		m.logsFiltering = true
		m.logsFilterQuery = "ERROR"
		vFiltered := m.View()
		if !strings.Contains(vFiltered, "ERROR") {
			t.Errorf("expected filter query 'ERROR' in logs modal, got: %s", vFiltered)
		}
	})

	t.Run("ModalInspect rendering default target vs custom lines", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width = 120
		m.activeModal = ModalInspect
		m.inspectTarget = monitor.ContainerDetail{
			ID:    "container_inspect_abc",
			Name:  "my-app",
			State: "running",
		}

		vDefault := m.View()
		if !strings.Contains(vDefault, "container_inspect_abc") {
			t.Errorf("expected inspect target in View, got: %s", vDefault)
		}

		// Custom lines inspect
		m.inspectCustomTitle = "DOCKER SYSTEM INFO"
		m.inspectCustomLines = []string{
			"Operating System: Linux",
			"Containers: 12",
			"Images: 45",
		}
		vCustom := m.View()
		if !strings.Contains(vCustom, "DOCKER SYSTEM INFO") {
			t.Errorf("expected custom inspect title, got: %s", vCustom)
		}
		if !strings.Contains(vCustom, "Operating System: Linux") {
			t.Errorf("expected custom inspect lines, got: %s", vCustom)
		}
	})

	t.Run("ModalComposeDown confirmation dialog", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalComposeDown
		m.composeDownTarget = "core-infra"

		v := m.View()
		if !strings.Contains(v, "CONFIRM DOCKER COMPOSE DOWN") {
			t.Errorf("expected 'CONFIRM DOCKER COMPOSE DOWN' in View, got: %s", v)
		}
		if !strings.Contains(v, "core-infra") {
			t.Errorf("expected target 'core-infra' in View, got: %s", v)
		}
		if !strings.Contains(v, "Confirm") {
			t.Errorf("expected 'Confirm' prompt in View, got: %s", v)
		}
	})

	t.Run("ModalRemoveContainer confirmation dialog", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveContainer
		m.removeContainerTargetName = "staging-worker"
		m.removeContainerTargetID = "worker_987654"

		v := m.View()
		if !strings.Contains(v, "CONFIRM CONTAINER REMOVAL") {
			t.Errorf("expected 'CONFIRM CONTAINER REMOVAL' in View, got: %s", v)
		}
		if !strings.Contains(v, "staging-worker") {
			t.Errorf("expected container name in View, got: %s", v)
		}
		if !strings.Contains(v, "worker_987654") {
			t.Errorf("expected container ID in View, got: %s", v)
		}
	})

	t.Run("ModalRemoveImage dialog with in-use warning variants", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveImage
		m.removeImageTargetName = "ubuntu:24.04"
		m.removeImageTargetID = "img_ubuntu_123"

		// Not in use
		m.removeImageTargetInUse = false
		vNotInUse := m.View()
		if !strings.Contains(vNotInUse, "CONFIRM IMAGE REMOVAL") {
			t.Errorf("expected 'CONFIRM IMAGE REMOVAL' in View, got: %s", vNotInUse)
		}
		if !strings.Contains(vNotInUse, "ubuntu:24.04") {
			t.Errorf("expected image name in View, got: %s", vNotInUse)
		}
		if strings.Contains(vNotInUse, "Image is currently in use by containers") {
			t.Errorf("expected no in-use warning when image is unused, got: %s", vNotInUse)
		}

		// In use
		m.removeImageTargetInUse = true
		vInUse := m.View()
		if !strings.Contains(vInUse, "Image is currently in use by containers") {
			t.Errorf("expected in-use warning when image is used, got: %s", vInUse)
		}
	})

	t.Run("ModalExec dialog standard selection vs custom input", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalExec
		m.execTargetName = "redis-cluster"

		// Standard predefined choices
		m.execIsCustomInput = false
		vStd := m.View()
		if !strings.Contains(vStd, "EXECUTE INTERACTIVE SHELL") {
			t.Errorf("expected exec modal title in View, got: %s", vStd)
		}
		if !strings.Contains(vStd, "redis-cluster") {
			t.Errorf("expected container name in exec modal View, got: %s", vStd)
		}
		if !strings.Contains(vStd, "/bin/sh") {
			t.Errorf("expected '/bin/sh' option in View, got: %s", vStd)
		}

		// Custom command input mode
		m.execIsCustomInput = true
		m.execCustomCmd = "redis-cli monitor"
		vCustom := m.View()
		if !strings.Contains(vCustom, "Enter custom command:") {
			t.Errorf("expected 'Enter custom command:' prompt in View, got: %s", vCustom)
		}
		if !strings.Contains(vCustom, "redis-cli monitor") {
			t.Errorf("expected custom command 'redis-cli monitor' in View, got: %s", vCustom)
		}
	})

	t.Run("ModalPrune dialog with usage, error and confirmation states", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalPrune

		// Normal usage
		m.pruneUsage = monitor.DockerDiskUsage{
			TotalSize: 1024 * 1024 * 100,
			TotalRecl: 1024 * 1024 * 30,
		}
		vUsage := m.View()
		if !strings.Contains(vUsage, "CLEANUP & PRUNE DOCKER RESOURCES") {
			t.Errorf("expected 'CLEANUP & PRUNE DOCKER RESOURCES' in View, got: %s", vUsage)
		}
		if !strings.Contains(vUsage, "100.0 MB") {
			t.Errorf("expected TotalSize 100.0 MB in View, got: %s", vUsage)
		}

		// With error
		m.pruneUsageErr = "prune df failed"
		vErr := m.View()
		if !strings.Contains(vErr, "prune df failed") {
			t.Errorf("expected prune error message in View, got: %s", vErr)
		}

		// Confirmation prompt
		m.pruneConfirm = 1
		vConfirm := m.View()
		if !strings.Contains(vConfirm, "Confirm Prune") {
			t.Errorf("expected 'Confirm Prune' in footer when pruneConfirm > 0, got: %s", vConfirm)
		}
	})

	t.Run("Help box rendering", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.showHelp = true

		v := m.View()
		if !strings.Contains(v, "HOTKEYS:") {
			t.Errorf("expected 'HOTKEYS:' in View, got: %s", v)
		}
		if !strings.Contains(v, "NAVIGATION") {
			t.Errorf("expected 'NAVIGATION' section in Help View, got: %s", v)
		}
		if !strings.Contains(v, "Auto-refresh interval:") {
			t.Errorf("expected 'Auto-refresh interval:' in Help View, got: %s", v)
		}
	})
}

func TestViewModel_EmbeddedShellLayout(t *testing.T) {
	t.Parallel()

	t.Run("narrow layout renders embedded shell terminal directly", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width = 100
		m.height = 30
		m.shellContainerName = "alpine-box"
		m.embeddedShell = &fakeEmbeddedShell{render: "TERMINAL_NARROW_VIEW_DATA"}

		v := m.View()
		if !strings.Contains(v, "TERMINAL_NARROW_VIEW_DATA") {
			t.Errorf("expected narrow embedded shell render in View, got: %s", v)
		}
		if !strings.Contains(v, "alpine-box") {
			t.Errorf("expected container name in terminal title, got: %s", v)
		}
	})

	t.Run("wide layout renders horizontal split with tab content and terminal", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width = 160
		m.height = 40
		m.activeTab = TabDocker
		m.shellContainerName = "wide-box"
		m.embeddedShell = &fakeEmbeddedShell{render: "TERMINAL_WIDE_SPLIT_DATA"}

		v := m.View()
		if !strings.Contains(v, "DOCKER CONTAINERS") {
			t.Errorf("expected tab content 'DOCKER CONTAINERS' in wide split View, got: %s", v)
		}
		if !strings.Contains(v, "TERMINAL_WIDE_SPLIT_DATA") {
			t.Errorf("expected embedded terminal render in wide split View, got: %s", v)
		}
	})

	t.Run("modal or help overrides embedded shell layout", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.width = 160
		m.height = 40
		m.embeddedShell = &fakeEmbeddedShell{render: "TERMINAL_HIDDEN_PAYLOAD"}

		// Active modal overrides
		m.activeModal = ModalComposeDown
		m.composeDownTarget = "stack-test"
		vModal := m.View()
		if strings.Contains(vModal, "TERMINAL_HIDDEN_PAYLOAD") {
			t.Errorf("expected embedded shell suppressed while modal is active, got: %s", vModal)
		}
		if !strings.Contains(vModal, "CONFIRM DOCKER COMPOSE DOWN") {
			t.Errorf("expected compose down modal rendered, got: %s", vModal)
		}

		// Help overrides
		m.activeModal = ModalNone
		m.showHelp = true
		vHelp := m.View()
		if strings.Contains(vHelp, "TERMINAL_HIDDEN_PAYLOAD") {
			t.Errorf("expected embedded shell suppressed while help is active, got: %s", vHelp)
		}
		if !strings.Contains(vHelp, "HOTKEYS:") {
			t.Errorf("expected help rendered, got: %s", vHelp)
		}
	})
}

func TestViewModel_FooterVariations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(m *Model)
		wantContains []string
	}{
		{
			name: "default tab view with hotkeys",
			setup: func(m *Model) {
				m.activeTab = TabDocker
			},
			wantContains: []string{"[Q] Quit", "[?] Help", "[/] Search"},
		},
		{
			name: "active filter mode in footer",
			setup: func(m *Model) {
				m.isFiltering = true
				m.filterQuery = "nginx"
			},
			wantContains: []string{"[Enter] Apply Filter", "[Esc] Cancel"},
		},
		{
			name: "modal exec footer options",
			setup: func(m *Model) {
				m.activeModal = ModalExec
			},
			wantContains: []string{"[1-7 / Enter]", "Select Action", "[Esc / Q] Cancel"},
		},
		{
			name: "modal remove container footer options",
			setup: func(m *Model) {
				m.activeModal = ModalRemoveContainer
			},
			wantContains: []string{"[Y]", "Confirm (Remove)", "[N / Esc] Cancel"},
		},
		{
			name: "modal remove image footer options",
			setup: func(m *Model) {
				m.activeModal = ModalRemoveImage
			},
			wantContains: []string{"[Y]", "Confirm (Remove Image)", "[N / Esc] Cancel"},
		},
		{
			name: "embedded shell focused footer",
			setup: func(m *Model) {
				m.embeddedShell = &fakeEmbeddedShell{}
				m.shellFocused = true
			},
			wantContains: []string{"[Ctrl+]]", "Return to Dockpulse", "[Ctrl+D]", "Exit shell"},
		},
		{
			name: "embedded shell unfocused footer",
			setup: func(m *Model) {
				m.embeddedShell = &fakeEmbeddedShell{}
				m.shellFocused = false
			},
			wantContains: []string{"[E]", "Focus terminal", "[Ctrl+X]", "Close terminal"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := newFastTestModel()
			tc.setup(&m)
			v := m.View()

			for _, want := range tc.wantContains {
				if !strings.Contains(v, want) {
					t.Errorf("expected footer to contain %q, but got:\n%s", want, v)
				}
			}
		})
	}
}
