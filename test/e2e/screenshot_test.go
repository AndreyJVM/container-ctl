package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
	tea "github.com/charmbracelet/bubbletea"
)

func TestGenerateScreenshotDumps(t *testing.T) {
	if os.Getenv("GENERATE_SCREENSHOTS") != "1" {
		t.Skip("skipping screenshot generation unless GENERATE_SCREENSHOTS=1")
	}

	outDir := "/tmp/container-ctl_screens"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create out dir: %v", err)
	}

	m := tui.NewModel(2*time.Second, 1, "v0.13.0")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 112, Height: 32})
	m = updated.(tui.Model)

	data := actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Status:            "running",
			StatusDisplay:     "Running",
			RunningContainers: 2,
			StoppedContainers: 1,
			Containers: []monitor.ContainerItem{
				testutil.AContainer(testutil.WithID("c1"), testutil.WithName("web-frontend"), testutil.WithImage("nginx:alpine"), testutil.WithState("running", "Up 2 hours")),
				testutil.AContainer(testutil.WithID("c2"), testutil.WithName("api-backend"), testutil.WithImage("golang:alpine"), testutil.WithState("running", "Up 2 hours")),
			},
		},
		Images: []monitor.DockerImage{
			testutil.AnImage(testutil.WithRepoTag("nginx", "alpine")),
			testutil.AnImage(testutil.WithRepoTag("golang", "alpine")),
		},
		Compose: []monitor.ComposeProject{
			{Name: "app-stack", Status: "running", StatusDisplay: "Running", RunningServices: 2, TotalServices: 2},
		},
		Networks: []monitor.DockerNetwork{
			{ID: "net1", Name: "app-net", Driver: "bridge"},
		},
		Volumes: []monitor.DockerVolume{
			{Name: "app-vol", Driver: "local"},
		},
		Swarm: monitor.SwarmSummary{
			IsActive:      true,
			StatusDisplay: "Active",
			NodesCount:    3,
		},
	}
	updated, _ = m.Update(data)
	m = updated.(tui.Model)

	// 1. Docker Tab
	m.SetTab(tui.TabDocker)
	m.CloseModal()
	_ = os.WriteFile(filepath.Join(outDir, "docker.txt"), []byte(m.View()), 0644)

	// 2. Images Tab
	m.SetTab(tui.TabImages)
	_ = os.WriteFile(filepath.Join(outDir, "images.txt"), []byte(m.View()), 0644)

	// 3. Compose Tab
	m.SetTab(tui.TabCompose)
	_ = os.WriteFile(filepath.Join(outDir, "compose.txt"), []byte(m.View()), 0644)

	// 4. Networks Tab
	m.SetTab(tui.TabNetworks)
	_ = os.WriteFile(filepath.Join(outDir, "networks.txt"), []byte(m.View()), 0644)

	// 5. Volumes Tab
	m.SetTab(tui.TabVolumes)
	_ = os.WriteFile(filepath.Join(outDir, "volumes.txt"), []byte(m.View()), 0644)

	// 6. Swarm Tab
	m.SetShowSwarm(true)
	m.SetTab(tui.TabSwarm)
	_ = os.WriteFile(filepath.Join(outDir, "swarm.txt"), []byte(m.View()), 0644)
	m.SetShowSwarm(false)

	// 7. Modals
	m.SetTab(tui.TabDocker)
	m.OpenModal(tui.ModalPrune)
	_ = os.WriteFile(filepath.Join(outDir, "modal_prune.txt"), []byte(m.View()), 0644)
	m.CloseModal()
}
