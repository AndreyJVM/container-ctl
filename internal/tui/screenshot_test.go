package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/dockpulse/internal/monitor"
)

func TestGenerateScreenshotDumps(t *testing.T) {
	if os.Getenv("GENERATE_SCREENSHOTS") != "1" {
		t.Skip("skipping screenshot generation unless GENERATE_SCREENSHOTS=1")
	}

	outDir := "/tmp/dockpulse_screens"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create out dir: %v", err)
	}

	data := fetchAllData()

	m := NewModel(2*time.Second, 1, "v0.13.0")
	m.width = 112
	m.height = 32
	m.dockerInfo = data.dockerInfo
	m.images = data.images
	m.compose = data.compose
	m.networks = data.networks
	m.volumes = data.volumes
	m.swarm = data.swarm
	m.lastUpdate = data.timestamp

	// Select a container with ports and network info for Docker view
	for i, c := range m.dockerInfo.Containers {
		if strings.Contains(c.Name, "web") {
			m.cursorIdx = i
			break
		}
	}

	// 1. Docker Tab
	m.activeTab = TabDocker
	m.activeModal = ModalNone
	_ = os.WriteFile(filepath.Join(outDir, "docker.txt"), []byte(m.View()), 0644)

	// 2. Images Tab
	m.activeTab = TabImages
	m.cursorIdx = 0
	m.activeModal = ModalNone
	origImages := m.images
	if len(m.images) > 14 {
		m.images = m.images[:14]
	}
	_ = os.WriteFile(filepath.Join(outDir, "images.txt"), []byte(m.View()), 0644)
	m.images = origImages

	// 3. Compose Tab
	m.activeTab = TabCompose
	m.cursorIdx = 0
	m.activeModal = ModalNone
	_ = os.WriteFile(filepath.Join(outDir, "compose.txt"), []byte(m.View()), 0644)

	// 4. Networks Tab
	m.activeTab = TabNetworks
	m.cursorIdx = 0
	m.activeModal = ModalNone
	_ = os.WriteFile(filepath.Join(outDir, "networks.txt"), []byte(m.View()), 0644)

	// 5. Volumes Tab
	m.activeTab = TabVolumes
	m.cursorIdx = 0
	m.activeModal = ModalNone
	_ = os.WriteFile(filepath.Join(outDir, "volumes.txt"), []byte(m.View()), 0644)

	// 6. Swarm Tab (adaptive view forced)
	m.SetShowSwarm(true)
	m.activeTab = TabSwarm
	m.cursorIdx = 0
	m.activeModal = ModalNone
	_ = os.WriteFile(filepath.Join(outDir, "swarm.txt"), []byte(m.View()), 0644)
	m.SetShowSwarm(false)

	// 7. Inspect Modal
	targetContainerID := ""
	for _, c := range m.dockerInfo.Containers {
		if strings.Contains(c.Name, "web") {
			targetContainerID = c.ID
			break
		}
	}
	if targetContainerID == "" && len(m.dockerInfo.Containers) > 0 {
		targetContainerID = m.dockerInfo.Containers[0].ID
	}

	if targetContainerID != "" {
		m.activeTab = TabDocker
		detail, _ := monitor.GetContainerDetail(targetContainerID)
		m.inspectTarget = detail
		m.activeModal = ModalInspect
		_ = os.WriteFile(filepath.Join(outDir, "modal-inspect.txt"), []byte(m.View()), 0644)
	}

	// 8. Logs Modal
	if targetContainerID != "" {
		m.activeTab = TabDocker
		rawLogs, _ := monitor.GetContainerLogs(targetContainerID, 25)
		lines := strings.Split(rawLogs, "\n")
		// Filter out trailing empty line
		var filtered []string
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				filtered = append(filtered, l)
			}
		}
		if len(filtered) == 0 {
			filtered = []string{
				"127.0.0.1 - - [20/Nov/2026:03:28:15 +0000] \"GET / HTTP/1.1\" 200 615 \"-\" \"curl/8.5.0\"",
				"127.0.0.1 - - [20/Nov/2026:03:28:18 +0000] \"GET /health HTTP/1.1\" 200 24 \"-\" \"HealthChecker/1.0\"",
				"127.0.0.1 - - [20/Nov/2026:03:28:22 +0000] \"GET /api/v1/status HTTP/1.1\" 200 182 \"-\" \"dockpulse-agent\"",
				"127.0.0.1 - - [20/Nov/2026:03:28:30 +0000] \"POST /api/v1/metrics HTTP/1.1\" 204 0 \"-\" \"dockpulse-collector\"",
				"127.0.0.1 - - [20/Nov/2026:03:29:00 +0000] \"GET /health HTTP/1.1\" 200 24 \"-\" \"HealthChecker/1.0\"",
			}
		}
		m.logsLines = filtered
		m.logsTargetID = targetContainerID
		m.logsTargetName = "web-gateway"
		m.activeModal = ModalLogs
		_ = os.WriteFile(filepath.Join(outDir, "modal-logs.txt"), []byte(m.View()), 0644)
	}
}
