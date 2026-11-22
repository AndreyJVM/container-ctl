package e2e

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

var binaryPath string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "container-ctl-e2e-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tempDir)

	binaryPath = filepath.Join(tempDir, "container-ctl")
	cmd := exec.Command("go", "build", "-o", binaryPath, "github.com/AndreyJVM/container-ctl/cmd/container-ctl")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build container-ctl binary: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestE2E_VersionFlag(t *testing.T) {
	flags := []string{"--version", "-v"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			cmd := exec.Command(binaryPath, flag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("unexpected error running %s: %v, output: %s", flag, err, output)
			}
			outStr := string(output)
			if !strings.Contains(outStr, "container-ctl") || !strings.Contains(outStr, "v0.") {
				t.Errorf("expected version output for %s, got: %s", flag, outStr)
			}
		})
	}
}

func TestE2E_HelpFlag(t *testing.T) {
	flags := []string{"--help", "-h"}
	for _, flag := range flags {
		t.Run(flag, func(t *testing.T) {
			cmd := exec.Command(binaryPath, flag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("unexpected error running %s: %v, output: %s", flag, err, output)
			}
			outStr := string(output)
			if !strings.Contains(outStr, "Usage:") || !strings.Contains(outStr, "Options:") || !strings.Contains(outStr, "TUI controls:") {
				t.Errorf("expected usage help output for %s, got: %s", flag, outStr)
			}
		})
	}
}

func TestE2E_PrintConfig(t *testing.T) {
	cmd := exec.Command(binaryPath, "--print-config")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error running --print-config: %v, output: %s", err, output)
	}
	outStr := string(output)
	if !strings.Contains(outStr, "interval:") || !strings.Contains(outStr, "initial_tab:") {
		t.Errorf("expected configuration output, got: %s", outStr)
	}
}

func TestE2E_InitConfig(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "config.yaml")
	cmd := exec.Command(binaryPath, "--init-config", "-c", tempFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error running --init-config: %v, output: %s", err, output)
	}
	outStr := string(output)
	if !strings.Contains(outStr, "Configuration file initialized at:") {
		t.Errorf("expected initialization message, got: %s", outStr)
	}
	if _, statErr := os.Stat(tempFile); statErr != nil {
		t.Errorf("expected config file to be created at %s", tempFile)
	}

	// Repeat without --force should fail
	cmdRepeat := exec.Command(binaryPath, "--init-config", "-c", tempFile)
	repeatOut, err := cmdRepeat.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error overwriting existing config without --force, got success: %s", repeatOut)
	}
	if !strings.Contains(string(repeatOut), "already exists") {
		t.Errorf("expected 'already exists' in error, got: %s", string(repeatOut))
	}

	// Repeat with --force should succeed
	cmdForce := exec.Command(binaryPath, "--init-config", "-c", tempFile, "--force")
	forceOut, err := cmdForce.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error initializing with --force: %v, output: %s", err, forceOut)
	}

	// Repeat with -f should succeed
	cmdShortForce := exec.Command(binaryPath, "--init-config", "-c", tempFile, "-f")
	shortForceOut, err := cmdShortForce.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error initializing with -f: %v, output: %s", err, shortForceOut)
	}
}

func TestE2E_InvalidFlag(t *testing.T) {
	cmd := exec.Command(binaryPath, "--unknown-flag-xyz")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit code for invalid flag, got success")
	}
	outStr := string(output)
	if !strings.Contains(outStr, "flag provided but not defined") {
		t.Errorf("expected flag error in output, got: %s", outStr)
	}
}

func TestE2E_IntervalAndTabFlags(t *testing.T) {
	// Valid interval and tab
	cmd := exec.Command(binaryPath, "-i", "1s", "-t", "images", "-v")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success for valid interval and tab: %v, output: %s", err, output)
	}

	// Invalid interval
	cmdInvalidInterval := exec.Command(binaryPath, "-i", "bad-time")
	outInvalidInterval, err := cmdInvalidInterval.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for invalid interval, got success: %s", outInvalidInterval)
	}

	// Invalid tab
	cmdInvalidTab := exec.Command(binaryPath, "-t", "unknown-tab")
	outInvalidTab, err := cmdInvalidTab.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for invalid tab, got success: %s", outInvalidTab)
	}
}

func TestE2E_NonInteractiveTerminalCheck(t *testing.T) {
	// Running container-ctl directly without a real TTY must fail gracefully with descriptive error
	cmd := exec.Command(binaryPath)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error when running non-interactively without TTY, got success")
	}
	outStr := string(output)
	if !strings.Contains(outStr, "requires an interactive terminal (TTY)") {
		t.Errorf("expected TTY requirement error, got: %s", outStr)
	}
}

func TestE2E_PositionalArgumentsError(t *testing.T) {
	cmd := exec.Command(binaryPath, "unexpected-arg")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for positional argument, got success")
	}
	outStr := string(output)
	if !strings.Contains(outStr, "unknown positional arguments: unexpected-arg") {
		t.Errorf("expected positional args error message, got: %s", outStr)
	}
}

func TestE2E_NonExistentConfigFile(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	cmd := exec.Command(binaryPath, "-c", nonExistent)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for non-existent config file, got success")
	}
	outStr := string(output)
	if !strings.Contains(outStr, "failed to read config file") {
		t.Errorf("expected config read error, got: %s", outStr)
	}
}

func TestE2E_EnvironmentVariables(t *testing.T) {
	cmd := exec.Command(binaryPath, "--print-config")
	cmd.Env = append(os.Environ(),
		"DOCKPULSE_INTERVAL=3s",
		"DOCKPULSE_TAB=volumes",
		"DOCKPULSE_SHOW_SWARM=true",
		"DOCKPULSE_SOCKET=/var/run/custom.sock",
		"DOCKPULSE_UNUSED_IMAGES=true",
		"DOCKPULSE_SORT=mem",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error running with env vars: %v, output: %s", err, output)
	}
	outStr := string(output)
	if !strings.Contains(outStr, "interval: 3s") {
		t.Errorf("expected interval: 3s in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "initial_tab: volumes (5)") {
		t.Errorf("expected initial_tab: volumes (5) in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "show_swarm: true") {
		t.Errorf("expected show_swarm: true in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "docker.socket_path: /var/run/custom.sock") {
		t.Errorf("expected docker.socket_path in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "ui.only_unused_images: true") {
		t.Errorf("expected ui.only_unused_images in output, got: %s", outStr)
	}
	if !strings.Contains(outStr, "ui.default_sort: mem") {
		t.Errorf("expected ui.default_sort in output, got: %s", outStr)
	}
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *safeBuffer) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Len()
}

func TestE2E_InteractivePTY_FullSession(t *testing.T) {
	cmd := exec.Command(binaryPath, "-i", "200ms")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 35, Cols: 120})
	if err != nil {
		t.Fatalf("failed to start pseudo-terminal: %v", err)
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// Drain ptmx output in background to safeBuffer
	var outBuf safeBuffer
	go func() {
		_, _ = io.Copy(&outBuf, ptmx)
	}()

	var offset int
	waitForOutput := func(expected string, timeout time.Duration) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			s := outBuf.String()
			if offset < len(s) {
				idx := strings.Index(s[offset:], expected)
				if idx != -1 {
					offset += idx + len(expected)
					return true
				}
			}
			time.Sleep(30 * time.Millisecond)
		}
		return false
	}

	// 1. Wait for child process TUI event loop to start and render initial Containers tab
	time.Sleep(500 * time.Millisecond)
	if !waitForOutput("[S] Start/Stop", 3*time.Second) {
		t.Fatalf("timed out waiting for initial Containers frame")
	}

	// 2. Switch to Images tab (2)
	_, _ = ptmx.Write([]byte("2"))
	if !waitForOutput("[u] Filter Unused", 2*time.Second) {
		t.Errorf("expected Images tab to render upon '2' keypress")
	}

	// 3. Press '?' to trigger Help modal and wait for it to appear
	_, _ = ptmx.Write([]byte("?"))
	if !waitForOutput("Auto-refresh interval:", 2*time.Second) {
		t.Errorf("expected TUI to render help modal upon '?' keypress")
	}

	// 4. Press '?' again to dismiss help modal
	_, _ = ptmx.Write([]byte("?"))
	time.Sleep(150 * time.Millisecond)

	// 5. Switch to Compose tab (3)
	_, _ = ptmx.Write([]byte("3"))
	if !waitForOutput("[D] Down", 2*time.Second) {
		t.Errorf("expected Compose tab to render upon '3' keypress")
	}

	// 6. Switch to Networks tab (4)
	_, _ = ptmx.Write([]byte("4"))
	if !waitForOutput("DOCKER NETWORKS", 2*time.Second) {
		t.Errorf("expected Networks tab to render upon '4' keypress")
	}

	// 7. Switch to Volumes tab (5)
	_, _ = ptmx.Write([]byte("5"))
	if !waitForOutput("DOCKER VOLUMES", 2*time.Second) {
		t.Errorf("expected Volumes tab to render upon '5' keypress")
	}

	// 8. Switch back to Containers tab (1)
	_, _ = ptmx.Write([]byte("1"))
	if !waitForOutput("[S] Start/Stop", 2*time.Second) {
		t.Errorf("expected Containers tab to render upon '1' keypress")
	}

	// 9. Press 'q' to gracefully quit the TUI application
	_, _ = ptmx.Write([]byte("q"))

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case waitErr := <-done:
		if waitErr != nil {
			t.Fatalf("expected clean exit 0 upon quitting with 'q', got: %v", waitErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for interactive container-ctl session to exit on 'q'")
	}

	captured := outBuf.String()
	if !strings.Contains(captured, "DOCKER CONTAINERS") {
		t.Errorf("expected TUI to render 'DOCKER CONTAINERS' screen, got output length %d", len(captured))
	}
	if !strings.Contains(captured, "Toggle this help window") {
		t.Errorf("expected TUI to render help modal upon '?' keypress")
	}
}

func TestE2E_InteractivePTY_FilterAndPrune(t *testing.T) {
	cmd := exec.Command(binaryPath, "-t", "networks", "-i", "200ms")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 35, Cols: 120})
	if err != nil {
		t.Fatalf("failed to start pseudo-terminal: %v", err)
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	var outBuf safeBuffer
	go func() {
		_, _ = io.Copy(&outBuf, ptmx)
	}()

	var offset int
	waitForOutput := func(expected string, timeout time.Duration) bool {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			s := outBuf.String()
			if offset < len(s) {
				idx := strings.Index(s[offset:], expected)
				if idx != -1 {
					offset += idx + len(expected)
					return true
				}
			}
			time.Sleep(30 * time.Millisecond)
		}
		return false
	}

	// 1. Wait for initial Networks tab
	time.Sleep(500 * time.Millisecond)
	if !waitForOutput("DOCKER NETWORKS", 3*time.Second) {
		t.Fatalf("timed out waiting for initial Networks tab frame")
	}

	// 2. Press '/' to trigger search filter
	_, _ = ptmx.Write([]byte("/"))
	if !waitForOutput("[Search:", 2*time.Second) {
		t.Errorf("expected [Search: prompt upon pressing '/'")
	}

	// 3. Type query and press Enter
	_, _ = ptmx.Write([]byte("bridge\r"))
	time.Sleep(150 * time.Millisecond)

	// 4. Press Esc to clear filter
	_, _ = ptmx.Write([]byte{0x1b})
	time.Sleep(150 * time.Millisecond)

	// 5. Press 'C' to open Prune modal dialog
	_, _ = ptmx.Write([]byte("C"))
	if !waitForOutput("CLEANUP & PRUNE DOCKER RESOURCES", 2*time.Second) {
		t.Errorf("expected CLEANUP & PRUNE dialog upon pressing 'C'")
	}

	// 6. Press Esc to cancel and dismiss Prune dialog
	_, _ = ptmx.Write([]byte{0x1b})
	time.Sleep(150 * time.Millisecond)

	// 7. Quit with 'q'
	_, _ = ptmx.Write([]byte("q"))

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case waitErr := <-done:
		if waitErr != nil {
			t.Fatalf("expected clean exit code 0 on quit, got: %v", waitErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for container-ctl to exit cleanly")
	}
}
