package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

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
	origOpener := browserOpener
	defer func() { browserOpener = origOpener }()

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
	browserOpener = func(_ string) bool { return true }
	m.cursorIdx = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = res.(Model)
	if !strings.Contains(m.actionMessage, "Opened http://localhost:3000 (copied to clipboard)") {
		t.Errorf("expected opened toast, got: %s", m.actionMessage)
	}

	// 3. Press 'w' when browser opener fails (fallback to copied URL)
	browserOpener = func(_ string) bool { return false }
	m.cursorIdx = 0
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = res.(Model)
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
