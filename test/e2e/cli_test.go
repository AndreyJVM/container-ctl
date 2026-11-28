package e2e

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	t.Parallel()
	flags := []string{"--version", "-v"}
	for _, flag := range flags {
		flag := flag
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
	flags := []string{"--help", "-h"}
	for _, flag := range flags {
		flag := flag
		t.Run(flag, func(t *testing.T) {
			t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

type ptySession struct {
	t      *testing.T
	cmd    *exec.Cmd
	ptmx   *os.File
	outBuf safeBuffer
	offset int
}

func startPTYSession(t *testing.T, cmd *exec.Cmd, size *pty.Winsize) *ptySession {
	t.Helper()
	if runtime.GOOS == "darwin" && os.Getenv("CI") != "" {
		t.Skip("skipping interactive PTY test on macOS in headless CI environment")
	}

	ptmx, err := pty.StartWithSize(cmd, size)
	if err != nil {
		t.Fatalf("failed to start pseudo-terminal: %v", err)
	}

	s := &ptySession{
		t:    t,
		cmd:  cmd,
		ptmx: ptmx,
	}

	t.Cleanup(func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	go func() {
		_, _ = io.Copy(&s.outBuf, ptmx)
	}()

	return s
}

func (s *ptySession) Write(p []byte) {
	s.t.Helper()
	_, _ = s.ptmx.Write(p)
}

func (s *ptySession) WriteString(str string) {
	s.t.Helper()
	_, _ = s.ptmx.Write([]byte(str))
}

func (s *ptySession) Resize(rows, cols uint16) {
	s.t.Helper()
	if err := pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols}); err != nil {
		s.t.Fatalf("failed to resize PTY: %v", err)
	}
}

func (s *ptySession) WaitForOutput(expected string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		str := s.outBuf.String()
		if s.offset < len(str) {
			idx := strings.Index(str[s.offset:], expected)
			if idx != -1 {
				s.offset += idx + len(expected)
				return true
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	return false
}

func (s *ptySession) RequireOutput(expected string, timeout time.Duration) {
	s.t.Helper()
	if !s.WaitForOutput(expected, timeout) {
		s.t.Fatalf("timed out waiting for output %q, current output:\n%s", expected, s.outBuf.String())
	}
}

func (s *ptySession) Output() string {
	return s.outBuf.String()
}

func (s *ptySession) WaitExit(timeout time.Duration) error {
	s.t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		s.t.Fatalf("timed out waiting for process to exit within %v", timeout)
		return nil
	}
}

func TestE2E_InteractivePTY_FullSession(t *testing.T) {
	cmd := exec.Command(binaryPath, "-i", "200ms")
	s := startPTYSession(t, cmd, &pty.Winsize{Rows: 35, Cols: 120})

	time.Sleep(500 * time.Millisecond)
	s.RequireOutput("[S] Start/Stop", 6*time.Second)

	s.WriteString("2")
	if !s.WaitForOutput("[u] Filter Unused", 6*time.Second) {
		t.Errorf("expected Images tab to render upon '2' keypress")
	}

	s.WriteString("?")
	if !s.WaitForOutput("Auto-refresh interval:", 6*time.Second) {
		t.Errorf("expected TUI to render help modal upon '?' keypress")
	}

	s.WriteString("?")
	time.Sleep(200 * time.Millisecond)

	s.WriteString("3")
	if !s.WaitForOutput("[D] Down", 6*time.Second) {
		t.Errorf("expected Compose tab to render upon '3' keypress")
	}

	s.WriteString("4")
	if !s.WaitForOutput("[4: Networks]", 6*time.Second) {
		t.Errorf("expected Networks tab to render upon '4' keypress")
	}

	s.WriteString("5")
	if !s.WaitForOutput("[5: Volumes]", 6*time.Second) {
		t.Errorf("expected Volumes tab to render upon '5' keypress")
	}

	s.WriteString("1")
	if !s.WaitForOutput("[S] Start/Stop", 6*time.Second) {
		t.Errorf("expected Containers tab to render upon '1' keypress")
	}

	s.WriteString("q")
	if err := s.WaitExit(7 * time.Second); err != nil {
		t.Fatalf("expected clean exit 0 upon quitting with 'q', got: %v", err)
	}

	captured := s.Output()
	assertContains(t, captured, "DOCKER CONTAINERS")
	assertContains(t, captured, "Toggle this help window")
}

func TestE2E_InteractivePTY_FilterAndPrune(t *testing.T) {
	cmd := exec.Command(binaryPath, "-t", "networks", "-i", "200ms")
	s := startPTYSession(t, cmd, &pty.Winsize{Rows: 35, Cols: 120})

	time.Sleep(500 * time.Millisecond)
	s.RequireOutput("DOCKER NETWORKS", 6*time.Second)

	s.WriteString("/")
	if !s.WaitForOutput("[Search:", 6*time.Second) {
		t.Errorf("expected [Search: prompt upon pressing '/'")
	}

	s.WriteString("bridge\r")
	time.Sleep(200 * time.Millisecond)

	s.Write([]byte{0x1b}) // Esc
	time.Sleep(200 * time.Millisecond)

	s.WriteString("C") // Prune modal
	if !s.WaitForOutput("CLEANUP & PRUNE DOCKER RESOURCES", 8*time.Second) {
		t.Errorf("expected CLEANUP & PRUNE dialog upon pressing 'C'")
	}

	s.Write([]byte{0x1b}) // Esc
	time.Sleep(200 * time.Millisecond)

	s.WriteString("q")
	if err := s.WaitExit(7 * time.Second); err != nil {
		t.Fatalf("expected clean exit code 0 on quit, got: %v", err)
	}
}

func TestE2E_InteractivePTY_CustomConfig(t *testing.T) {
	customYAML := `
interval: 200ms
initial_tab: 4
modules:
  networks:
    enabled: true
    title: "NetMesh"
    order: 1
  containers:
    enabled: true
    title: "PodBox"
    order: 2
`
	cfgPath := filepath.Join(t.TempDir(), "custom_e2e_config.yaml")
	if err := os.WriteFile(cfgPath, []byte(customYAML), 0644); err != nil {
		t.Fatalf("failed to write custom config file: %v", err)
	}

	cmd := exec.Command(binaryPath, "-c", cfgPath)
	s := startPTYSession(t, cmd, &pty.Winsize{Rows: 35, Cols: 120})

	time.Sleep(500 * time.Millisecond)
	s.RequireOutput("NetMesh", 6*time.Second)

	s.WriteString("2")
	if !s.WaitForOutput("PodBox", 6*time.Second) {
		t.Errorf("expected PodBox tab title to be active upon '2'")
	}
	time.Sleep(300 * time.Millisecond)

	s.WriteString("q")
	if err := s.WaitExit(7 * time.Second); err != nil {
		t.Fatalf("expected clean exit code 0 on quit, got: %v", err)
	}
}

func TestE2E_InteractivePTY_DynamicResize(t *testing.T) {
	cmd := exec.Command(binaryPath, "-i", "200ms")
	s := startPTYSession(t, cmd, &pty.Winsize{Rows: 24, Cols: 80})

	time.Sleep(500 * time.Millisecond)
	s.RequireOutput("[S] Start/Stop", 6*time.Second)

	s.Resize(45, 160)
	time.Sleep(300 * time.Millisecond)

	s.WriteString("2")
	if !s.WaitForOutput("DOCKER IMAGES", 6*time.Second) {
		t.Errorf("expected Images view after wide resize")
	}

	s.Resize(20, 70)
	time.Sleep(300 * time.Millisecond)

	s.WriteString("q")
	if err := s.WaitExit(7 * time.Second); err != nil {
		t.Fatalf("expected clean exit code 0 on quit, got: %v", err)
	}
}

func TestE2E_InteractivePTY_SignalInterrupt(t *testing.T) {
	cmd := exec.Command(binaryPath, "-i", "200ms")
	s := startPTYSession(t, cmd, &pty.Winsize{Rows: 30, Cols: 100})

	time.Sleep(500 * time.Millisecond)
	s.RequireOutput("[S] Start/Stop", 6*time.Second)

	s.Write([]byte{0x03}) // Ctrl+C
	if err := s.WaitExit(7 * time.Second); err != nil {
		t.Fatalf("expected clean exit 0 upon Ctrl+C, got: %v", err)
	}
}

func TestE2E_InteractivePTY_SortingAndDetailsToggle(t *testing.T) {
	cmd := exec.Command(binaryPath, "-i", "200ms")
	s := startPTYSession(t, cmd, &pty.Winsize{Rows: 35, Cols: 120})

	time.Sleep(500 * time.Millisecond)
	s.RequireOutput("[S] Start/Stop", 6*time.Second)

	// Cycle sort with 'o'
	s.WriteString("o")
	time.Sleep(200 * time.Millisecond)

	// Toggle details card with 'v'
	s.WriteString("v")
	time.Sleep(200 * time.Millisecond)

	// Toggle details card back with 'v'
	s.WriteString("v")
	time.Sleep(200 * time.Millisecond)

	// Quit cleanly with 'q'
	s.WriteString("q")
	if err := s.WaitExit(7 * time.Second); err != nil {
		t.Fatalf("expected clean exit 0 upon quitting with 'q', got: %v", err)
	}
}
