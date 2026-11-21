package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
