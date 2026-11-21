package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGaugeStyle(t *testing.T) {
	tests := []struct {
		percent  float64
		expected string
	}{
		{20.0, "green"},
		{49.9, "green"},
		{50.0, "yellow"},
		{79.9, "yellow"},
		{80.0, "red"},
		{95.0, "red"},
	}

	for _, tt := range tests {
		style := GaugeStyle(tt.percent)
		sample := style.Render("test")
		if sample == "" {
			t.Errorf("GaugeStyle(%.1f) returned empty render", tt.percent)
		}
	}
}

func TestRenderGauge(t *testing.T) {
	// Zero or negative width
	if RenderGauge(50.0, 0, true) != "" {
		t.Error("expected empty string for 0 width")
	}
	if RenderGauge(50.0, -5, true) != "" {
		t.Error("expected empty string for negative width")
	}

	// Stopped container
	stopped := RenderGauge(50.0, 10, false)
	if !strings.Contains(stopped, "──────────") {
		t.Errorf("expected dashed line for stopped container, got: %s", stopped)
	}

	// Running container 50% on 10 blocks -> visual width should be 12 (10 blocks + 2 brackets)
	running50 := RenderGauge(50.0, 10, true)
	if lipgloss.Width(running50) != 12 {
		t.Errorf("expected visual width 12 for 10-block gauge, got: %d (%s)", lipgloss.Width(running50), running50)
	}

	// 0% running has at least 10 empty blocks
	running0 := RenderPlainGauge(0.0, 10, true)
	if running0 != "[░░░░░░░░░░]" {
		t.Errorf("expected [░░░░░░░░░░], got: %s", running0)
	}

	// 100% running has 10 filled blocks
	running100 := RenderPlainGauge(100.0, 10, true)
	if running100 != "[██████████]" {
		t.Errorf("expected [██████████], got: %s", running100)
	}

	// 1% running shows at least 1 block
	running1 := RenderPlainGauge(1.0, 10, true)
	if !strings.HasPrefix(running1, "[█") {
		t.Errorf("expected at least 1 filled block for 1%%, got: %s", running1)
	}
}

func TestRenderPlainGauge(t *testing.T) {
	if RenderPlainGauge(50.0, 0, true) != "" {
		t.Error("expected empty string for 0 width")
	}
	stopped := RenderPlainGauge(50.0, 5, false)
	if stopped != "[─────]" {
		t.Errorf("expected [─────], got: %s", stopped)
	}
	half := RenderPlainGauge(50.0, 6, true)
	if half != "[███░░░]" {
		t.Errorf("expected [███░░░], got: %s", half)
	}
}

func TestFormatMB(t *testing.T) {
	tests := []struct {
		mb       float64
		expected string
	}{
		{-10, "0 MB"},
		{0, "0 MB"},
		{256.5, "256.5 MB"},
		{1024.0, "1.00 GB"},
		{2048.0, "2.00 GB"},
		{1536.0, "1.50 GB"},
	}

	for _, tt := range tests {
		actual := FormatMB(tt.mb)
		if actual != tt.expected {
			t.Errorf("FormatMB(%.1f) = %q, expected %q", tt.mb, actual, tt.expected)
		}
	}
}
