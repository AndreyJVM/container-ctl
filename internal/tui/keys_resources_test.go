package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNetworksInspectAndCopy(t *testing.T) {
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, int(TabNetworks), "v1.0.0")
	m.networks = []monitor.DockerNetwork{
		{
			ID:     "net123",
			Name:   "production-bridge",
			Driver: "bridge",
			Scope:  "local",
		},
	}
	m.networkCursorIdx = 0

	// 1. Inspect network
	mInspect, _ := m.handleNetworksInspect()
	if mInspect.activeModal != ModalInspect {
		t.Errorf("expected activeModal %s, got %s", ModalInspect, mInspect.activeModal)
	}
	if !strings.Contains(mInspect.inspectCustomTitle, "production-bridge") {
		t.Errorf("expected title with network name, got %s", mInspect.inspectCustomTitle)
	}
	if mInspect.inspectCopyID != "net123" {
		t.Errorf("expected inspectCopyID net123, got %s", mInspect.inspectCopyID)
	}

	// 2. Copy network name
	buf.Reset()
	mCopy, _ := m.handleNetworksCopy()
	if !strings.Contains(mCopy.actionMessage, "production-bridge") {
		t.Errorf("expected toast with network name, got %s", mCopy.actionMessage)
	}
}

func TestVolumesInspectAndCopy(t *testing.T) {
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, int(TabVolumes), "v1.0.0")
	m.volumes = []monitor.DockerVolume{
		{
			Name:       "db_data",
			Driver:     "local",
			Mountpoint: "/var/lib/docker/volumes/db_data/_data",
		},
	}
	m.volumeCursorIdx = 0

	// 1. Inspect volume
	mInspect, _ := m.handleVolumesInspect()
	if mInspect.activeModal != ModalInspect {
		t.Errorf("expected activeModal %s, got %s", ModalInspect, mInspect.activeModal)
	}
	if !strings.Contains(mInspect.inspectCustomTitle, "db_data") {
		t.Errorf("expected title with volume name, got %s", mInspect.inspectCustomTitle)
	}
	if mInspect.inspectCopyID != "db_data" {
		t.Errorf("expected inspectCopyID db_data, got %s", mInspect.inspectCopyID)
	}

	// 2. Copy volume name
	buf.Reset()
	mCopy, _ := m.handleVolumesCopy()
	if !strings.Contains(mCopy.actionMessage, "db_data") {
		t.Errorf("expected toast with volume name, got %s", mCopy.actionMessage)
	}
}

func TestSwarmLogsAndCopy(t *testing.T) {
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, int(TabSwarm), "v1.0.0")
	m.swarm = monitor.SwarmSummary{
		IsActive: true,
		Services: []monitor.SwarmService{
			{
				ID:   "svc123",
				Name: "frontend-web",
				Mode: "replicated",
			},
		},
	}
	m.swarmCursorIdx = 0

	// 1. Swarm logs
	mLogs, _ := m.handleSwarmLogs()
	if mLogs.logsTargetID != "svc123" {
		t.Errorf("expected logsTargetID svc123, got %s", mLogs.logsTargetID)
	}

	// 2. Copy swarm service name
	buf.Reset()
	mCopy, _ := m.handleSwarmCopy()
	if !strings.Contains(mCopy.actionMessage, "frontend-web") {
		t.Errorf("expected toast with service name, got %s", mCopy.actionMessage)
	}
}

func TestNetworksAndVolumesHotkeys(t *testing.T) {
	m := Model{
		version:         "v0.18.1",
		activeTab:       TabNetworks,
		refreshInterval: 2 * time.Second,
		networks: []monitor.DockerNetwork{
			testutil.ANetwork(func(n *monitor.DockerNetwork) {
				n.Name = "production-net"
			}),
		},
		volumes: []monitor.DockerVolume{
			testutil.AVolume(func(v *monitor.DockerVolume) {
				v.Name = "database-data"
			}),
		},
	}

	// 1. Copy network name with "y"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard: production-net") {
		t.Errorf("expected copied network name in toast, got: %s", m.actionMessage)
	}

	// 2. Open prune modal from Networks tab with "c"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Errorf("expected prune modal opened from Networks tab, got %s", m.activeModal)
	}
	m.closeModal()

	// 3. Switch to Volumes tab and copy volume name with "Y"
	m.setTab(TabVolumes)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard: database-data") {
		t.Errorf("expected copied volume name in toast, got: %s", m.actionMessage)
	}

	// 4. Open prune modal from Volumes tab with "C"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	m = res.(Model)
	if m.activeModal != "prune" {
		t.Errorf("expected prune modal opened from Volumes tab, got %s", m.activeModal)
	}
}

func TestNetworksAndVolumesInspectModal(t *testing.T) {
	m := Model{
		version:         "v0.18.1",
		activeTab:       TabNetworks,
		refreshInterval: 2 * time.Second,
		networks: []monitor.DockerNetwork{
			{ID: "net-xyz-123", Name: "app-net", Driver: "bridge", Subnet: "172.20.0.0/16"},
		},
		volumes: []monitor.DockerVolume{
			{Name: "data-volume", Driver: "local", Scope: "local"},
		},
	}

	// 1. Inspect on Networks tab with Enter
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.activeModal != ModalInspect {
		t.Fatalf("expected ModalInspect opened for network, got %q", m.activeModal)
	}
	if !strings.Contains(m.inspectCustomTitle, "NETWORK INSPECTION: app-net") {
		t.Errorf("expected network title, got %q", m.inspectCustomTitle)
	}
	if len(m.inspectCustomLines) == 0 {
		t.Errorf("expected inspect lines populated")
	}
	if m.inspectCopyID != "net-xyz-123" {
		t.Errorf("expected copy ID net-xyz-123, got %q", m.inspectCopyID)
	}

	// View renders the network inspection modal
	view := m.View()
	if !strings.Contains(view, "NETWORK INSPECTION: app-net") {
		t.Errorf("expected network inspection header in view, got: %s", view)
	}

	// Test copy hotkey in modal
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard") && !strings.Contains(m.actionMessage, "Clipboard error") {
		t.Errorf("expected clipboard feedback toast, got: %s", m.actionMessage)
	}

	// Close modal with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != ModalNone {
		t.Fatalf("expected modal closed, got %q", m.activeModal)
	}

	// 2. Inspect on Volumes tab with 'i'
	m.activeTab = TabVolumes
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = res.(Model)
	if m.activeModal != ModalInspect {
		t.Fatalf("expected ModalInspect opened for volume, got %q", m.activeModal)
	}
	if !strings.Contains(m.inspectCustomTitle, "VOLUME INSPECTION: data-volume") {
		t.Errorf("expected volume title, got %q", m.inspectCustomTitle)
	}
	if len(m.inspectCustomLines) == 0 {
		t.Errorf("expected inspect lines populated")
	}
	if m.inspectCopyID != "data-volume" {
		t.Errorf("expected copy ID data-volume, got %q", m.inspectCopyID)
	}

	volView := m.View()
	if !strings.Contains(volView, "VOLUME INSPECTION: data-volume") {
		t.Errorf("expected volume inspection header in view, got: %s", volView)
	}

	// Scroll down and up
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)

	// Close modal with Esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != ModalNone {
		t.Fatalf("expected modal closed, got %q", m.activeModal)
	}
}

func TestSwarmTabNavigationAndLogs(t *testing.T) {
	m := NewModel(2*time.Second, int(TabSwarm), "v0.4.1")
	m.swarm = monitor.SwarmSummary{
		IsActive:      true,
		StatusDisplay: "Active",
		NodesCount:    3,
		ManagersCount: 1,
		Services: []monitor.SwarmService{
			{ID: "svc1", Name: "web-svc", Image: "nginx:alpine", Replicas: "3/3"},
			{ID: "svc2", Name: "api-svc", Image: "golang:alpine", Replicas: "1/2"},
		},
	}
	m.swarmCursorIdx = 0

	// Down moves to second service
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.swarmCursorIdx != 1 {
		t.Errorf("expected swarmCursorIdx=1, got %d", m.swarmCursorIdx)
	}

	// Down wraps to 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.swarmCursorIdx != 0 {
		t.Errorf("expected swarmCursorIdx=0 after wrap, got %d", m.swarmCursorIdx)
	}

	// Up wraps to 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.swarmCursorIdx != 1 {
		t.Errorf("expected swarmCursorIdx=1 after up wrap, got %d", m.swarmCursorIdx)
	}

	// Key 'k' moves to 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = res.(Model)
	if m.swarmCursorIdx != 0 {
		t.Errorf("expected swarmCursorIdx=0 after 'k', got %d", m.swarmCursorIdx)
	}

	// Key 'j' moves to 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = res.(Model)
	if m.swarmCursorIdx != 1 {
		t.Errorf("expected swarmCursorIdx=1 after 'j', got %d", m.swarmCursorIdx)
	}

	// View shows cursor marker '► ' and footer hints
	view := m.View()
	if !strings.Contains(view, "► ") {
		t.Errorf("expected cursor marker '► ' in swarm tab view")
	}
	if !strings.Contains(view, "Select Service") || !strings.Contains(view, "Service Logs") {
		t.Errorf("expected swarm footer hints in View(), got: %s", view)
	}

	// Pressing 'L' attempts to fetch logs
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = res.(Model)
	if m.activeModal == "logs" {
		if !strings.Contains(m.logsTargetName, "api-svc") {
			t.Errorf("expected logsTargetName to contain 'api-svc', got %q", m.logsTargetName)
		}
	} else if m.actionMessage == "" {
		t.Errorf("expected logs modal or actionMessage on pressing 'l'")
	}
}
