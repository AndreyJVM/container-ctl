package tui

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func newFastSwarmTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabSwarm
	m.showSwarm = true
	return m
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

func TestSwarmEmptyListSafety(t *testing.T) {
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
}
