package tui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/clipboard"
	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// Tests from keys_docker_test.go
// =============================================================================

func newFastDockerTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabDocker
	return m
}

func TestContainerCursorAndModals(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()

	// Populate dummy containers using fixture builder
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(
			testutil.WithID("c1"),
			testutil.WithName("web-server"),
			testutil.WithImage("nginx:latest"),
			testutil.WithMetrics(5.2, 45.0, 375.0),
		),
		testutil.AContainer(
			testutil.WithID("c2"),
			testutil.WithName("database"),
			testutil.WithImage("postgres:16"),
			testutil.WithState("stopped", "exited"),
		),
	}
	m.cursorIdx = 0

	// Cursor navigation
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.cursorIdx != 1 {
		t.Errorf("expected cursorIdx=1 after down, got %d", m.cursorIdx)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.cursorIdx != 0 {
		t.Errorf("expected cursorIdx=0 after up, got %d", m.cursorIdx)
	}

	// Down wrap around
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.cursorIdx != 1 {
		t.Errorf("expected cursorIdx=1 after up wrap, got %d", m.cursorIdx)
	}

	// Open inspect modal
	m.inspectTarget = monitor.ContainerDetail{
		ID:         "c1",
		Name:       "web-server",
		Image:      "nginx:latest",
		State:      "running",
		CPUPerc:    5.2,
		MemUsageMB: 45.0,
	}
	m.activeModal = ModalInspect

	view := m.View()
	if !strings.Contains(view, "CONTAINER INSPECTION") {
		t.Errorf("expected inspect modal title in View(), got %q", view)
	}

	// Close inspect modal with 'q'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal to close on 'q', got %q", m.activeModal)
	}

	// Open logs modal and test scrolling
	m.activeModal = ModalLogs
	m.logsLines = []string{
		"line 1", "line 2", "line 3", "line 4", "line 5",
		"line 6", "line 7", "line 8", "line 9", "line 10",
		"line 11", "line 12", "line 13", "line 14", "line 15",
	}
	m.logsScroll = 0

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = res.(Model)
	if m.logsScroll != 1 {
		t.Errorf("expected logsScroll=1 after down, got %d", m.logsScroll)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = res.(Model)
	if m.logsScroll != 0 {
		t.Errorf("expected logsScroll=0 after up, got %d", m.logsScroll)
	}

	// Page down
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = res.(Model)
	if m.logsScroll == 0 {
		t.Errorf("expected logsScroll to increase on PgDown")
	}

	// Page up
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = res.(Model)
	if m.logsScroll != 0 {
		t.Errorf("expected logsScroll to return to 0 on PgUp, got %d", m.logsScroll)
	}

	// Close logs modal
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal to close on Esc, got %q", m.activeModal)
	}
}

func TestRenderDockerTabFilteringView(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("alpha"), testutil.WithImage("alpine")),
	}

	// Active search input
	m.isFiltering = true
	m.filterQuery = "alp"
	vSearch := m.View()
	if !strings.Contains(vSearch, "[Search: alp█]") {
		t.Errorf("expected search prompt in View(), got: %s", vSearch)
	}

	// Applied filter view
	m.isFiltering = false
	vFilter := m.View()
	if !strings.Contains(vFilter, "Filter: alp") {
		t.Errorf("expected filter badge in View(), got: %s", vFilter)
	}

	// Empty search results
	m.filterQuery = "nonexistent"
	vEmpty := m.View()
	if !strings.Contains(vEmpty, "No containers found") {
		t.Errorf("expected no containers found message in View(), got: %s", vEmpty)
	}
}

func TestDockerTabHealthBadges(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("healthy-app"), testutil.WithImage("nginx"), testutil.WithState("running", "Up"), testutil.WithHealth("healthy")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("broken-app"), testutil.WithImage("postgres"), testutil.WithState("running", "Up"), testutil.WithHealth("unhealthy")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("booting-app"), testutil.WithImage("redis"), testutil.WithState("running", "Up"), testutil.WithHealth("starting")),
		testutil.AContainer(testutil.WithID("c4"), testutil.WithName("plain-app"), testutil.WithImage("busybox"), testutil.WithState("running", "Up"), testutil.WithHealth("")),
	}

	view := m.View()
	if !strings.Contains(view, "STATUS") {
		t.Errorf("expected STATUS column in header, got: %s", view)
	}
	if !strings.Contains(view, "[healthy]") {
		t.Errorf("expected [healthy] badge in View(), got: %s", view)
	}
	if !strings.Contains(view, "[unhealthy]") {
		t.Errorf("expected [unhealthy] badge in View(), got: %s", view)
	}
	if !strings.Contains(view, "[starting]") {
		t.Errorf("expected [starting] badge in View(), got: %s", view)
	}
}

func TestDockerTabSelectedContainerDetails(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(
			testutil.WithID("abcdef123456"),
			testutil.WithName("web-server"),
			testutil.WithImage("nginx:alpine"),
			testutil.WithState("running", "Up 2 hours"),
			testutil.WithPorts("8080->80/tcp, 8443->443/tcp"),
			testutil.WithMounts("/var/www -> /usr/share/nginx/html (ro)", "nginx-logs -> /var/log/nginx (rw)"),
			testutil.WithCompose("my-stack", "web"),
			testutil.WithNetwork("172.18.0.5", "my-network"),
			testutil.WithRestartPolicy("unless-stopped"),
			testutil.WithCreated("2026-11-07 15:30:00"),
			testutil.WithNetworkIO(12*1024*1024, 45*1024*1024),
			testutil.WithBlockIO(10*1024*1024, 20*1024*1024),
		),
	}

	view := m.View()
	if !strings.Contains(view, "SELECTED CONTAINER DETAILS") {
		t.Errorf("expected details panel header in View(), got: %s", view)
	}
	if !strings.Contains(view, "8080 → 80/tcp") || !strings.Contains(view, "8443 → 443/tcp") {
		t.Errorf("expected formatted ports in details panel, got: %s", view)
	}
	if !strings.Contains(view, "172.18.0.5") || !strings.Contains(view, "my-network") {
		t.Errorf("expected IP and network in details panel, got: %s", view)
	}
	if !strings.Contains(view, "unless-stopped") {
		t.Errorf("expected restart policy in details panel, got: %s", view)
	}
	if !strings.Contains(view, "2026-11-07 15:30:00") {
		t.Errorf("expected created timestamp in details panel, got: %s", view)
	}
	if !strings.Contains(view, "/var/www -> /usr/share/nginx/html (ro)") {
		t.Errorf("expected volume mount in details panel, got: %s", view)
	}
	if !strings.Contains(view, "my-stack") {
		t.Errorf("expected compose project in details panel, got: %s", view)
	}
	if !strings.Contains(view, "Net I/O:") || !strings.Contains(view, "12.0 MB / 45.0 MB") {
		t.Errorf("expected Net I/O in details panel, got: %s", view)
	}
	if !strings.Contains(view, "Block I/O:") || !strings.Contains(view, "10.0 MB / 20.0 MB") {
		t.Errorf("expected Block I/O in details panel, got: %s", view)
	}
}

func TestContainerExecHotkeysAndMessages(t *testing.T) {
	originalStart := startEmbeddedShell
	startEmbeddedShell = func(_ *exec.Cmd, _, _ int) (embeddedShellSession, error) {
		return &fakeEmbeddedShell{}, nil
	}
	defer func() { startEmbeddedShell = originalStart }()

	// Test when Docker CLI is not available
	restoreUnavailable := monitor.SetDockerCLIAvailableCheckerForTesting(func() bool { return false })
	mUnavailable := newFastDockerTestModel()
	mUnavailable.dockerInfo = monitor.DockerSummary{
		Status:        "running",
		StatusDisplay: "Running",
		Containers: []monitor.ContainerItem{
			testutil.AContainer(testutil.WithID("c1"), testutil.WithName("running-container"), testutil.WithState("running", "Up 2 hours")),
		},
	}
	mUnavailable.cursorIdx = 0
	resUnavail, _ := mUnavailable.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	mUnavailResult := resUnavail.(Model)
	if !strings.Contains(mUnavailResult.actionMessage, "docker binary not found") {
		t.Errorf("expected docker not found message, got %q", mUnavailResult.actionMessage)
	}
	restoreUnavailable()

	// Test when Docker CLI is available
	restoreAvailable := monitor.SetDockerCLIAvailableCheckerForTesting(func() bool { return true })
	defer restoreAvailable()

	m := newFastDockerTestModel()
	m.dockerInfo = monitor.DockerSummary{
		Status:        "running",
		StatusDisplay: "Running",
		Containers: []monitor.ContainerItem{
			testutil.AContainer(testutil.WithID("c1"), testutil.WithName("running-container"), testutil.WithState("running", "Up 2 hours")),
			testutil.AContainer(testutil.WithID("c2"), testutil.WithName("stopped-container"), testutil.WithState("exited", "Exited (0) 5 mins ago")),
		},
	}
	m.cursorIdx = 0

	// 1. Press 'e' on running container opens exec modal
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m = res.(Model)
	if m.activeModal != ModalExec {
		t.Fatalf("expected activeModal 'exec', got %q", m.activeModal)
	}
	if m.execTargetID != "c1" {
		t.Errorf("expected execTargetID 'c1', got %q", m.execTargetID)
	}
	// Select first option (embedded auto shell) via enter.
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected non-nil shell read and wait command on selecting option")
	}
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	m = res.(Model)

	// 2. Press 'E' on stopped container
	m.activeModal = ""
	m.cursorIdx = 1
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = res.(Model)
	if cmd != nil {
		t.Errorf("expected nil cmd for stopped container, got non-nil")
	}
	if !strings.Contains(m.actionMessage, "Cannot connect") {
		t.Errorf("expected cannot connect message, got %q", m.actionMessage)
	}

	// 3. ExecFinishedMsg with nil error
	res, _ = m.Update(execFinishedMsg{containerName: "running-container", err: nil})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Shell session in running-container ended") {
		t.Errorf("expected shell session finished message, got %q", m.actionMessage)
	}

	// 4. ExecFinishedMsg with error
	res, _ = m.Update(execFinishedMsg{containerName: "running-container", err: fmt.Errorf("shell not found")})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Shell error in running-container") {
		t.Errorf("expected shell error message, got %q", m.actionMessage)
	}

	// 5. Check footer and help hints
	m.activeModal = ""
	view := m.View()
	if !strings.Contains(view, "[E]") || !strings.Contains(view, "Shell") {
		t.Errorf("expected '[E] Shell' in tab footer view, got: %s", view)
	}

	m.showHelp = true
	helpView := m.View()
	if !strings.Contains(helpView, "[E]") || !strings.Contains(helpView, "Interactive shell") {
		t.Errorf("expected [E] and Interactive shell in help modal, got: %s", helpView)
	}
}

func TestContainerSorting(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("zeta-app"), testutil.WithState("running", "Up"), testutil.WithMetrics(10.0, 200.0, 1024.0), testutil.WithHealth("healthy")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("alpha-app"), testutil.WithState("running", "Up"), testutil.WithMetrics(50.0, 50.0, 1024.0), testutil.WithHealth("unhealthy")),
		testutil.AContainer(testutil.WithID("c3"), testutil.WithName("beta-app"), testutil.WithState("stopped", "Exited"), testutil.WithMetrics(0.0, 0.0, 1024.0)),
		testutil.AContainer(testutil.WithID("c4"), testutil.WithName("gamma-app"), testutil.WithState("running", "Up"), testutil.WithMetrics(30.0, 500.0, 1024.0), testutil.WithHealth("starting")),
	}

	// 1. Default order
	list := m.getFilteredContainers()
	if list[0].Name != "zeta-app" || list[1].Name != "alpha-app" {
		t.Errorf("expected default order preserved, got: %s, %s", list[0].Name, list[1].Name)
	}

	// 2. Press O -> CPU desc
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = newModel.(Model)
	if m.containerSortMode != SortCPU {
		t.Fatalf("expected SortCPU, got %d", m.containerSortMode)
	}
	list = m.getFilteredContainers()
	if list[0].Name != "alpha-app" || list[0].CPUPerc != 50.0 {
		t.Errorf("expected top CPU alpha-app, got %s", list[0].Name)
	}
	if list[1].Name != "gamma-app" || list[1].CPUPerc != 30.0 {
		t.Errorf("expected 2nd CPU gamma-app, got %s", list[1].Name)
	}
	view := m.View()
	if !strings.Contains(view, "Sort: CPU ↓") {
		t.Errorf("expected Sort: CPU ↓ in View(), got: %s", view)
	}

	// 3. Press O -> MEM desc
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = newModel.(Model)
	if m.containerSortMode != SortMem {
		t.Fatalf("expected SortMem, got %d", m.containerSortMode)
	}
	list = m.getFilteredContainers()
	if list[0].Name != "gamma-app" || list[0].MemUsageMB != 500.0 {
		t.Errorf("expected top MEM gamma-app, got %s", list[0].Name)
	}

	// 4. Press O -> Name asc
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = newModel.(Model)
	if m.containerSortMode != SortName {
		t.Fatalf("expected SortName, got %d", m.containerSortMode)
	}
	list = m.getFilteredContainers()
	if list[0].Name != "alpha-app" || list[1].Name != "beta-app" || list[2].Name != "gamma-app" || list[3].Name != "zeta-app" {
		t.Errorf("unexpected name sorting order: %v", list)
	}
	testutil.AssertContainsInOrder(t, m.View(), "alpha-app", "beta-app", "gamma-app", "zeta-app")

	// 5. Press O -> Status
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = newModel.(Model)
	if m.containerSortMode != SortStatus {
		t.Fatalf("expected SortStatus, got %d", m.containerSortMode)
	}
	list = m.getFilteredContainers()
	if list[0].Name != "alpha-app" {
		t.Errorf("expected unhealthy alpha-app first, got %s", list[0].Name)
	}
	if list[1].Name != "gamma-app" {
		t.Errorf("expected starting gamma-app second, got %s", list[1].Name)
	}

	// 6. Press O -> Wrap around to Default
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = newModel.(Model)
	if m.containerSortMode != SortDefault {
		t.Fatalf("expected wrap around to SortDefault, got %d", m.containerSortMode)
	}
}

func TestContainerStartStopAndRestartHotkeys(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("api-srv"), testutil.WithState("running", "Up")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("db-srv"), testutil.WithState("exited", "Exited")),
	}

	// 1. 's' on running container -> stops
	m.cursorIdx = 0
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	mStop := res.(Model)
	if cmd == nil {
		t.Errorf("expected non-nil cmd on 's' for running container")
	}
	if !strings.Contains(mStop.actionMessage, "Stopping container api-srv") {
		t.Errorf("expected stopping toast, got: %q", mStop.actionMessage)
	}

	// 2. 's' on stopped container -> starts
	m.cursorIdx = 1
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	mStart := res.(Model)
	if cmd == nil {
		t.Errorf("expected non-nil cmd on 's' for stopped container")
	}
	if !strings.Contains(mStart.actionMessage, "Starting container db-srv") {
		t.Errorf("expected starting toast, got: %q", mStart.actionMessage)
	}

	// 3. 'p' on container -> restarts
	m.cursorIdx = 0
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	mRestart := res.(Model)
	if cmd == nil {
		t.Errorf("expected non-nil cmd on 'p'")
	}
	if !strings.Contains(mRestart.actionMessage, "Restarting api-srv") {
		t.Errorf("expected restarting toast, got: %q", mRestart.actionMessage)
	}

	// 4. 'i' and 'l' hotkeys invoke inspect and logs without panic
	m.cursorIdx = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	mInspect := res.(Model)
	if mInspect.activeModal != ModalInspect && !strings.Contains(mInspect.actionMessage, "Inspect error") {
		t.Errorf("expected inspect modal or toast error, got: %q", mInspect.actionMessage)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	mLogs := res.(Model)
	if mLogs.activeModal != ModalLogs && !strings.Contains(mLogs.actionMessage, "Logs error") {
		t.Errorf("expected logs modal or toast error, got: %q", mLogs.actionMessage)
	}
}

func TestContainerKillAndRemoveWorkflow(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("app-run"), testutil.WithState("running", "Up")),
		testutil.AContainer(testutil.WithID("c2"), testutil.WithName("app-stop"), testutil.WithState("stopped", "Exited")),
	}
	m.cursorIdx = 0

	// 1. Press K on running container triggers kill toast & cmd
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected non-nil cmd for killContainerCmd")
	}
	if !strings.Contains(m.actionMessage, "Force killing app-run") {
		t.Errorf("expected toast 'Force killing app-run', got: %s", m.actionMessage)
	}

	// 2. Press K on stopped container shows cannot kill toast
	m.cursorIdx = 1
	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}})
	m = res.(Model)
	if cmd != nil {
		t.Errorf("expected nil cmd when killing stopped container")
	}
	if !strings.Contains(m.actionMessage, "Cannot kill") {
		t.Errorf("expected 'Cannot kill' toast, got: %s", m.actionMessage)
	}

	// 3. Press x on container opens remove confirmation modal
	m.cursorIdx = 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = res.(Model)
	if m.activeModal != ModalRemoveContainer {
		t.Fatalf("expected activeModal 'remove_container', got: %s", m.activeModal)
	}
	if m.removeContainerTargetID != "c2" || m.removeContainerTargetName != "app-stop" {
		t.Errorf("unexpected remove targets: id=%s name=%s", m.removeContainerTargetID, m.removeContainerTargetName)
	}

	// 4. Modal view verification
	view := m.View()
	if !strings.Contains(view, "CONFIRM CONTAINER REMOVAL") || !strings.Contains(view, "app-stop") {
		t.Errorf("expected remove modal content, got: %s", view)
	}

	// 5. Cancel removal with n
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal closed after 'n', got: %s", m.activeModal)
	}

	// 6. Cancel removal with esc
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal closed after 'esc', got: %s", m.activeModal)
	}

	// 7. Cancel removal with q
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = res.(Model)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal closed after 'q', got: %s", m.activeModal)
	}

	// 8. Press d on container opens remove modal and confirm with y
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = res.(Model)
	if m.activeModal != ModalRemoveContainer {
		t.Fatalf("expected activeModal 'remove_container' after 'd', got: %s", m.activeModal)
	}

	res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if m.activeModal != "" {
		t.Errorf("expected activeModal closed after 'y', got: %s", m.activeModal)
	}
	if cmd == nil {
		t.Errorf("expected non-nil cmd for removeContainerCmd")
	}
	if !strings.Contains(m.actionMessage, "Removing container app-stop") {
		t.Errorf("expected remove toast, got: %s", m.actionMessage)
	}
}

func TestContainerCopyAndOpenURL(t *testing.T) {
	m := newFastDockerTestModel()
	m.dockerInfo.Status = "running"
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(
			testutil.WithID("c1"),
			testutil.WithName("web-frontend"),
			testutil.WithState("running", "Up 2 hours"),
			testutil.WithPorts("0.0.0.0:3000->3000/tcp"),
		),
		testutil.AContainer(
			testutil.WithID("c2"),
			testutil.WithName("headless-worker"),
			testutil.WithState("running", "Up 2 hours"),
			testutil.WithPorts(""),
		),
		testutil.AContainer(
			testutil.WithID("c3"),
			testutil.WithName("stopped-service"),
			testutil.WithState("exited", "Exited (0) 5 mins ago"),
			testutil.WithPorts("0.0.0.0:8080->8080/tcp"),
		),
	}

	// 1. Press 'y' on container with ports -> copies docker exec command
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Copied to clipboard: docker exec -it web-frontend sh") {
		t.Errorf("expected copy toast, got: %s", m.actionMessage)
	}

	// 2. Press 'w' when browser opener succeeds
	cleanupOpener := SetBrowserOpenerForTesting(func(_ string) bool { return true })
	m.cursorIdx = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = res.(Model)
	cleanupOpener()
	if !strings.Contains(m.actionMessage, "Opened http://localhost:3000 (copied to clipboard)") {
		t.Errorf("expected opened toast, got: %s", m.actionMessage)
	}

	// 3. Press 'w' when browser opener fails (fallback to copied URL)
	cleanupOpenerFail := SetBrowserOpenerForTesting(func(_ string) bool { return false })
	m.cursorIdx = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = res.(Model)
	cleanupOpenerFail()
	if !strings.Contains(m.actionMessage, "Copied URL: http://localhost:3000") {
		t.Errorf("expected copied URL toast when browser open fails, got: %s", m.actionMessage)
	}

	// 4. Move cursor down to container without ports and press 'w'
	m.cursorIdx = 1
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "No mapped ports for headless-worker") {
		t.Errorf("expected no mapped ports toast, got: %s", m.actionMessage)
	}

	// 5. Move cursor down to stopped container with ports and press 'w'
	m.cursorIdx = 2
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Cannot open URL: container stopped-service is not running") {
		t.Errorf("expected container not running toast, got: %s", m.actionMessage)
	}
}

func TestDockerActions_EmptyListSafety(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	m.dockerInfo.Containers = nil
	m.cursorIdx = 0

	// None of the container action keys should panic when list is empty
	keys := []string{"s", "S", "p", "P", "K", "e", "E", "x", "X", "d", "D", "y", "Y", "w", "W", "i", "I", "l", "L"}
	for _, k := range keys {
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd for key %q on empty container list", k)
		}
		if updated.activeModal != "" {
			t.Errorf("expected no modal opened for key %q on empty list", k)
		}
	}
}

func TestToggleDetailsHotkeys(t *testing.T) {
	t.Parallel()

	m := newFastDockerTestModel()
	if m.hideDetails {
		t.Errorf("expected hideDetails to be false initially")
	}

	// 1. Press 'v' to hide details
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = res.(Model)
	if !m.hideDetails {
		t.Errorf("expected hideDetails to be true after pressing 'v'")
	}
	if !strings.Contains(m.actionMessage, "hidden") {
		t.Errorf("expected toast message mentioning hidden, got: %s", m.actionMessage)
	}

	// 2. Press 'V' to show details again
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = res.(Model)
	if m.hideDetails {
		t.Errorf("expected hideDetails to be false after pressing 'V'")
	}
	if !strings.Contains(m.actionMessage, "visible") {
		t.Errorf("expected toast message mentioning visible, got: %s", m.actionMessage)
	}

	// 3. Test SetHideDetails
	m.SetHideDetails(true)
	if !m.hideDetails {
		t.Errorf("expected hideDetails true after SetHideDetails(true)")
	}
	m.SetHideDetails(false)
	if m.hideDetails {
		t.Errorf("expected hideDetails false after SetHideDetails(false)")
	}
}

// =============================================================================
// Tests from keys_images_test.go
// =============================================================================

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

// =============================================================================
// Tests from keys_compose_test.go
// =============================================================================

func newFastComposeTestModel() Model {
	m := newFastTestModel()
	m.activeTab = TabCompose
	m.compose = []monitor.ComposeProject{
		{
			Name:            "proj-alpha",
			Status:          "running",
			StatusDisplay:   "Running",
			TotalServices:   2,
			RunningServices: 2,
			Services: []monitor.ComposeService{
				{Name: "web", State: "running"},
				{Name: "api", State: "running"},
			},
		},
		{
			Name:            "proj-beta",
			Status:          "stopped",
			StatusDisplay:   "Stopped",
			TotalServices:   1,
			RunningServices: 0,
			Services: []monitor.ComposeService{
				{Name: "worker", State: "exited"},
			},
		},
	}
	m.composeCursorIdx = 0
	return m
}

func TestComposeTabNavigationAndActions(t *testing.T) {
	t.Parallel()

	t.Run("cursor navigation with wrapping", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()

		// Down moves to second project
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = res.(Model)
		if m.composeCursorIdx != 1 {
			t.Errorf("expected composeCursorIdx=1, got %d", m.composeCursorIdx)
		}

		// Down wraps back to 0
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = res.(Model)
		if m.composeCursorIdx != 0 {
			t.Errorf("expected composeCursorIdx=0 after wrap, got %d", m.composeCursorIdx)
		}

		// Up wraps to 1
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = res.(Model)
		if m.composeCursorIdx != 1 {
			t.Errorf("expected composeCursorIdx=1 after up wrap, got %d", m.composeCursorIdx)
		}

		// Key 'k' moves to 0
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		m = res.(Model)
		if m.composeCursorIdx != 0 {
			t.Errorf("expected composeCursorIdx=0 after 'k', got %d", m.composeCursorIdx)
		}

		// Key 'j' moves to 1
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = res.(Model)
		if m.composeCursorIdx != 1 {
			t.Errorf("expected composeCursorIdx=1 after 'j', got %d", m.composeCursorIdx)
		}
	})

	t.Run("view rendering contains footer hints and selection marker", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()
		view := m.View()

		for _, hint := range []string{"Logs", "Restart", "Stop", "Start", "Down"} {
			if !strings.Contains(view, hint) {
				t.Errorf("expected footer view to contain hint %q, got: %s", hint, view)
			}
		}
		if !strings.Contains(view, "► ") {
			t.Errorf("expected cursor marker '► ' in compose tab view")
		}
	})

	t.Run("action hotkeys start stop restart logs", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()
		m.composeCursorIdx = 1 // target proj-beta

		// Test 'p' restarts compose project
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
		mRestart := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on 'p'")
		}
		if !strings.Contains(mRestart.actionMessage, "Restarting Compose project proj-beta") {
			t.Errorf("expected restarting toast, got: %q", mRestart.actionMessage)
		}

		// Test 's' stops compose project
		res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		mStop := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on 's'")
		}
		if !strings.Contains(mStop.actionMessage, "Stopping Compose project proj-beta") {
			t.Errorf("expected stopping toast, got: %q", mStop.actionMessage)
		}

		// Test 'u' starts compose project
		res, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
		mStart := res.(Model)
		if cmd == nil {
			t.Errorf("expected non-nil cmd on 'u'")
		}
		if !strings.Contains(mStart.actionMessage, "Starting Compose project proj-beta") {
			t.Errorf("expected starting toast, got: %q", mStart.actionMessage)
		}

		// Test 'l' opens logs modal or triggers toast feedback
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		mLogs := res.(Model)
		if mLogs.activeModal == ModalLogs {
			if !strings.Contains(mLogs.logsTargetName, "proj-beta") {
				t.Errorf("expected logsTargetName to contain 'proj-beta', got %q", mLogs.logsTargetName)
			}
		} else if !strings.Contains(mLogs.actionMessage, "Logs error") {
			t.Errorf("expected logs modal or error toast, got message=%q", mLogs.actionMessage)
		}
	})

	t.Run("clipboard copy project name with 'y'", func(t *testing.T) {
		defer clipboard.TestLock()()
		origWriter := clipboard.GetOutputWriter()
		var buf bytes.Buffer
		clipboard.SetOutputWriter(&buf)
		defer func() { clipboard.SetOutputWriter(origWriter) }()

		m := newFastComposeTestModel()
		m.composeCursorIdx = 0

		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd from clipboard copy")
		}
		if !strings.Contains(updated.actionMessage, "Copied to clipboard: proj-alpha") {
			t.Errorf("expected copy toast, got: %q", updated.actionMessage)
		}
		expectedB64 := base64.StdEncoding.EncodeToString([]byte("proj-alpha"))
		if !strings.Contains(buf.String(), expectedB64) {
			t.Errorf("expected clipboard output to contain base64 %q, got: %q", expectedB64, buf.String())
		}
	})

	t.Run("compose down modal workflow confirmation and cancel", func(t *testing.T) {
		t.Parallel()
		m := newFastComposeTestModel()
		m.composeCursorIdx = 1 // proj-beta

		// Open modal with 'd'
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
		m = res.(Model)
		if m.activeModal != ModalComposeDown {
			t.Fatalf("expected activeModal 'compose_down', got %q", m.activeModal)
		}
		if m.composeDownTarget != "proj-beta" {
			t.Errorf("expected composeDownTarget 'proj-beta', got %q", m.composeDownTarget)
		}

		modalView := m.View()
		if !strings.Contains(modalView, "CONFIRM DOCKER COMPOSE DOWN") || !strings.Contains(modalView, "proj-beta") {
			t.Errorf("expected compose down modal content, got: %s", modalView)
		}

		// Cancel modal with 'n'
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
		mCancel := res.(Model)
		if mCancel.activeModal != "" || mCancel.composeDownTarget != "" {
			t.Errorf("expected modal closed and target cleared after 'n', got modal=%q", mCancel.activeModal)
		}

		// Cancel modal with 'esc'
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		mCancelEsc := res.(Model)
		if mCancelEsc.activeModal != "" {
			t.Errorf("expected modal closed after 'esc'")
		}

		// Cancel modal with 'q'
		res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		mCancelQ := res.(Model)
		if mCancelQ.activeModal != "" {
			t.Errorf("expected modal closed after 'q'")
		}

		// Re-open and confirm with 'y'
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		mConfirm := res.(Model)
		if mConfirm.activeModal != "" || mConfirm.composeDownTarget != "" {
			t.Errorf("expected activeModal cleared after 'y'")
		}
		if cmd == nil {
			t.Errorf("expected non-nil tea.Cmd for downComposeCmd")
		}
		if !strings.Contains(mConfirm.actionMessage, "Downing Compose project proj-beta") {
			t.Errorf("expected downing toast, got: %q", mConfirm.actionMessage)
		}
	})
}

func TestComposeNavigationKeys(t *testing.T) {
	t.Parallel()

	m := newFastComposeTestModel()
	m.compose = append(m.compose,
		monitor.ComposeProject{Name: "proj-gamma"},
		monitor.ComposeProject{Name: "proj-delta"},
		monitor.ComposeProject{Name: "proj-epsilon"},
		monitor.ComposeProject{Name: "proj-zeta"},
	) // Total 6 projects

	m.composeCursorIdx = 0

	// End / 'G' moves to last element
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = res.(Model)
	if m.composeCursorIdx != 5 {
		t.Errorf("expected composeCursorIdx=5 after 'G', got %d", m.composeCursorIdx)
	}

	// Home / 'g' moves to first element
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = res.(Model)
	if m.composeCursorIdx != 0 {
		t.Errorf("expected composeCursorIdx=0 after 'g', got %d", m.composeCursorIdx)
	}

	// PageDown moves cursor forward by 5
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = res.(Model)
	if m.composeCursorIdx != 5 {
		t.Errorf("expected composeCursorIdx=5 after PgDown, got %d", m.composeCursorIdx)
	}

	// PageUp moves cursor back by 5
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = res.(Model)
	if m.composeCursorIdx != 0 {
		t.Errorf("expected composeCursorIdx=0 after PgUp, got %d", m.composeCursorIdx)
	}
}

func TestComposeActions_EmptyListSafety(t *testing.T) {
	t.Parallel()

	m := newFastComposeTestModel()
	m.compose = nil
	m.composeCursorIdx = 0

	// Pressing actions on empty list does not panic and returns nil cmd
	actionKeys := []string{"s", "S", "u", "U", "p", "P", "d", "D", "l", "L", "y", "Y"}
	for _, k := range actionKeys {
		res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		updated := res.(Model)
		if cmd != nil {
			t.Errorf("expected nil cmd when key %q pressed on empty list", k)
		}
		if updated.activeModal != "" {
			t.Errorf("expected no modal opened for key %q on empty list", k)
		}
	}
}

// =============================================================================
// Tests from keys_networks_test.go
// =============================================================================

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

// =============================================================================
// Tests from keys_volumes_test.go
// =============================================================================

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

// =============================================================================
// Tests from keys_swarm_test.go
// =============================================================================

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
