package tui

import (
	"fmt"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleImagesToggleUnused() (Model, tea.Cmd) {
	m.onlyUnusedImages = !m.onlyUnusedImages
	m.clampCursor()
	if m.onlyUnusedImages {
		m.showToast("Filter: Unused images only")
	} else {
		m.showToast("Showing all images")
	}
	return m, nil
}

func (m Model) handleImagesRemoveModal() (Model, tea.Cmd) {
	images := m.getFilteredImages()
	if len(images) > 0 && m.imageCursorIdx < len(images) {
		img := images[m.imageCursorIdx]
		displayName := img.Repository
		if img.Tag != "" && img.Tag != "<none>" {
			displayName = fmt.Sprintf("%s:%s", img.Repository, img.Tag)
		}
		m.removeImageTargetID = img.ID
		m.removeImageTargetName = displayName
		m.removeImageTargetInUse = img.InUse
		m.openModal(ModalRemoveImage)
	}
	return m, nil
}

func (m Model) handleImagesCopy() (Model, tea.Cmd) {
	images := m.getFilteredImages()
	if len(images) > 0 && m.imageCursorIdx < len(images) {
		img := images[m.imageCursorIdx]
		name := img.Repository
		if img.Tag != "" && img.Tag != "<none>" {
			name = fmt.Sprintf("%s:%s", img.Repository, img.Tag)
		}
		if err := clipboard.Write(name); err != nil {
			m.showToast(fmt.Sprintf("Clipboard error: %v", err))
		} else {
			m.showToast(fmt.Sprintf("Copied to clipboard: %s", name))
		}
	}
	return m, nil
}
