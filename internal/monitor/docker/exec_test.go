package docker

import (
	"strings"
	"testing"
)

func TestGetContainerExecCommand_EmptyID(t *testing.T) {
	if cmd := GetContainerExecCommand("   "); cmd != nil {
		t.Errorf("expected nil for whitespace ID, got %#v", cmd)
	}
	if cmd := GetContainerExecCommandWithOptions("", ExecOptions{}); cmd != nil {
		t.Errorf("expected nil for empty ID, got %#v", cmd)
	}
}

func TestGetContainerExecCommand(t *testing.T) {
	containerID := "abc123def456"
	cmd := GetContainerExecCommand(containerID)
	if cmd == nil {
		t.Fatal("expected non-nil *exec.Cmd")
	}

	expectedArgs := []string{
		"docker", "exec", "-it", containerID, "sh", "-c",
		"if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
	}

	if len(cmd.Args) != len(expectedArgs) {
		t.Fatalf("expected %d args, got %d (%v)", len(expectedArgs), len(cmd.Args), cmd.Args)
	}

	for i, arg := range cmd.Args {
		if arg != expectedArgs[i] {
			t.Errorf("arg[%d] = %q, want %q", i, arg, expectedArgs[i])
		}
	}
}

func TestGetContainerExecCommandWithTerminal(t *testing.T) {
	containerID := "abc123def456"
	cmd := GetContainerExecCommandWithOptions(containerID, ExecOptions{
		Mode:     "auto",
		Terminal: "xterm-256color",
	})
	if cmd == nil {
		t.Fatal("expected non-nil *exec.Cmd")
	}

	want := []string{
		"docker", "exec", "-it", "-e", "TERM=xterm-256color", containerID,
		"sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
	}
	if strings.Join(cmd.Args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected command args: %#v", cmd.Args)
	}

	// Verify Mode: "root" preserves -e TERM while setting -u 0
	cmdRoot := GetContainerExecCommandWithOptions(containerID, ExecOptions{
		Mode:     "root",
		Terminal: "xterm-256color",
	})
	wantRoot := []string{
		"docker", "exec", "-it", "-u", "0", "-e", "TERM=xterm-256color", containerID,
		"sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
	}
	if strings.Join(cmdRoot.Args, "\x00") != strings.Join(wantRoot, "\x00") {
		t.Fatalf("unexpected root command args: %#v", cmdRoot.Args)
	}
}

func TestGetContainerExecCommandModes(t *testing.T) {
	cID := "c123"

	t.Run("attach", func(t *testing.T) {
		cmd := GetContainerExecCommandWithOptions(cID, ExecOptions{Mode: "attach"})
		want := []string{"docker", "attach", "--sig-proxy=false", cID}
		if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", cmd.Args, want)
		}
	})

	t.Run("sh", func(t *testing.T) {
		cmd := GetContainerExecCommandWithOptions(cID, ExecOptions{Mode: "sh"})
		want := []string{"docker", "exec", "-it", cID, "sh"}
		if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", cmd.Args, want)
		}
	})

	t.Run("zsh", func(t *testing.T) {
		cmd := GetContainerExecCommandWithOptions(cID, ExecOptions{Mode: "zsh"})
		want := []string{"docker", "exec", "-it", cID, "zsh"}
		if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", cmd.Args, want)
		}
	})

	t.Run("custom command", func(t *testing.T) {
		cmd := GetContainerExecCommandWithOptions(cID, ExecOptions{Mode: "custom", CustomCmd: "python3 app.py"})
		want := []string{"docker", "exec", "-it", cID, "sh", "-c", "python3 app.py"}
		if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", cmd.Args, want)
		}
	})

	t.Run("custom empty command falls back to sh", func(t *testing.T) {
		cmd := GetContainerExecCommandWithOptions(cID, ExecOptions{Mode: "custom", CustomCmd: "   "})
		want := []string{"docker", "exec", "-it", cID, "sh", "-c", "sh"}
		if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", cmd.Args, want)
		}
	})

	t.Run("explicit user", func(t *testing.T) {
		cmd := GetContainerExecCommandWithOptions(cID, ExecOptions{Mode: "sh", User: "1000"})
		want := []string{"docker", "exec", "-it", "-u", "1000", cID, "sh"}
		if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
			t.Errorf("got %v, want %v", cmd.Args, want)
		}
	})
}
