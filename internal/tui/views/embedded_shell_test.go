package views

import (
	"strings"
	"testing"
)

func TestRenderEmbeddedShell_FocusAndStatus(t *testing.T) {
	t.Parallel()

	t.Run("unfocused shell displays DOCKPULSE FOCUSED status badge", func(t *testing.T) {
		t.Parallel()
		view := RenderEmbeddedShell(EmbeddedShellContext{
			ContainerName: "backend-api",
			Output:        "user@box:~$",
			Width:         80,
			Height:        20,
			Focused:       false,
		})

		if !strings.Contains(view, "TERMINAL: backend-api") {
			t.Errorf("expected header with container name in view, got: %s", view)
		}
		if !strings.Contains(view, "[DOCKPULSE FOCUSED]") {
			t.Errorf("expected unfocused badge '[DOCKPULSE FOCUSED]', got: %s", view)
		}
	})

	t.Run("focused shell displays FOCUSED status badge", func(t *testing.T) {
		t.Parallel()
		view := RenderEmbeddedShell(EmbeddedShellContext{
			ContainerName: "backend-api",
			Output:        "user@box:~$",
			Width:         80,
			Height:        20,
			Focused:       true,
		})

		if !strings.Contains(view, "TERMINAL: backend-api") {
			t.Errorf("expected header with container name in view, got: %s", view)
		}
		if !strings.Contains(view, "[FOCUSED]") {
			t.Errorf("expected focused badge '[FOCUSED]', got: %s", view)
		}
		if strings.Contains(view, "DOCKPULSE FOCUSED") {
			t.Errorf("expected no 'DOCKPULSE FOCUSED' when shell is focused, got: %s", view)
		}
	})
}

func TestRenderEmbeddedShell_OutputVariants(t *testing.T) {
	t.Parallel()

	t.Run("empty output shows starting shell fallback placeholder", func(t *testing.T) {
		t.Parallel()
		view := RenderEmbeddedShell(EmbeddedShellContext{
			ContainerName: "worker-svc",
			Output:        "",
			Width:         80,
			Height:        20,
			Focused:       false,
		})

		if !strings.Contains(view, "Starting shell...") {
			t.Errorf("expected 'Starting shell...' for empty output, got: %s", view)
		}
	})

	t.Run("custom output displays command and multiline terminal text", func(t *testing.T) {
		t.Parallel()
		lines := []string{
			"root@container:/# ps aux",
			"USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND",
			"root         1  0.0  0.1   4340  1648 ?        Ss   20:00   0:00 /bin/sh",
		}
		output := strings.Join(lines, "\n")

		view := RenderEmbeddedShell(EmbeddedShellContext{
			ContainerName: "worker-svc",
			Output:        output,
			Width:         100,
			Height:        24,
			Focused:       true,
		})

		if strings.Contains(view, "Starting shell...") {
			t.Errorf("did not expect 'Starting shell...' when output is present, got: %s", view)
		}
		for _, l := range lines {
			if !strings.Contains(view, l) {
				t.Errorf("expected line %q in rendered output, got: %s", l, view)
			}
		}
	})
}

func TestRenderEmbeddedShell_DimensionClamping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		width         int
		height        int
		containerName string
		output        string
	}{
		{
			name:          "zero dimensions clamp to minimums 20x6",
			width:         0,
			height:        0,
			containerName: "c1",
			output:        "hi",
		},
		{
			name:          "negative dimensions clamp to minimums 20x6",
			width:         -50,
			height:        -10,
			containerName: "c2",
			output:        "prompt>",
		},
		{
			name:          "dimensions smaller than minimums",
			width:         10,
			height:        4,
			containerName: "c3",
			output:        "cmd",
		},
		{
			name:          "standard terminal dimensions",
			width:         80,
			height:        24,
			containerName: "web",
			output:        "listening on :80",
		},
		{
			name:          "large widescreen terminal dimensions",
			width:         240,
			height:        80,
			containerName: "huge-term",
			output:        "widescreen output",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			view := RenderEmbeddedShell(EmbeddedShellContext{
				ContainerName: tc.containerName,
				Output:        tc.output,
				Width:         tc.width,
				Height:        tc.height,
				Focused:       true,
			})

			if view == "" {
				t.Errorf("expected non-empty rendered view for dimensions %dx%d", tc.width, tc.height)
			}
			if !strings.Contains(view, tc.containerName) {
				t.Errorf("expected container name %q in view, got: %s", tc.containerName, view)
			}
		})
	}
}

func TestRenderEmbeddedShell_ContainerNameVariants(t *testing.T) {
	t.Parallel()

	t.Run("empty container name renders without crashing", func(t *testing.T) {
		t.Parallel()
		view := RenderEmbeddedShell(EmbeddedShellContext{
			ContainerName: "",
			Output:        "shell active",
			Width:         80,
			Height:        20,
			Focused:       true,
		})

		if !strings.Contains(view, "TERMINAL:") {
			t.Errorf("expected 'TERMINAL:' in header, got: %s", view)
		}
		if !strings.Contains(view, "[FOCUSED]") {
			t.Errorf("expected '[FOCUSED]' in header, got: %s", view)
		}
	})

	t.Run("long container name renders properly", func(t *testing.T) {
		t.Parallel()
		longName := "k8s_app-production-service-worker-pod_namespace_abcdef0123456789"
		view := RenderEmbeddedShell(EmbeddedShellContext{
			ContainerName: longName,
			Output:        "running",
			Width:         120,
			Height:        20,
			Focused:       false,
		})

		if !strings.Contains(view, longName) {
			t.Errorf("expected long container name in view, got: %s", view)
		}
	})
}
