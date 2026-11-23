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

func newFastVolumesTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabVolumes
	return m
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

func TestVolumesHotkeys(t *testing.T) {
	defer clipboard.TestLock()()

	m := newFastVolumesTestModel()
	m.volumes = []monitor.DockerVolume{
		testutil.AVolume(func(v *monitor.DockerVolume) {
			v.Name = "database-data"
		}),
	}

	// 1. Switch to Volumes tab and copy volume name with "Y"
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard: database-data") {
		t.Errorf("expected copied volume name in toast, got: %s", m.actionMessage)
	}

	// 2. Open prune modal from Volumes tab with "C"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	m = res.(Model)
	if m.activeModal != ModalPrune {
		t.Errorf("expected prune modal opened from Volumes tab, got %s", m.activeModal)
	}
}

func TestVolumesInspectModal(t *testing.T) {
	t.Parallel()

	m := newFastVolumesTestModel()
	m.volumes = []monitor.DockerVolume{
		{Name: "data-volume", Driver: "local", Scope: "local"},
	}

	// Inspect on Volumes tab with 'i'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
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

func TestVolumesNavigationKeys(t *testing.T) {
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
}

func TestVolumesEmptyListSafety(t *testing.T) {
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
}

func TestVolumesRemoveHotkeys(t *testing.T) {
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
}
