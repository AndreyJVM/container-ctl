package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestThemeStyles_RenderingContent(t *testing.T) {
	t.Parallel()

	testStyles := []struct {
		name  string
		style lipgloss.Style
	}{
		{"HeaderStyle", HeaderStyle},
		{"ActiveTabStyle", ActiveTabStyle},
		{"InactiveTabStyle", InactiveTabStyle},
		{"SelectedRowStyle", SelectedRowStyle},
		{"TitleStyle", TitleStyle},
		{"DimStyle", DimStyle},
		{"BoldStyle", BoldStyle},
		{"GreenStyle", GreenStyle},
		{"YellowStyle", YellowStyle},
		{"RedStyle", RedStyle},
		{"CyanStyle", CyanStyle},
		{"FooterStyle", FooterStyle},
		{"BoxStyle", BoxStyle},
	}

	for _, ts := range testStyles {
		ts := ts
		t.Run(ts.name+" renders text payload", func(t *testing.T) {
			t.Parallel()
			payload := "dockpulse-styled-text"
			rendered := ts.style.Render(payload)

			if !strings.Contains(rendered, payload) {
				t.Errorf("expected rendered style %s to contain %q, got: %q", ts.name, payload, rendered)
			}
		})
	}
}

func TestThemeStyles_EmptyAndMultilineRendering(t *testing.T) {
	t.Parallel()

	t.Run("empty string rendering produces safe output without panic", func(t *testing.T) {
		t.Parallel()
		for _, s := range []lipgloss.Style{HeaderStyle, ActiveTabStyle, InactiveTabStyle, SelectedRowStyle, TitleStyle, DimStyle, BoldStyle, GreenStyle, YellowStyle, RedStyle, CyanStyle, FooterStyle, BoxStyle} {
			_ = s.Render("")
		}
	})

	t.Run("multiline string rendering preserves all lines", func(t *testing.T) {
		t.Parallel()
		multiline := "line one\nline two\nline three"
		for _, s := range []lipgloss.Style{DimStyle, BoldStyle, BoxStyle, FooterStyle} {
			rendered := s.Render(multiline)
			for _, line := range strings.Split(multiline, "\n") {
				if !strings.Contains(rendered, line) {
					t.Errorf("expected rendered multiline to contain %q, got: %s", line, rendered)
				}
			}
		}
	})
}

func TestThemeStyles_StyleProperties(t *testing.T) {
	t.Parallel()

	t.Run("bold flags correspond to style design", func(t *testing.T) {
		t.Parallel()

		boldStyles := map[string]lipgloss.Style{
			"HeaderStyle":      HeaderStyle,
			"ActiveTabStyle":   ActiveTabStyle,
			"SelectedRowStyle": SelectedRowStyle,
			"TitleStyle":       TitleStyle,
			"BoldStyle":        BoldStyle,
			"GreenStyle":       GreenStyle,
			"YellowStyle":      YellowStyle,
			"RedStyle":         RedStyle,
		}
		for name, style := range boldStyles {
			if !style.GetBold() {
				t.Errorf("expected %s to have Bold=true", name)
			}
		}

		nonBoldStyles := map[string]lipgloss.Style{
			"InactiveTabStyle": InactiveTabStyle,
			"DimStyle":         DimStyle,
			"CyanStyle":        CyanStyle,
			"FooterStyle":      FooterStyle,
		}
		for name, style := range nonBoldStyles {
			if style.GetBold() {
				t.Errorf("expected %s to have Bold=false", name)
			}
		}
	})

	t.Run("tab and header horizontal padding", func(t *testing.T) {
		t.Parallel()

		paddedStyles := map[string]lipgloss.Style{
			"HeaderStyle":      HeaderStyle,
			"ActiveTabStyle":   ActiveTabStyle,
			"InactiveTabStyle": InactiveTabStyle,
		}
		for name, style := range paddedStyles {
			top, right, bottom, left := style.GetPadding()
			if top != 0 || bottom != 0 || left != 1 || right != 1 {
				t.Errorf("expected %s to have padding (0, 1), got top=%d, right=%d, bottom=%d, left=%d",
					name, top, right, bottom, left)
			}
		}
	})

	t.Run("box style rounded border and padding", func(t *testing.T) {
		t.Parallel()

		border, top, right, bottom, left := BoxStyle.GetBorder()
		if !top || !right || !bottom || !left {
			t.Errorf("expected BoxStyle to have all border sides enabled, got top=%v, right=%v, bottom=%v, left=%v",
				top, right, bottom, left)
		}
		if border != lipgloss.RoundedBorder() {
			t.Errorf("expected BoxStyle to use RoundedBorder")
		}

		pTop, pRight, pBottom, pLeft := BoxStyle.GetPadding()
		if pTop != 1 || pBottom != 1 || pLeft != 2 || pRight != 2 {
			t.Errorf("expected BoxStyle padding (1, 2), got top=%d, right=%d, bottom=%d, left=%d",
				pTop, pRight, pBottom, pLeft)
		}
	})

	t.Run("margins for title and footer", func(t *testing.T) {
		t.Parallel()

		if TitleStyle.GetMarginTop() != 1 || TitleStyle.GetMarginBottom() != 0 {
			t.Errorf("expected TitleStyle MarginTop=1 and MarginBottom=0, got top=%d, bottom=%d",
				TitleStyle.GetMarginTop(), TitleStyle.GetMarginBottom())
		}
		if FooterStyle.GetMarginTop() != 1 {
			t.Errorf("expected FooterStyle MarginTop=1, got %d", FooterStyle.GetMarginTop())
		}
	})
}

func TestBoxStyle_BorderEnclosure(t *testing.T) {
	t.Parallel()

	content := "modal confirmation dialog"
	rendered := BoxStyle.Render(content)

	// Rounded corners check
	for _, corner := range []string{"╭", "╮", "╰", "╯"} {
		if !strings.Contains(rendered, corner) {
			t.Errorf("expected rounded corner %q in rendered box, got: %s", corner, rendered)
		}
	}

	if !strings.Contains(rendered, content) {
		t.Errorf("expected box to enclose content %q, got: %s", content, rendered)
	}
}
