package theme

import (
	"testing"
)

func TestThemeStyles(t *testing.T) {
	styles := []struct {
		name string
		out  string
	}{
		{"HeaderStyle", HeaderStyle.Render("header")},
		{"ActiveTabStyle", ActiveTabStyle.Render("active")},
		{"InactiveTabStyle", InactiveTabStyle.Render("inactive")},
		{"SelectedRowStyle", SelectedRowStyle.Render("row")},
		{"TitleStyle", TitleStyle.Render("title")},
		{"DimStyle", DimStyle.Render("dim")},
		{"BoldStyle", BoldStyle.Render("bold")},
		{"GreenStyle", GreenStyle.Render("green")},
		{"YellowStyle", YellowStyle.Render("yellow")},
		{"RedStyle", RedStyle.Render("red")},
		{"CyanStyle", CyanStyle.Render("cyan")},
		{"FooterStyle", FooterStyle.Render("footer")},
		{"BoxStyle", BoxStyle.Render("box")},
	}

	for _, s := range styles {
		if s.out == "" {
			t.Errorf("style %s rendered empty string", s.name)
		}
	}
}
