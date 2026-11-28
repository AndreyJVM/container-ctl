package e2e

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	"github.com/AndreyJVM/container-ctl/internal/tui"
	"github.com/AndreyJVM/container-ctl/internal/tui/actions"
)

// TestE2E_ActionResultMessageHandling verifies how asynchronous action results (actions.FinishedMsg)
// update the model's toast notifications, trigger background data refreshes, and handle errors.
func TestE2E_ActionResultMessageHandling(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Successful action with message and refresh=true
	successMsg := actions.FinishedMsg{
		Message: "Container web-gateway restarted successfully",
		Refresh: true,
	}
	updated, refreshCmd := m.Update(successMsg)
	m = updated.(tui.Model)

	assertToastContains(t, m, "Container web-gateway restarted successfully")
	assertContains(t, m.View(), "Container web-gateway restarted successfully")
	if refreshCmd == nil {
		t.Fatalf("expected data refresh command dispatched when FinishedMsg has refresh=true")
	}

	// 2. Action failure with error and refresh=false
	errMsg := actions.FinishedMsg{
		Err:     errors.New("connection reset by docker daemon"),
		Refresh: false,
	}
	updated, noRefreshCmd := m.Update(errMsg)
	m = updated.(tui.Model)

	assertToastContains(t, m, "connection reset by docker daemon")
	assertContains(t, m.View(), "connection reset by docker daemon")
	if noRefreshCmd != nil {
		t.Fatalf("expected no refresh command when refresh=false")
	}

	// 3. Action with both error and message: error takes precedence
	combinedMsg := actions.FinishedMsg{
		Err:     errors.New("permission denied accessing socket"),
		Message: "This message should be overshadowed by error",
		Refresh: false,
	}
	updated, _ = m.Update(combinedMsg)
	m = updated.(tui.Model)

	assertToastContains(t, m, "permission denied accessing socket")
	assertNotContains(t, m.ToastMessage(), "overshadowed")
}

// TestE2E_ExternalExecFinishedHandling verifies model state updates and toasts
// when an external terminal shell session terminates cleanly or with errors.
func TestE2E_ExternalExecFinishedHandling(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// 1. Clean termination (err == nil)
	cleanMsg := tui.NewExecFinishedMsg("web-gateway", nil)
	updated, cleanCmd := m.Update(cleanMsg)
	m = updated.(tui.Model)

	assertToastContains(t, m, "Shell session in web-gateway ended")
	if cleanCmd == nil {
		t.Fatalf("expected data refresh command when exec session finishes")
	}

	// 2. Terminated via error (e.g. binary not found)
	errExec := errors.New("exec: /bin/bash: executable file not found in $PATH")
	failMsg := tui.NewExecFinishedMsg("api-backend", errExec)
	updated, _ = m.Update(failMsg)
	m = updated.(tui.Model)

	assertToastContains(t, m, "Shell error in api-backend")
	assertToastContains(t, m, "executable file not found")
}

// TestE2E_EmbeddedShellWindowResizeAndAutoClose verifies dynamic PTY resizing
// when terminal dimensions change, as well as automatic closure if resize fails.
func TestE2E_EmbeddedShellWindowResizeAndAutoClose(t *testing.T) {
	fake := &e2eFakeShellSession{}
	cleanupShell := tui.SetStartEmbeddedShellForTesting(func(cmd *exec.Cmd, width, height int) (tui.EmbeddedShellSession, error) {
		return fake, nil
	})
	defer cleanupShell()

	m := sendWindowSize(createComprehensiveTestModel(), 120, 35)

	// 1. Launch embedded shell
	m, _ = sendKey(m, "e")
	assertModal(t, m, tui.ModalExec)
	m, _ = sendKey(m, "enter")
	assertModalClosed(t, m)

	if !m.IsShellActive() {
		t.Fatalf("expected embedded shell to be active")
	}

	// 2. Terminal resize: sends tea.WindowSizeMsg to model
	m = sendWindowSize(m, 160, 50)
	if len(fake.resizeCalls) == 0 {
		t.Fatalf("expected embedded shell Resize() to be called on window size change")
	}
	lastResize := fake.resizeCalls[len(fake.resizeCalls)-1]
	if lastResize.W <= 0 || lastResize.H <= 0 {
		t.Fatalf("invalid resize dimensions: %v", lastResize)
	}
	if !m.IsShellActive() {
		t.Fatalf("shell should remain active after successful resize")
	}

	// 3. Terminal resize error: mock returns an error on Resize()
	fake.resizeErr = errors.New("PTY window resize ioctl failed")
	m = sendWindowSize(m, 100, 30)

	// Model must catch the error, show a toast, and auto-close the shell
	if m.IsShellActive() {
		t.Fatalf("expected embedded shell to auto-close when resize returns an error")
	}
	assertToastContains(t, m, "Shell error in web-gateway: PTY window resize ioctl failed")
}

// TestE2E_ContainerPortMappingAndBrowserURL verifies pressing 'w' extracts
// container URLs and handles running vs stopped vs unmapped containers properly.
func TestE2E_ContainerPortMappingAndBrowserURL(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Setup containers with varied port mapping conditions
	containers := []monitor.ContainerItem{
		testutil.AContainer(
			testutil.WithID("c-running-multiport"),
			testutil.WithName("multiport-service"),
			testutil.WithState("running", "Up 3 hours"),
			testutil.WithPorts("0.0.0.0:8080->80/tcp, 0.0.0.0:8443->443/tcp"),
		),
		testutil.AContainer(
			testutil.WithID("c-running-noports"),
			testutil.WithName("worker-daemon"),
			testutil.WithState("running", "Up 5 hours"),
			testutil.WithPorts(""),
		),
		testutil.AContainer(
			testutil.WithID("c-stopped"),
			testutil.WithName("stopped-service"),
			testutil.WithState("exited", "Exited (0) 10 minutes ago"),
			testutil.WithPorts("0.0.0.0:9090->9090/tcp"),
		),
	}

	updated, _ := m.Update(actions.DataFetchedMsg{
		DockerInfo: monitor.DockerSummary{
			Containers: containers,
		},
		Timestamp: "12:30:00",
	})
	m = updated.(tui.Model)

	// 1. Cursor on multiport running container: press 'w'
	m, _ = sendKey(m, "w")
	assertContainsAny(t, m.ToastMessage(), "8080", "Copied", "Opened")

	// 2. Cursor down to running container with no mapped ports: press 'w'
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "w")
	assertToastContains(t, m, "No mapped ports for worker-daemon")

	// 3. Cursor down to stopped container: press 'w'
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "w")
	assertToastContains(t, m, "Cannot open URL: container stopped-service is not running")
}

// TestE2E_ImageRemovalWithInUseConfirmation verifies in-use safety warning
// and confirmation workflow when deleting an in-use image.
func TestE2E_ImageRemovalWithInUseConfirmation(t *testing.T) {
	t.Parallel()

	m := createComprehensiveTestModel()

	// Setup images: img1 is in-use, img2 is unused
	imagesMsg := actions.DataFetchedMsg{
		Images: []monitor.DockerImage{
			testutil.AnImage(
				testutil.WithImageID("img-in-use", "sha256:inuse123"),
				testutil.WithRepoTag("postgres", "16-alpine"),
				testutil.WithImageStatus(true, false, 2),
			),
			testutil.AnImage(
				testutil.WithImageID("img-unused", "sha256:unused123"),
				testutil.WithRepoTag("busybox", "latest"),
				testutil.WithImageStatus(false, false, 0),
			),
		},
		Timestamp: "12:35:00",
	}
	updated, _ := m.Update(imagesMsg)
	m = updated.(tui.Model)

	// Switch to Images tab (2)
	m, _ = sendKey(m, "2")
	assertTab(t, m, tui.TabImages)

	// 1. Press 'd' / 'x' to open Remove Modal for in-use image (postgres:16-alpine)
	m, _ = sendKey(m, "x")
	assertModal(t, m, tui.ModalRemoveImage)

	modalView := m.View()
	assertContains(t, modalView, "CONFIRM IMAGE REMOVAL")
	assertContains(t, modalView, "postgres:16-alpine")
	assertContainsAny(t, modalView, "IN USE", "in use")

	// 2. Cancel with 'n'
	m, _ = sendKey(m, "n")
	assertModalClosed(t, m)

	// 3. Open again and confirm with 'y' -> dispatches remove image command
	m, _ = sendKey(m, "x")
	assertModal(t, m, tui.ModalRemoveImage)
	m, removeCmd := sendKey(m, "y")
	assertModalClosed(t, m)
	if removeCmd == nil {
		t.Fatalf("expected remove image command dispatched upon confirmation")
	}
}
