package tui

import (
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModalConfirmCancelKeys(t *testing.T) {
	t.Parallel()

	cancelKeys := []string{"esc", "q", "n", "N"}
	for _, k := range cancelKeys {
		if !isModalCancelKey(k) {
			t.Errorf("expected isModalCancelKey(%q) to be true", k)
		}
	}

	confirmKeys := []string{"y", "Y"}
	for _, k := range confirmKeys {
		if !isModalConfirmKey(k) {
			t.Errorf("expected isModalConfirmKey(%q) to be true", k)
		}
	}

	otherKeys := []string{"x", "a", "enter", "tab", "1"}
	for _, k := range otherKeys {
		if isModalCancelKey(k) {
			t.Errorf("expected isModalCancelKey(%q) to be false", k)
		}
		if isModalConfirmKey(k) {
			t.Errorf("expected isModalConfirmKey(%q) to be false", k)
		}
	}
}

func TestDirectConfirmModalHandlers(t *testing.T) {
	t.Parallel()

	t.Run("handleComposeDownModalInput cancel", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalComposeDown
		m.composeDownTarget = "my-stack"

		res, cmd := m.handleComposeDownModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd on cancel")
		}
		if updated.activeModal != ModalNone {
			t.Errorf("expected modal closed")
		}
		if updated.composeDownTarget != "" {
			t.Errorf("expected target cleared")
		}
	})

	t.Run("handleComposeDownModalInput confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalComposeDown
		m.composeDownTarget = "my-stack"
		m.dockerInfo = monitor.DockerSummary{
			Containers: []monitor.ContainerItem{{Name: "web", ComposeProject: "my-stack"}},
		}

		res, cmd := m.handleComposeDownModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		updated := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on confirm")
		}
		if updated.activeModal != ModalNone {
			t.Errorf("expected modal closed")
		}
	})

	t.Run("handleRemoveContainerModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveContainer
		m.removeContainerTargetID = "c123"
		m.removeContainerTargetName = "web-container"

		res, cmd := m.handleRemoveContainerModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeContainerTargetID != "" {
			t.Errorf("unexpected state on cancel: %+v", updated)
		}

		m.activeModal = ModalRemoveContainer
		m.removeContainerTargetID = "c123"
		m.removeContainerTargetName = "web-container"
		res, cmd = m.handleRemoveContainerModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})

	t.Run("handleRemoveImageModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveImage
		m.removeImageTargetID = "img123"
		m.removeImageTargetName = "alpine:latest"
		m.removeImageTargetInUse = true

		res, cmd := m.handleRemoveImageModalInput(tea.KeyMsg{Type: tea.KeyEsc})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeImageTargetID != "" {
			t.Errorf("unexpected state on cancel")
		}

		m.activeModal = ModalRemoveImage
		m.removeImageTargetID = "img123"
		m.removeImageTargetName = "alpine:latest"
		m.removeImageTargetInUse = true
		res, cmd = m.handleRemoveImageModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})

	t.Run("handleRemoveNetworkModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveNetwork
		m.removeNetworkTargetID = "net123"
		m.removeNetworkTargetName = "bridge-net"

		res, cmd := m.handleRemoveNetworkModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeNetworkTargetID != "" {
			t.Errorf("unexpected state on cancel")
		}

		m.activeModal = ModalRemoveNetwork
		m.removeNetworkTargetID = "net123"
		m.removeNetworkTargetName = "bridge-net"
		res, cmd = m.handleRemoveNetworkModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})

	t.Run("handleRemoveVolumeModalInput cancel and confirm", func(t *testing.T) {
		t.Parallel()
		m := newFastTestModel()
		m.activeModal = ModalRemoveVolume
		m.removeVolumeTargetName = "vol-data"
		m.removeVolumeTargetInUse = true

		res, cmd := m.handleRemoveVolumeModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
		updated := res.(Model)
		if cmd != nil || updated.activeModal != ModalNone || updated.removeVolumeTargetName != "" {
			t.Errorf("unexpected state on cancel")
		}

		m.activeModal = ModalRemoveVolume
		m.removeVolumeTargetName = "vol-data"
		m.removeVolumeTargetInUse = true
		res, cmd = m.handleRemoveVolumeModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})
		updated = res.(Model)
		if cmd == nil || updated.activeModal != ModalNone {
			t.Errorf("expected valid cmd on confirm")
		}
	})
}
