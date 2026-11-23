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

func newFastImagesTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabImages
	return m
}

func TestFilterModeKeys_ImagesTab(t *testing.T) {
	t.Parallel()

	m := newFastImagesTestModel()
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
	t.Parallel()

	m := newFastImagesTestModel()
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

	// 6. Search query "dangling" matches dangling images
	m.filterQuery = "dangling"
	if got := len(m.getFilteredImages()); got != 1 || m.getFilteredImages()[0].Repository != "<none>" {
		t.Fatalf("expected 1 dangling image matching /dangling, got %d", len(m.getFilteredImages()))
	}
}

func TestImagesActions_RemoveModalAndCopy(t *testing.T) {
	defer clipboard.TestLock()()
	origWriter := clipboard.GetOutputWriter()
	var buf bytes.Buffer
	clipboard.SetOutputWriter(&buf)
	defer func() { clipboard.SetOutputWriter(origWriter) }()

	m := newFastImagesTestModel()
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

	// 1. Remove modal on tagged image via 'x'
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	mNew := res.(Model)
	if mNew.activeModal != ModalRemoveImage {
		t.Errorf("expected activeModal %s, got %s", ModalRemoveImage, mNew.activeModal)
	}
	if mNew.removeImageTargetName != "my-app:latest" {
		t.Errorf("expected remove target 'my-app:latest', got %s", mNew.removeImageTargetName)
	}
	if mNew.removeImageTargetInUse {
		t.Errorf("expected removeImageTargetInUse=false for img-1")
	}

	// Cancel removal modal with 'n'
	res, _ = mNew.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	mCanceled := res.(Model)
	if mCanceled.activeModal != "" {
		t.Errorf("expected modal closed after 'n', got %s", mCanceled.activeModal)
	}

	// 2. Remove modal on untagged image via 'd'
	m.imageCursorIdx = 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mUntagged := res.(Model)
	if mUntagged.removeImageTargetName != "untagged-app" {
		t.Errorf("expected remove target 'untagged-app', got %s", mUntagged.removeImageTargetName)
	}
	if !mUntagged.removeImageTargetInUse {
		t.Errorf("expected removeImageTargetInUse=true for in-use image")
	}

	// Confirm removal with 'y'
	res, cmd := mUntagged.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mConfirmed := res.(Model)
	if mConfirmed.activeModal != "" {
		t.Errorf("expected modal closed after 'y'")
	}
	if cmd == nil {
		t.Errorf("expected non-nil cmd for removeImageCmd")
	}
	if !strings.Contains(mConfirmed.actionMessage, "Removing image untagged-app") {
		t.Errorf("expected toast 'Removing image untagged-app', got: %s", mConfirmed.actionMessage)
	}

	// 3. Copy image name to clipboard via 'y'
	m.imageCursorIdx = 0
	buf.Reset()
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mCopy := res.(Model)
	if cmd != nil {
		t.Errorf("expected nil cmd for clipboard copy")
	}
	if !strings.Contains(mCopy.actionMessage, "Copied to clipboard: my-app:latest") {
		t.Errorf("expected toast with image name, got: %s", mCopy.actionMessage)
	}
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("my-app:latest"))
	if !strings.Contains(buf.String(), expectedB64) {
		t.Errorf("expected clipboard output to contain base64 %q, got: %q", expectedB64, buf.String())
	}

	// 4. Copy untagged image name (omits <none> tag)
	m.imageCursorIdx = 1
	buf.Reset()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	mCopyUntagged := res.(Model)
	if !strings.Contains(mCopyUntagged.actionMessage, "Copied to clipboard: untagged-app") {
		t.Errorf("expected toast with 'untagged-app', got: %s", mCopyUntagged.actionMessage)
	}

	// 5. Toggle unused images messages via 'u' and 'U'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	mToggle := res.(Model)
	if !strings.Contains(mToggle.actionMessage, "Filter: Unused images only") {
		t.Errorf("expected unused filter toast, got: %s", mToggle.actionMessage)
	}

	res, _ = mToggle.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'U'}})
	mToggle2 := res.(Model)
	if !strings.Contains(mToggle2.actionMessage, "Showing all images") {
		t.Errorf("expected showing all images toast, got: %s", mToggle2.actionMessage)
	}
}

func TestImagesNavigationKeys(t *testing.T) {
	t.Parallel()

	m := newFastImagesTestModel()
	m.images = []monitor.DockerImage{
		testutil.AnImage(testutil.WithImageID("img1", "sha:1"), testutil.WithRepoTag("app1", "v1")),
		testutil.AnImage(testutil.WithImageID("img2", "sha:2"), testutil.WithRepoTag("app2", "v2")),
		testutil.AnImage(testutil.WithImageID("img3", "sha:3"), testutil.WithRepoTag("app3", "v3")),
		testutil.AnImage(testutil.WithImageID("img4", "sha:4"), testutil.WithRepoTag("app4", "v4")),
		testutil.AnImage(testutil.WithImageID("img5", "sha:5"), testutil.WithRepoTag("app5", "v5")),
		testutil.AnImage(testutil.WithImageID("img6", "sha:6"), testutil.WithRepoTag("app6", "v6")),
	}
	m.imageCursorIdx = 0

	// Down moves cursor
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.imageCursorIdx != 1 {
		t.Errorf("expected imageCursorIdx=1 after down, got %d", m.imageCursorIdx)
	}

	// Up moves cursor back
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.imageCursorIdx != 0 {
		t.Errorf("expected imageCursorIdx=0 after up, got %d", m.imageCursorIdx)
	}

	// Up wraps to last item
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.imageCursorIdx != 5 {
		t.Errorf("expected imageCursorIdx=5 after up wrap, got %d", m.imageCursorIdx)
	}

	// Down wraps to first item
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.imageCursorIdx != 0 {
		t.Errorf("expected imageCursorIdx=0 after down wrap, got %d", m.imageCursorIdx)
	}

	// 'G' / End moves to last item
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = res.(Model)
	if m.imageCursorIdx != 5 {
		t.Errorf("expected imageCursorIdx=5 after 'G', got %d", m.imageCursorIdx)
	}

	// 'g' / Home moves to first item
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = res.(Model)
	if m.imageCursorIdx != 0 {
		t.Errorf("expected imageCursorIdx=0 after 'g', got %d", m.imageCursorIdx)
	}

	// PgDown moves by 5
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = res.(Model)
	if m.imageCursorIdx != 5 {
		t.Errorf("expected imageCursorIdx=5 after PgDown, got %d", m.imageCursorIdx)
	}

	// PgUp moves by 5
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = res.(Model)
	if m.imageCursorIdx != 0 {
		t.Errorf("expected imageCursorIdx=0 after PgUp, got %d", m.imageCursorIdx)
	}
}

func TestImagesActions_EmptyListSafety(t *testing.T) {
	t.Parallel()

	m := newFastImagesTestModel()
	m.images = nil
	m.imageCursorIdx = 0

	// None of the image hotkeys should panic on empty list
	actionKeys := []string{"u", "U", "x", "X", "d", "D", "y", "Y"}
	for _, k := range actionKeys {
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd for key %q on empty images list", k)
		}
		if updated.activeModal != "" {
			t.Errorf("expected no modal opened for key %q on empty images list", k)
		}
	}
}
