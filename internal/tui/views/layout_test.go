package views

import (
	"strings"
	"testing"

	"github.com/AndreyJVM/container-ctl/internal/monitor"
	"github.com/AndreyJVM/container-ctl/internal/testutil"
)

func TestRenderHeader(t *testing.T) {
	t.Parallel()

	t.Run("renders populated header context", func(t *testing.T) {
		t.Parallel()
		hdr := RenderHeader(HeaderContext{
			Version:               "v0.9.1",
			LastUpdate:            "12:34:56",
			DockerInfo:            monitor.DockerSummary{RunningContainers: 5, StoppedContainers: 2, ImagesCount: 14, Version: "27.5.1"},
			FormattedDockerStatus: "Running (Active)",
		})

		testutil.AssertContainsAll(t, hdr,
			"Dockpulse",
			"v0.9.1",
			"12:34:56",
			"Containers: 5 active, 2 stopped",
			"Images: 14",
			"Engine: 27.5.1",
			"Running (Active)",
		)
	})

	t.Run("falls back to default version when version string is empty", func(t *testing.T) {
		t.Parallel()
		hdr := RenderHeader(HeaderContext{
			Version:               "",
			LastUpdate:            "00:00:00",
			DockerInfo:            monitor.DockerSummary{},
			FormattedDockerStatus: "Stopped",
		})

		if !strings.Contains(hdr, "v0.4.0") {
			t.Errorf("expected fallback version 'v0.4.0' when Version is empty, got: %s", hdr)
		}
	})
}

func TestRenderTabs(t *testing.T) {
	t.Parallel()

	tabs := []string{"1: Containers", "2: Images", "3: Compose", "4: Networks", "5: Volumes", "6: Swarm"}

	t.Run("active tab is enclosed in brackets and inactive have spaces", func(t *testing.T) {
		t.Parallel()
		tabBar := RenderTabs(tabs, 2, 80)

		if !strings.Contains(tabBar, "[2: Images]") {
			t.Errorf("expected active tab '[2: Images]' in tab bar, got: %s", tabBar)
		}
		if strings.Contains(tabBar, "[1: Containers]") {
			t.Errorf("expected inactive tab 1 without brackets, got: %s", tabBar)
		}
		if !strings.Contains(tabBar, " 1: Containers ") {
			t.Errorf("expected inactive tab 1 with spaces padding, got: %s", tabBar)
		}
	})

	t.Run("activeTab out of range leaves all tabs inactive", func(t *testing.T) {
		t.Parallel()
		tabBar := RenderTabs(tabs, 99, 80)
		for _, tab := range tabs {
			bracketed := "[" + tab + "]"
			if strings.Contains(tabBar, bracketed) {
				t.Errorf("did not expect bracketed tab %s when activeTab is 99, got: %s", bracketed, tabBar)
			}
		}
	})

	t.Run("border width clamping", func(t *testing.T) {
		t.Parallel()

		// Clamped to minimum 70
		barNarrow := RenderTabs(tabs, 1, 40)
		expectedNarrowBorder := strings.Repeat("─", 70)
		if !strings.Contains(barNarrow, expectedNarrowBorder) {
			t.Errorf("expected border clamped to length 70, got: %s", barNarrow)
		}

		// Width scales with large width
		barWide := RenderTabs(tabs, 1, 110)
		expectedWideBorder := strings.Repeat("─", 110)
		if !strings.Contains(barWide, expectedWideBorder) {
			t.Errorf("expected border scaled to length 110, got: %s", barWide)
		}
	})
}

func TestRenderFooter_Modals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		ctx          FooterContext
		wantContains []string
	}{
		{
			name:         "compose down confirmation",
			ctx:          FooterContext{ActiveModal: "compose_down", Width: 80},
			wantContains: []string{"[Y]", "Confirm (Down)", "[N / Esc]", "Cancel"},
		},
		{
			name:         "remove container confirmation",
			ctx:          FooterContext{ActiveModal: "remove_container", Width: 80},
			wantContains: []string{"[Y]", "Confirm (Remove)", "[N / Esc]", "Cancel"},
		},
		{
			name:         "remove image confirmation",
			ctx:          FooterContext{ActiveModal: "remove_image", Width: 80},
			wantContains: []string{"[Y]", "Confirm (Remove Image)", "[N / Esc]", "Cancel"},
		},
		{
			name:         "exec selection",
			ctx:          FooterContext{ActiveModal: "exec", Width: 80},
			wantContains: []string{"[1-7 / Enter]", "Select Action", "[Esc / Q]", "Cancel"},
		},
		{
			name:         "prune selection menu",
			ctx:          FooterContext{ActiveModal: "prune", PruneConfirm: 0, Width: 80},
			wantContains: []string{"[1-4]", "Select Action", "[Esc / Q]", "Close"},
		},
		{
			name:         "prune confirmation prompt",
			ctx:          FooterContext{ActiveModal: "prune", PruneConfirm: 1, Width: 80},
			wantContains: []string{"[Y]", "Confirm Prune", "[N / Esc]", "Cancel"},
		},
		{
			name:         "generic logs modal scroll and close",
			ctx:          FooterContext{ActiveModal: "logs", Width: 80},
			wantContains: []string{"[Esc / Q]", "Close", "[↑ / ↓ / PgUp / PgDn]", "Scroll"},
		},
		{
			name:         "generic inspect modal scroll and close",
			ctx:          FooterContext{ActiveModal: "inspect", Width: 80},
			wantContains: []string{"[Esc / Q]", "Close", "[↑ / ↓ / PgUp / PgDn]", "Scroll"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			footer := RenderFooter(tc.ctx)
			for _, want := range tc.wantContains {
				if !strings.Contains(footer, want) {
					t.Errorf("expected footer to contain %q, got: %s", want, footer)
				}
			}
		})
	}
}

func TestRenderFooter_EmbeddedShell(t *testing.T) {
	t.Parallel()

	t.Run("focused embedded shell hints", func(t *testing.T) {
		t.Parallel()
		footer := RenderFooter(FooterContext{
			ShellActive:  true,
			ShellFocused: true,
			Width:        80,
		})
		testutil.AssertContainsAll(t, footer, "[Ctrl+]]", "Return to Dockpulse", "[Ctrl+D]", "Exit shell")
	})

	t.Run("unfocused embedded shell hints", func(t *testing.T) {
		t.Parallel()
		footer := RenderFooter(FooterContext{
			ShellActive:  true,
			ShellFocused: false,
			Width:        80,
		})
		testutil.AssertContainsAll(t, footer, "[E]", "Focus terminal", "[Ctrl+X]", "Close terminal", "[Q]", "Quit")
	})
}

func TestRenderFooter_TabsAndFiltering(t *testing.T) {
	t.Parallel()

	t.Run("Tab 1 Containers active filter input", func(t *testing.T) {
		t.Parallel()
		footer := RenderFooter(FooterContext{ActiveTab: 1, IsFiltering: true, Width: 80})
		testutil.AssertContainsAll(t, footer, "[Enter]", "Apply Filter", "[Esc]", "Cancel")
	})

	t.Run("Tab 1 Containers with applied filter query", func(t *testing.T) {
		t.Parallel()
		footer := RenderFooter(FooterContext{ActiveTab: 1, FilterQuery: "postgres", Width: 80})
		testutil.AssertContainsAll(t, footer, "[Esc]", "Clear Filter", "[↑ / ↓]", "Select", "[L]", "Logs")
	})

	t.Run("Tab 1 Containers narrow compact width (< 95)", func(t *testing.T) {
		t.Parallel()
		footer := RenderFooter(FooterContext{ActiveTab: 1, Width: 80})
		testutil.AssertContainsAll(t, footer, "[↑ / ↓]", "Select", "[K]", "Kill", "[x]", "Remove", "[/]", "Search", "[O]", "Sort")
	})

	t.Run("Tab 1 Containers wide width (>= 160) single line", func(t *testing.T) {
		t.Parallel()
		footer := RenderFooter(FooterContext{ActiveTab: 1, Width: 180})
		testutil.AssertContainsAll(t, footer, "[↑ / ↓]", "Select", "[/]", "Search", "[O]", "Sort", "[K]", "Kill", "[x]", "Remove")
		// In wide mode (>= 160), content is rendered in a single consolidated row (fewer lines than 2-row standard footer)
		footerStd := RenderFooter(FooterContext{ActiveTab: 1, Width: 100})
		linesWide := strings.Split(strings.TrimSpace(footer), "\n")
		linesStd := strings.Split(strings.TrimSpace(footerStd), "\n")
		if len(linesWide) >= len(linesStd) {
			t.Errorf("expected wide footer to have fewer rows than standard two-row footer, got %d vs %d", len(linesWide), len(linesStd))
		}
	})

	t.Run("Tab 2 Images active filter vs applied filter vs normal", func(t *testing.T) {
		t.Parallel()

		fFiltering := RenderFooter(FooterContext{ActiveTab: 2, IsFiltering: true, Width: 80})
		testutil.AssertContainsAll(t, fFiltering, "Apply Filter", "Cancel")

		fApplied := RenderFooter(FooterContext{ActiveTab: 2, FilterQuery: "redis", Width: 80})
		testutil.AssertContainsAll(t, fApplied, "Reset Filter", "Remove", "Prune")
		if strings.Contains(fApplied, "Filter Unused") {
			t.Errorf("expected Filter Unused absent when filter query is applied, got: %s", fApplied)
		}

		fNormal := RenderFooter(FooterContext{ActiveTab: 2, Width: 80})
		testutil.AssertContainsAll(t, fNormal, "Search", "Filter Unused", "Remove", "Prune", "Quit")
	})

	t.Run("Tab 3 Compose active filter vs applied filter vs normal", func(t *testing.T) {
		t.Parallel()

		fFiltering := RenderFooter(FooterContext{ActiveTab: 3, IsFiltering: true, Width: 80})
		testutil.AssertContainsAll(t, fFiltering, "Apply Filter", "Cancel")

		fApplied := RenderFooter(FooterContext{ActiveTab: 3, FilterQuery: "api", Width: 80})
		testutil.AssertContainsAll(t, fApplied, "Select Stack", "Reset Filter", "Start", "Stop", "Restart", "Down")

		fNormal := RenderFooter(FooterContext{ActiveTab: 3, Width: 80})
		testutil.AssertContainsAll(t, fNormal, "Select Stack", "Search", "Logs", "Start", "Stop", "Down")
	})

	t.Run("Tab 4 and 5 Networks and Volumes", func(t *testing.T) {
		t.Parallel()

		for _, tab := range []int{4, 5} {
			fFiltering := RenderFooter(FooterContext{ActiveTab: tab, IsFiltering: true, Width: 80})
			testutil.AssertContainsAll(t, fFiltering, "Apply Filter", "Cancel")

			fApplied := RenderFooter(FooterContext{ActiveTab: tab, FilterQuery: "bridge", Width: 80})
			testutil.AssertContainsAll(t, fApplied, "Select", "Inspect", "Reset Filter", "Copy", "Prune")

			fNormal := RenderFooter(FooterContext{ActiveTab: tab, Width: 80})
			testutil.AssertContainsAll(t, fNormal, "Select", "Inspect", "Search", "Copy", "Prune", "Refresh")
		}
	})

	t.Run("Tab 6 Swarm cluster footer", func(t *testing.T) {
		t.Parallel()
		footer := RenderFooter(FooterContext{ActiveTab: 6, Width: 80})
		testutil.AssertContainsAll(t, footer, "Select Service", "Service Logs", "Refresh", "Help", "Quit")
	})

	t.Run("Default tab fallback with Swarm visible vs hidden", func(t *testing.T) {
		t.Parallel()

		fSwarmHidden := RenderFooter(FooterContext{ActiveTab: 99, SwarmVisible: false, Width: 80})
		testutil.AssertContainsAll(t, fSwarmHidden, "[1-5 / Tab / ← →]", "Tabs", "Refresh", "Help", "Quit")

		fSwarmVisible := RenderFooter(FooterContext{ActiveTab: 99, SwarmVisible: true, Width: 80})
		testutil.AssertContainsAll(t, fSwarmVisible, "[1-6 / Tab / ← →]", "Tabs", "Refresh", "Help", "Quit")
	})
}

func TestRenderFooter_DividerWidth(t *testing.T) {
	t.Parallel()

	t.Run("clamps divider to minimum width 70", func(t *testing.T) {
		t.Parallel()
		footerNarrow := RenderFooter(FooterContext{ActiveTab: 1, Width: 35})
		expectedDivider := strings.Repeat("─", 70)
		if !strings.Contains(footerNarrow, expectedDivider) {
			t.Errorf("expected divider clamped to 70 chars, got: %s", footerNarrow)
		}
	})

	t.Run("scales divider with wide width", func(t *testing.T) {
		t.Parallel()
		footerWide := RenderFooter(FooterContext{ActiveTab: 1, Width: 125})
		expectedDivider := strings.Repeat("─", 125)
		if !strings.Contains(footerWide, expectedDivider) {
			t.Errorf("expected divider scaled to 125 chars, got: %s", footerWide)
		}
	})
}
