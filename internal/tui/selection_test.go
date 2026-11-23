package tui

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_SelectionToggleAndSelectAll(t *testing.T) {
	m := newFastTestModel()
	m.activeTab = TabDocker

	containers := []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("app-1")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("app-2")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("app-3")),
	}
	m.dockerInfo = monitor.DockerSummary{
		Status:     "running",
		Containers: containers,
	}
	m.cursorIdx = 0

	// Initially none selected
	if m.isContainerSelected("c1") {
		t.Fatalf("expected c1 not selected initially")
	}

	// Toggle c1 -> selected and advances cursor to 1
	m.toggleContainerSelection()
	if !m.isContainerSelected("c1") {
		t.Errorf("expected c1 selected after toggle")
	}
	if m.cursorIdx != 1 {
		t.Errorf("expected cursor to advance to 1, got %d", m.cursorIdx)
	}

	// Toggle c2 -> selected and advances cursor to 2
	m.toggleContainerSelection()
	if !m.isContainerSelected("c2") {
		t.Errorf("expected c2 selected after toggle")
	}
	if m.cursorIdx != 2 {
		t.Errorf("expected cursor to advance to 2, got %d", m.cursorIdx)
	}

	// Check getSelectedContainers
	selected := m.getSelectedContainers()
	if len(selected) != 2 {
		t.Errorf("expected 2 selected containers, got %d", len(selected))
	}

	// Toggle select-all (since only 2 of 3 selected, should select all)
	m.toggleSelectAllContainers()
	if len(m.getSelectedContainers()) != 3 {
		t.Errorf("expected all 3 containers selected, got %d", len(m.getSelectedContainers()))
	}

	// Toggle select-all again (since all are selected, should deselect all)
	m.toggleSelectAllContainers()
	if len(m.getSelectedContainers()) != 0 {
		t.Errorf("expected all containers deselected, got %d", len(m.getSelectedContainers()))
	}

	// Select c1 again and clear
	m.cursorIdx = 0
	m.toggleContainerSelection()
	if len(m.getSelectedContainers()) != 1 {
		t.Errorf("expected 1 container selected, got %d", len(m.getSelectedContainers()))
	}
	m.clearSelectedContainers()
	if len(m.getSelectedContainers()) != 0 {
		t.Errorf("expected 0 selected after clear, got %d", len(m.getSelectedContainers()))
	}
}

func TestModel_BatchActionHandlers(t *testing.T) {
	m := newFastTestModel()
	m.activeTab = TabDocker

	c1 := testutil.AContainer(testutil.WithID("c1"), testutil.WithName("app-1"), testutil.WithState("stopped", "Exited (0)"))
	c2 := testutil.AContainer(testutil.WithID("c2"), testutil.WithName("app-2"), testutil.WithState("running", "Up 2 hours"))

	m.dockerInfo = monitor.DockerSummary{
		Status:     "running",
		Containers: []monitor.ContainerItem{c1, c2},
	}

	// Select both c1 and c2
	m.selectedContainers = map[string]struct{}{
		"c1": {},
		"c2": {},
	}

	// 1. Batch Start/Stop: since c1 is stopped, it should start
	mStart, cmdStart := m.handleDockerBatchStartStop()
	if cmdStart == nil {
		t.Errorf("expected cmdStart not nil")
	}
	if !strings.Contains(mStart.actionMessage, "Starting 2 selected container(s)") {
		t.Errorf("expected start toast, got: %s", mStart.actionMessage)
	}

	// 2. Batch Start/Stop when all are running: should stop
	mRunning := m
	mRunning.selectedContainers = map[string]struct{}{
		"c2": {},
	}
	mStop, cmdStop := mRunning.handleDockerBatchStartStop()
	if cmdStop == nil {
		t.Errorf("expected cmdStop not nil")
	}
	if !strings.Contains(mStop.actionMessage, "Stopping 1 selected container(s)") {
		t.Errorf("expected stop toast, got: %s", mStop.actionMessage)
	}

	// 3. Batch Restart
	mRestart, cmdRestart := m.handleDockerBatchRestart()
	if cmdRestart == nil {
		t.Errorf("expected cmdRestart not nil")
	}
	if !strings.Contains(mRestart.actionMessage, "Restarting 2 selected container(s)") {
		t.Errorf("expected restart toast, got: %s", mRestart.actionMessage)
	}

	// 4. Batch Kill
	mKill, cmdKill := m.handleDockerBatchKill()
	if cmdKill == nil {
		t.Errorf("expected cmdKill not nil")
	}
	if !strings.Contains(mKill.actionMessage, "Force killing 2 selected container(s)") {
		t.Errorf("expected kill toast, got: %s", mKill.actionMessage)
	}

	// 5. Batch Remove Modal
	mRemove, _ := m.handleDockerBatchRemoveModal()
	if mRemove.activeModal != ModalRemoveContainer {
		t.Errorf("expected ModalRemoveContainer, got: %s", mRemove.activeModal)
	}
	if len(mRemove.removeContainerBatch) != 2 {
		t.Errorf("expected 2 containers in removeContainerBatch, got %d", len(mRemove.removeContainerBatch))
	}
}

func TestModel_MultiSelectKeyBindings(t *testing.T) {
	m := newFastTestModel()
	m.activeTab = TabDocker

	c1 := testutil.AContainer(testutil.WithID("c1"), testutil.WithName("srv-1"), testutil.WithState("stopped", "Exited (0)"))
	c2 := testutil.AContainer(testutil.WithID("c2"), testutil.WithName("srv-2"), testutil.WithState("running", "Up 1 hour"))
	m.dockerInfo = monitor.DockerSummary{
		Status:     "running",
		Containers: []monitor.ContainerItem{c1, c2},
	}
	m.cursorIdx = 0

	// 1. Press Space -> select c1
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if !m.isContainerSelected("c1") {
		t.Errorf("expected c1 selected with Space")
	}

	// 2. Press 'a' -> select all
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = res.(Model)
	if len(m.getSelectedContainers()) != 2 {
		t.Errorf("expected 2 selected with 'a', got %d", len(m.getSelectedContainers()))
	}

	// 3. Press 'x' -> opens ModalRemoveContainer in batch mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = res.(Model)
	if m.activeModal != ModalRemoveContainer || len(m.removeContainerBatch) != 2 {
		t.Errorf("expected ModalRemoveContainer with 2 batch items, got modal %s, batch len %d", m.activeModal, len(m.removeContainerBatch))
	}

	// 4. In modal, press 'y' -> triggers batch remove command and clears selection
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected batch remove cmd on 'y'")
	}
	if len(m.selectedContainers) != 0 {
		t.Errorf("expected selectedContainers cleared after remove confirm")
	}
	if m.activeModal != ModalNone {
		t.Errorf("expected modal closed, got %s", m.activeModal)
	}

	// 5. Press Space on c1, then Esc -> clears selection
	m.cursorIdx = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if !m.isContainerSelected("c1") {
		t.Errorf("expected c1 selected")
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if len(m.selectedContainers) != 0 {
		t.Errorf("expected selection cleared on Esc")
	}
}

func TestModel_ImagesSelectionAndBatchRemove(t *testing.T) {
	m := newFastTestModel()
	m.activeTab = TabImages

	images := []monitor.DockerImage{
		{ID: "img1", Repository: "redis", Tag: "alpine", InUse: false},
		{ID: "img2", Repository: "postgres", Tag: "16", InUse: true},
		{ID: "img3", Repository: "nginx", Tag: "latest", InUse: false},
	}
	m.images = images
	m.imageCursorIdx = 0

	// 1. Space toggles selection and advances cursor
	m.toggleImageSelection()
	if _, ok := m.selectedImages["img1"]; !ok {
		t.Errorf("expected img1 selected")
	}
	if m.imageCursorIdx != 1 {
		t.Errorf("expected cursor advanced to 1, got %d", m.imageCursorIdx)
	}

	// 2. Select All
	m.toggleSelectAllImages()
	if len(m.selectedImages) != 3 {
		t.Errorf("expected all 3 images selected, got %d", len(m.selectedImages))
	}

	// 3. Deselect All
	m.toggleSelectAllImages()
	if len(m.selectedImages) != 0 {
		t.Errorf("expected all images deselected, got %d", len(m.selectedImages))
	}

	// 4. Select img1 & img2 via Space keys
	m.imageCursorIdx = 0
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = res.(Model)
	if len(m.selectedImages) != 2 {
		t.Errorf("expected 2 images selected, got %d", len(m.selectedImages))
	}

	// 5. Press 'x' -> opens ModalRemoveImage in batch mode
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = res.(Model)
	if m.activeModal != ModalRemoveImage || len(m.removeImageBatch) != 2 {
		t.Errorf("expected ModalRemoveImage with 2 batch items, got modal %s, batch len %d", m.activeModal, len(m.removeImageBatch))
	}

	// 6. In modal, press 'y' -> triggers batch remove command and clears selection
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected batch remove cmd on 'y'")
	}
	if len(m.selectedImages) != 0 {
		t.Errorf("expected selectedImages cleared after remove confirm")
	}
	if m.activeModal != ModalNone {
		t.Errorf("expected modal closed, got %s", m.activeModal)
	}

	// 7. Clear via Esc
	m.imageCursorIdx = 0
	m.toggleImageSelection()
	if len(m.selectedImages) != 1 {
		t.Errorf("expected 1 image selected")
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if len(m.selectedImages) != 0 {
		t.Errorf("expected selection cleared on Esc")
	}
}

func TestModel_NetworksSelectionAndBatchRemove(t *testing.T) {
	m := newFastTestModel()
	m.activeTab = TabNetworks

	networks := []monitor.DockerNetwork{
		{ID: "net1", Name: "frontend", Containers: 0},
		{ID: "net2", Name: "backend", Containers: 1},
	}
	m.networks = networks
	m.networkCursorIdx = 0

	// 1. Space toggles selection and advances cursor
	m.toggleNetworkSelection()
	if _, ok := m.selectedNetworks["net1"]; !ok {
		t.Errorf("expected net1 selected")
	}
	if m.networkCursorIdx != 1 {
		t.Errorf("expected cursor advanced to 1, got %d", m.networkCursorIdx)
	}

	// 2. Select all
	m.toggleSelectAllNetworks()
	if len(m.selectedNetworks) != 2 {
		t.Errorf("expected 2 networks selected, got %d", len(m.selectedNetworks))
	}

	// 3. Press 'd' -> opens ModalRemoveNetwork in batch mode
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = res.(Model)
	if m.activeModal != ModalRemoveNetwork || len(m.removeNetworkBatch) != 2 {
		t.Errorf("expected ModalRemoveNetwork with 2 batch items, got modal %s, batch len %d", m.activeModal, len(m.removeNetworkBatch))
	}

	// 4. Confirm remove with 'y'
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected batch remove cmd on 'y'")
	}
	if len(m.selectedNetworks) != 0 {
		t.Errorf("expected selectedNetworks cleared after confirm")
	}

	// 5. Esc clears selection
	m.networkCursorIdx = 0
	m.toggleNetworkSelection()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if len(m.selectedNetworks) != 0 {
		t.Errorf("expected selection cleared on Esc")
	}
}

func TestModel_VolumesSelectionAndBatchRemove(t *testing.T) {
	m := newFastTestModel()
	m.activeTab = TabVolumes

	volumes := []monitor.DockerVolume{
		{Name: "vol-data", Driver: "local", RefCount: 0},
		{Name: "vol-logs", Driver: "local", RefCount: 1},
	}
	m.volumes = volumes
	m.volumeCursorIdx = 0

	// 1. Space toggles selection and advances cursor
	m.toggleVolumeSelection()
	if _, ok := m.selectedVolumes["vol-data"]; !ok {
		t.Errorf("expected vol-data selected")
	}
	if m.volumeCursorIdx != 1 {
		t.Errorf("expected cursor advanced to 1, got %d", m.volumeCursorIdx)
	}

	// 2. Select all
	m.toggleSelectAllVolumes()
	if len(m.selectedVolumes) != 2 {
		t.Errorf("expected 2 volumes selected, got %d", len(m.selectedVolumes))
	}

	// 3. Press 'x' -> opens ModalRemoveVolume in batch mode
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = res.(Model)
	if m.activeModal != ModalRemoveVolume || len(m.removeVolumeBatch) != 2 {
		t.Errorf("expected ModalRemoveVolume with 2 batch items, got modal %s, batch len %d", m.activeModal, len(m.removeVolumeBatch))
	}

	// 4. Confirm remove with 'y'
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected batch remove cmd on 'y'")
	}
	if len(m.selectedVolumes) != 0 {
		t.Errorf("expected selectedVolumes cleared after confirm")
	}

	// 5. Esc clears selection
	m.volumeCursorIdx = 0
	m.toggleVolumeSelection()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if len(m.selectedVolumes) != 0 {
		t.Errorf("expected selection cleared on Esc")
	}
}
