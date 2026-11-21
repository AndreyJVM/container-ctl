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

func TestFilterModeKeys_ImagesTab(t *testing.T) {
	m := NewModel(2*time.Second, int(TabImages), "v0.9.0")
	m.images = []monitor.DockerImage{
		testutil.AnImage(testutil.WithImageID("img1", "sha256:111"), testutil.WithRepoTag("alpine", "latest")),
		testutil.AnImage(testutil.WithImageID("img2", "sha256:222"), testutil.WithRepoTag("ubuntu", "22.04")),
		testutil.AnImage(testutil.WithImageID("img3", "sha256:333"), testutil.WithRepoTag("redis", "alpine")),
	}

	// 1. Activate search mode on images tab with '/'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = res.(Model)
	if !m.isFiltering {
		t.Fatalf("expected isFiltering=true after '/' on TabImages")
	}

	// 2. Type "alpine"
	for _, r := range []rune("alpine") {
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}
	if m.filterQuery != "alpine" {
		t.Fatalf("expected filterQuery='alpine', got %q", m.filterQuery)
	}

	filtered := m.getFilteredImages()
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered images matching 'alpine', got %d", len(filtered))
	}

	// 3. Backspace
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.filterQuery != "alpin" {
		t.Fatalf("expected filterQuery='alpin' after backspace, got %q", m.filterQuery)
	}

	// 4. Enter to apply filter and exit typing mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.isFiltering {
		t.Fatalf("expected isFiltering=false after Enter")
	}
	if m.filterQuery != "alpin" {
		t.Fatalf("expected filterQuery to persist after Enter")
	}

	// 5. Esc in normal mode clears filter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.filterQuery != "" {
		t.Fatalf("expected filterQuery='' after Esc, got %q", m.filterQuery)
	}
}

func TestFilterUnusedImages_ToggleAndSearch(t *testing.T) {
	m := NewModel(2*time.Second, int(TabImages), "v0.13.0")
	m.images = []monitor.DockerImage{
		testutil.AnImage(testutil.WithImageID("img1", "sha256:111"), testutil.WithRepoTag("alpine", "latest"), testutil.WithImageStatus(true, false, 1)),
		testutil.AnImage(testutil.WithImageID("img2", "sha256:222"), testutil.WithRepoTag("ubuntu", "22.04"), testutil.WithImageStatus(false, false, 0)),
		testutil.AnImage(testutil.WithImageID("img3", "sha256:333"), testutil.WithRepoTag("<none>", "<none>"), testutil.WithImageStatus(false, true, 0)),
	}

	// 1. Initial state: all 3 images returned
	if got := len(m.getFilteredImages()); got != 3 {
		t.Fatalf("expected 3 images initially, got %d", got)
	}

	// 2. Press 'u' to toggle only unused images
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = res.(Model)
	if !m.IsOnlyUnusedImages() {
		t.Fatalf("expected IsOnlyUnusedImages=true after pressing 'u'")
	}
	filtered := m.getFilteredImages()
	if len(filtered) != 2 {
		t.Fatalf("expected 2 unused images, got %d", len(filtered))
	}
	for _, img := range filtered {
		if img.InUse {
			t.Fatalf("expected unused image, got in-use image %s", img.Repository)
		}
	}

	// 3. Press 'esc' to reset unused-only filter
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.IsOnlyUnusedImages() {
		t.Fatalf("expected IsOnlyUnusedImages=false after pressing 'esc'")
	}
	if got := len(m.getFilteredImages()); got != 3 {
		t.Fatalf("expected 3 images after resetting filter, got %d", got)
	}

	// 4. Search query "unused" matches unused images
	m.filterQuery = "unused"
	if got := len(m.getFilteredImages()); got != 2 {
		t.Fatalf("expected 2 images matching /unused, got %d", got)
	}

	// 5. Search query "inuse" matches in-use images
	m.filterQuery = "inuse"
	if got := len(m.getFilteredImages()); got != 1 {
		t.Fatalf("expected 1 image matching /inuse, got %d", got)
	}
}

func TestImagesActions_RemoveModalAndCopy(t *testing.T) {
	origWriter := clipboard.OutputWriter
	var buf bytes.Buffer
	clipboard.OutputWriter = &buf
	defer func() { clipboard.OutputWriter = origWriter }()

	m := NewModel(2*time.Second, int(TabImages), "v1.0.0")
	m.images = []monitor.DockerImage{
		{
			ID:         "img-1",
			Repository: "my-app",
			Tag:        "latest",
			InUse:      false,
		},
		{
			ID:         "img-2",
			Repository: "untagged-app",
			Tag:        "<none>",
			InUse:      true,
		},
	}
	m.imageCursorIdx = 0

	// 1. Remove modal on tagged image
	mNew, _ := m.handleImagesRemoveModal()
	if mNew.activeModal != ModalRemoveImage {
		t.Errorf("expected activeModal %s, got %s", ModalRemoveImage, mNew.activeModal)
	}
	if mNew.removeImageTargetName != "my-app:latest" {
		t.Errorf("expected remove target 'my-app:latest', got %s", mNew.removeImageTargetName)
	}

	// 2. Remove modal on untagged image
	m.imageCursorIdx = 1
	mUntagged, _ := m.handleImagesRemoveModal()
	if mUntagged.removeImageTargetName != "untagged-app" {
		t.Errorf("expected remove target 'untagged-app', got %s", mUntagged.removeImageTargetName)
	}

	// 3. Copy image name to clipboard
	m.imageCursorIdx = 0
	buf.Reset()
	mCopy, _ := m.handleImagesCopy()
	if !strings.Contains(mCopy.actionMessage, "my-app:latest") {
		t.Errorf("expected toast with image name, got: %s", mCopy.actionMessage)
	}

	// 4. Toggle unused images messages
	mToggle, _ := m.handleImagesToggleUnused()
	if !strings.Contains(mToggle.actionMessage, "Unused images only") {
		t.Errorf("expected unused filter toast, got: %s", mToggle.actionMessage)
	}
	mToggle2, _ := mToggle.handleImagesToggleUnused()
	if !strings.Contains(mToggle2.actionMessage, "Showing all images") {
		t.Errorf("expected showing all images toast, got: %s", mToggle2.actionMessage)
	}
}
