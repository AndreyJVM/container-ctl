package tui

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func newFastNetworksTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabNetworks
	return m
}

func newFastVolumesTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabVolumes
	return m
}

func newFastSwarmTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabSwarm
	m.showSwarm = true
	return m
}

func TestNetworksInspectAndCopy(t *testing.T) {
	defer clipboard.TestLock()()
	origWriter := clipboard.GetOutputWriter()
	var buf bytes.Buffer
	clipboard.SetOutputWriter(&buf)
	defer func() { clipboard.SetOutputWriter(origWriter) }()

	m := newFastNetworksTestModel()
	m.networks = []monitor.DockerNetwork{
		{
			ID:     "net123",
			Name:   "production-bridge",
			Driver: "bridge",
			Scope:  "local",
		},
	}
	m.networkCursorIdx = 0

	// 1. Inspect network via Enter
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mInspect := res.(Model)
	if mInspect.activeModal != ModalInspect {
		t.Errorf("expected activeModal %s, got %s", ModalInspect, mInspect.activeModal)
	}
	if !strings.Contains(mInspect.inspectCustomTitle, "production-bridge") {
		t.Errorf("expected title with network name, got %s", mInspect.inspectCustomTitle)
	}
	if mInspect.inspectCopyID != "net123" {
		t.Errorf("expected inspectCopyID net123, got %s", mInspect.inspectCopyID)
	}
	if len(mInspect.inspectCustomLines) == 0 {
		t.Errorf("expected inspect lines to be populated")
	}

	// 2. Copy network name via 'y'
	buf.Reset()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mCopy := res.(Model)
	if !strings.Contains(mCopy.actionMessage, "production-bridge") {
		t.Errorf("expected toast with network name, got %s", mCopy.actionMessage)
	}
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("production-bridge"))
	if !strings.Contains(buf.String(), expectedB64) {
		t.Errorf("expected clipboard output to contain base64 %q, got: %q", expectedB64, buf.String())
	}
}

func TestVolumesInspectAndCopy(t *testing.T) {
	defer clipboard.TestLock()()
	origWriter := clipboard.GetOutputWriter()
	var buf bytes.Buffer
	clipboard.SetOutputWriter(&buf)
	defer func() { clipboard.SetOutputWriter(origWriter) }()

	m := newFastVolumesTestModel()
	m.volumes = []monitor.DockerVolume{
		{
			Name:       "db_data",
			Driver:     "local",
			Mountpoint: "/var/lib/docker/volumes/db_data/_data",
		},
	}
	m.volumeCursorIdx = 0

	// 1. Inspect volume via 'i'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	mInspect := res.(Model)
	if mInspect.activeModal != ModalInspect {
		t.Errorf("expected activeModal %s, got %s", ModalInspect, mInspect.activeModal)
	}
	if !strings.Contains(mInspect.inspectCustomTitle, "db_data") {
		t.Errorf("expected title with volume name, got %s", mInspect.inspectCustomTitle)
	}
	if mInspect.inspectCopyID != "db_data" {
		t.Errorf("expected inspectCopyID db_data, got %s", mInspect.inspectCopyID)
	}
	if len(mInspect.inspectCustomLines) == 0 {
		t.Errorf("expected inspect lines to be populated")
	}

	// 2. Copy volume name via 'y'
	buf.Reset()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mCopy := res.(Model)
	if !strings.Contains(mCopy.actionMessage, "db_data") {
		t.Errorf("expected toast with volume name, got %s", mCopy.actionMessage)
	}
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("db_data"))
	if !strings.Contains(buf.String(), expectedB64) {
		t.Errorf("expected clipboard output to contain base64 %q, got: %q", expectedB64, buf.String())
	}
}

func TestSwarmLogsAndCopy(t *testing.T) {
	defer clipboard.TestLock()()
	origWriter := clipboard.GetOutputWriter()
	var buf bytes.Buffer
	clipboard.SetOutputWriter(&buf)
	defer func() { clipboard.SetOutputWriter(origWriter) }()

	m := newFastSwarmTestModel()
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

	// 1. Swarm logs via 'l'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	mLogs := res.(Model)
	if mLogs.activeModal == ModalLogs {
		if mLogs.logsTargetID != "svc123" {
			t.Errorf("expected logsTargetID svc123, got %s", mLogs.logsTargetID)
		}
	} else if !strings.Contains(mLogs.actionMessage, "Logs error") {
		t.Errorf("expected logs modal or error toast, got %q", mLogs.actionMessage)
	}

	// 2. Copy swarm service name via 'y'
	buf.Reset()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mCopy := res.(Model)
	if !strings.Contains(mCopy.actionMessage, "frontend-web") {
		t.Errorf("expected toast with service name, got %s", mCopy.actionMessage)
	}
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("frontend-web"))
	if !strings.Contains(buf.String(), expectedB64) {
		t.Errorf("expected clipboard output to contain base64 %q, got: %q", expectedB64, buf.String())
	}
}

func TestSwarmScalingKeys(t *testing.T) {
	m := newFastSwarmTestModel()
	m.swarm = monitor.SwarmSummary{
		IsActive: true,
		Services: []monitor.SwarmService{
			{
				ID:             "svc-rep",
				Name:           "api-service",
				Mode:           "replicated",
				TargetReplicas: 3,
			},
			{
				ID:             "svc-glob",
				Name:           "node-exporter",
				Mode:           "global",
				TargetReplicas: 1,
			},
		},
	}
	m.swarmCursorIdx = 0

	// 1. Pressing 's' opens ModalSwarmScale
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	mScale := res.(Model)
	if mScale.activeModal != ModalSwarmScale {
		t.Fatalf("expected activeModal %s, got %s", ModalSwarmScale, mScale.activeModal)
	}
	if mScale.swarmScaleTargetID != "svc-rep" || mScale.swarmScaleTargetName != "api-service" {
		t.Errorf("unexpected target info: %s, %s", mScale.swarmScaleTargetID, mScale.swarmScaleTargetName)
	}
	if mScale.swarmScaleCurrent != 3 || mScale.swarmScaleDesired != 3 {
		t.Errorf("expected replicas 3, got current=%d desired=%d", mScale.swarmScaleCurrent, mScale.swarmScaleDesired)
	}

	// 2. Pressing '+' triggers quick scale up command
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	_ = res.(Model)
	if cmd == nil {
		t.Errorf("expected non-nil cmd on '+' quick scale")
	}

	// 3. Pressing '-' triggers quick scale down command
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	_ = res.(Model)
	if cmd == nil {
		t.Errorf("expected non-nil cmd on '-' quick scale")
	}

	// 4. Global service cannot be scaled
	m.swarmCursorIdx = 1
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	mGlobal := res.(Model)
	if mGlobal.activeModal == ModalSwarmScale {
		t.Errorf("did not expect ModalSwarmScale on global service")
	}
	if !strings.Contains(mGlobal.actionMessage, "Cannot scale global") {
		t.Errorf("expected global service error toast, got: %q", mGlobal.actionMessage)
	}

	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	mGlobalQuick := res.(Model)
	if cmd != nil {
		t.Errorf("expected nil cmd when quick scaling global service")
	}
	if !strings.Contains(mGlobalQuick.actionMessage, "Cannot scale global") {
		t.Errorf("expected global service error toast, got: %q", mGlobalQuick.actionMessage)
	}
}

func TestNetworksAndVolumesHotkeys(t *testing.T) {
	defer clipboard.TestLock()()

	m := newFastNetworksTestModel()
	m.networks = []monitor.DockerNetwork{
		testutil.ANetwork(func(n *monitor.DockerNetwork) {
			n.Name = "production-net"
		}),
	}
	m.volumes = []monitor.DockerVolume{
		testutil.AVolume(func(v *monitor.DockerVolume) {
			v.Name = "database-data"
		}),
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
	if m.activeModal != ModalPrune {
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
	if m.activeModal != ModalPrune {
		t.Errorf("expected prune modal opened from Volumes tab, got %s", m.activeModal)
	}
}

func TestNetworksAndVolumesInspectModal(t *testing.T) {
	t.Parallel()

	m := newFastNetworksTestModel()
	m.networks = []monitor.DockerNetwork{
		{ID: "net-xyz-123", Name: "app-net", Driver: "bridge", Subnet: "172.20.0.0/16"},
	}
	m.volumes = []monitor.DockerVolume{
		{Name: "data-volume", Driver: "local", Scope: "local"},
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
	t.Parallel()

	m := newFastSwarmTestModel()
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
	if m.activeModal == ModalLogs {
		if !strings.Contains(m.logsTargetName, "api-svc") {
			t.Errorf("expected logsTargetName to contain 'api-svc', got %q", m.logsTargetName)
		}
	} else if m.actionMessage == "" {
		t.Errorf("expected logs modal or actionMessage on pressing 'l'")
	}
}

func TestResourcesNavigationKeys(t *testing.T) {
	t.Parallel()

	t.Run("networks tab navigation", func(t *testing.T) {
		t.Parallel()
		m := newFastNetworksTestModel()
		m.networks = []monitor.DockerNetwork{
			{Name: "net1"}, {Name: "net2"}, {Name: "net3"},
			{Name: "net4"}, {Name: "net5"}, {Name: "net6"},
		}
		m.networkCursorIdx = 0

		// Down and Up wrapping
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = res.(Model)
		if m.networkCursorIdx != 5 {
			t.Errorf("expected networkCursorIdx=5 after up wrap, got %d", m.networkCursorIdx)
		}

		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = res.(Model)
		if m.networkCursorIdx != 0 {
			t.Errorf("expected networkCursorIdx=0 after down wrap, got %d", m.networkCursorIdx)
		}

		// 'G' / End
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
		m = res.(Model)
		if m.networkCursorIdx != 5 {
			t.Errorf("expected networkCursorIdx=5 after 'G', got %d", m.networkCursorIdx)
		}

		// 'g' / Home
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
		m = res.(Model)
		if m.networkCursorIdx != 0 {
			t.Errorf("expected networkCursorIdx=0 after 'g', got %d", m.networkCursorIdx)
		}

		// PageDown and PageUp
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		m = res.(Model)
		if m.networkCursorIdx != 5 {
			t.Errorf("expected networkCursorIdx=5 after PgDown, got %d", m.networkCursorIdx)
		}

		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
		m = res.(Model)
		if m.networkCursorIdx != 0 {
			t.Errorf("expected networkCursorIdx=0 after PgUp, got %d", m.networkCursorIdx)
		}
	})

	t.Run("volumes tab navigation", func(t *testing.T) {
		t.Parallel()
		m := newFastVolumesTestModel()
		m.volumes = []monitor.DockerVolume{
			{Name: "vol1"}, {Name: "vol2"}, {Name: "vol3"},
			{Name: "vol4"}, {Name: "vol5"}, {Name: "vol6"},
		}
		m.volumeCursorIdx = 0

		// Down and Up wrapping
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = res.(Model)
		if m.volumeCursorIdx != 5 {
			t.Errorf("expected volumeCursorIdx=5 after up wrap, got %d", m.volumeCursorIdx)
		}

		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = res.(Model)
		if m.volumeCursorIdx != 0 {
			t.Errorf("expected volumeCursorIdx=0 after down wrap, got %d", m.volumeCursorIdx)
		}

		// 'G' / End
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
		m = res.(Model)
		if m.volumeCursorIdx != 5 {
			t.Errorf("expected volumeCursorIdx=5 after 'G', got %d", m.volumeCursorIdx)
		}

		// 'g' / Home
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
		m = res.(Model)
		if m.volumeCursorIdx != 0 {
			t.Errorf("expected volumeCursorIdx=0 after 'g', got %d", m.volumeCursorIdx)
		}
	})
}

func TestResourcesActions_EmptyListSafety(t *testing.T) {
	t.Parallel()

	t.Run("empty networks tab hotkeys", func(t *testing.T) {
		t.Parallel()
		m := newFastNetworksTestModel()
		m.networks = nil
		m.networkCursorIdx = 0

		for _, k := range []string{"enter", "i", "I", "y", "Y", "x", "X", "d", "D"} {
			res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			updated := res.(Model)
			if cmd != nil {
				t.Errorf("expected nil cmd for key %q on empty networks", k)
			}
			if updated.activeModal != "" {
				t.Errorf("expected no modal opened for key %q on empty networks", k)
			}
		}
	})

	t.Run("empty volumes tab hotkeys", func(t *testing.T) {
		t.Parallel()
		m := newFastVolumesTestModel()
		m.volumes = nil
		m.volumeCursorIdx = 0

		for _, k := range []string{"enter", "i", "I", "y", "Y", "x", "X", "d", "D"} {
			res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			updated := res.(Model)
			if cmd != nil {
				t.Errorf("expected nil cmd for key %q on empty volumes", k)
			}
			if updated.activeModal != "" {
				t.Errorf("expected no modal opened for key %q on empty volumes", k)
			}
		}
	})

	t.Run("empty swarm services tab hotkeys", func(t *testing.T) {
		t.Parallel()
		m := newFastSwarmTestModel()
		m.swarm.Services = nil
		m.swarmCursorIdx = 0

		for _, k := range []string{"l", "L", "y", "Y"} {
			res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			updated := res.(Model)
			if cmd != nil {
				t.Errorf("expected nil cmd for key %q on empty swarm services", k)
			}
			if updated.activeModal != "" {
				t.Errorf("expected no modal opened for key %q on empty swarm services", k)
			}
		}
	})
}

func TestNetworksAndVolumesRemoveHotkeys(t *testing.T) {
	t.Parallel()

	t.Run("networks tab opens remove modal on x and d", func(t *testing.T) {
		t.Parallel()
		for _, k := range []string{"x", "X", "d", "D"} {
			m := newFastNetworksTestModel()
			m.networks = []monitor.DockerNetwork{
				{
					ID:         "net-abc-123",
					Name:       "production-net",
					Containers: 3,
				},
			}
			m.networkCursorIdx = 0

			res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			updated := res.(Model)
			if updated.activeModal != ModalRemoveNetwork {
				t.Errorf("key %q: expected activeModal %s, got %s", k, ModalRemoveNetwork, updated.activeModal)
			}
			if updated.removeNetworkTargetID != "net-abc-123" {
				t.Errorf("key %q: expected removeNetworkTargetID 'net-abc-123', got %s", k, updated.removeNetworkTargetID)
			}
			if updated.removeNetworkTargetName != "production-net" {
				t.Errorf("key %q: expected removeNetworkTargetName 'production-net', got %s", k, updated.removeNetworkTargetName)
			}
			if updated.removeNetworkContainersCount != 3 {
				t.Errorf("key %q: expected removeNetworkContainersCount 3, got %d", k, updated.removeNetworkContainersCount)
			}
		}
	})

	t.Run("volumes tab opens remove modal on x and d", func(t *testing.T) {
		t.Parallel()
		for _, k := range []string{"x", "X", "d", "D"} {
			m := newFastVolumesTestModel()
			m.volumes = []monitor.DockerVolume{
				{
					Name:       "postgres_data",
					RefCount:   1,
					Containers: []string{"db"},
				},
			}
			m.volumeCursorIdx = 0

			res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
			updated := res.(Model)
			if updated.activeModal != ModalRemoveVolume {
				t.Errorf("key %q: expected activeModal %s, got %s", k, ModalRemoveVolume, updated.activeModal)
			}
			if updated.removeVolumeTargetName != "postgres_data" {
				t.Errorf("key %q: expected removeVolumeTargetName 'postgres_data', got %s", k, updated.removeVolumeTargetName)
			}
			if !updated.removeVolumeTargetInUse {
				t.Errorf("key %q: expected removeVolumeTargetInUse true, got false", k)
			}
		}
	})
}
