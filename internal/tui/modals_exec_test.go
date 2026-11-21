package tui

import (
	"os/exec"
	"testing"
	"time"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleExecModalInput(t *testing.T) {
	originalStart := startEmbeddedShell
	startEmbeddedShell = func(_ *exec.Cmd, _, _ int) (embeddedShellSession, error) {
		return &fakeEmbeddedShell{}, nil
	}
	defer func() { startEmbeddedShell = originalStart }()

	m := NewModel(2*time.Second, 1, "v1.0.0")
	m.dockerInfo.Containers = []monitor.ContainerItem{
		testutil.AContainer(testutil.WithID("c1"), testutil.WithName("test-box"), testutil.WithState("running", "Up")),
	}
	m.activeModal = ModalExec
	m.execTargetID = "c1"
	m.execTargetName = "test-box"

	// 1. Navigation up / down with wrapping
	m.execCursorIdx = 0
	res, _ := m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = res.(Model)
	if m.execCursorIdx != 6 {
		t.Errorf("expected cursor wrapped to 6 on up/k, got %d", m.execCursorIdx)
	}

	res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = res.(Model)
	if m.execCursorIdx != 0 {
		t.Errorf("expected cursor wrapped to 0 on down/j, got %d", m.execCursorIdx)
	}

	// 2. Custom input typing, backspace, and esc
	res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = res.(Model)
	if !m.execIsCustomInput {
		t.Fatalf("expected execIsCustomInput=true after pressing 'c'")
	}

	for _, r := range "ls -la" {
		res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = res.(Model)
	}
	if m.execCustomCmd != "ls -la" {
		t.Errorf("expected custom cmd 'ls -la', got %q", m.execCustomCmd)
	}

	res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyBackspace})
	m = res.(Model)
	if m.execCustomCmd != "ls -l" {
		t.Errorf("expected custom cmd 'ls -l' after backspace, got %q", m.execCustomCmd)
	}

	res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyEsc})
	m = res.(Model)
	if m.execIsCustomInput {
		t.Errorf("expected execIsCustomInput=false after esc")
	}

	// 3. Enter with empty custom command exits custom input
	m.execIsCustomInput = true
	m.execCustomCmd = "   "
	res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if m.execIsCustomInput {
		t.Errorf("expected execIsCustomInput=false after enter with whitespace")
	}

	// 4. Cancel modal with 'q'
	res, _ = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = res.(Model)
	if m.activeModal != "" || m.execTargetID != "" {
		t.Errorf("expected modal closed on 'q'")
	}

	// 5. Select embedded shell via '1'
	m.activeModal = ModalExec
	m.execTargetID = "c1"
	m.execTargetName = "test-box"
	res, cmd := m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected cmd from launchEmbeddedShell")
	}

	// 6. Select sh via '2'
	m.activeModal = ModalExec
	m.execTargetID = "c1"
	m.execTargetName = "test-box"
	res, cmd = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected cmd from launchExecProcess")
	}

	// 7. Select options via Enter
	m.activeModal = ModalExec
	m.execTargetID = "c1"
	m.execTargetName = "test-box"
	m.execCursorIdx = 4 // root
	res, cmd = m.handleExecModalInput(tea.KeyMsg{Type: tea.KeyEnter})
	m = res.(Model)
	if cmd == nil {
		t.Errorf("expected cmd from enter on root option")
	}
}
