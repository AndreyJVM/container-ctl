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

func TestNetworksHotkeys(t *testing.T) {
	defer clipboard.TestLock()()

	m := newFastNetworksTestModel()
	m.networks = []monitor.DockerNetwork{
		testutil.ANetwork(func(n *monitor.DockerNetwork) {
			n.Name = "production-net"
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
}

func TestNetworksInspectModal(t *testing.T) {
	t.Parallel()

	m := newFastNetworksTestModel()
	m.networks = []monitor.DockerNetwork{
		{ID: "net-xyz-123", Name: "app-net", Driver: "bridge", Subnet: "172.20.0.0/16"},
	}

	// Inspect on Networks tab with Enter
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
}

func TestNetworksNavigationKeys(t *testing.T) {
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
}

func TestNetworksEmptyListSafety(t *testing.T) {
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
}

func TestNetworksRemoveHotkeys(t *testing.T) {
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
}
