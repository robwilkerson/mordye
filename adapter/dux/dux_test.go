package dux

import (
	"os"
	"strings"
	"testing"

	"github.com/robwilkerson/mordye"
)

func theme(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// settings is a Dux settings file selecting a fixed theme, with an optional
// flavor and accent choice for it.
func settings(id, flavor, accent string) []byte {
	variants := ""
	if flavor != "" {
		variants = `"` + id + `": {"dark": {"flavor": "` + flavor + `", "accent": "` + accent + `"}}`
	}
	return []byte(`{"currentThemeName": "custom", "registryThemeVariants": {` + variants + `}}`)
}

func parse(t *testing.T, s []byte, themeName string) mordye.Output {
	t.Helper()
	in, err := Parse(s, theme(t, themeName))
	if err != nil {
		t.Fatal(err)
	}
	out, err := mordye.Complete(in)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func slot(o mordye.Output, name string) string {
	n := o.Terminal.Normal
	return map[string]string{
		"black": n.Black, "red": n.Red, "green": n.Green, "yellow": n.Yellow,
		"blue": n.Blue, "magenta": n.Magenta, "cyan": n.Cyan, "white": n.White,
	}[name]
}

// The colors each theme defines land in their slots unchanged. These are the
// exact matches the prototype found against each theme's official terminal
// port.
func TestThemesKeepTheirOwnColors(t *testing.T) {
	cases := []struct {
		theme, flavor, accent string
		want                  map[string]string
	}{
		{"catppuccin", "macchiato", "green",
			map[string]string{"black": "#494d64", "red": "#ed8796", "green": "#a6da95", "yellow": "#eed49f", "blue": "#8aadf4"}},
		{"catppuccin", "mocha", "mauve",
			map[string]string{"red": "#f38ba8", "green": "#a6e3a1", "yellow": "#f9e2af", "blue": "#89b4fa"}},
		{"tokyoNight", "", "", map[string]string{"red": "#f7768e", "blue": "#7aa2f7", "magenta": "#bb9af7"}},
		{"nord", "", "", map[string]string{"red": "#bf616a", "blue": "#81a1c1", "magenta": "#b48ead", "cyan": "#88c0d0"}},
		{"gruvbox-material", "", "", map[string]string{"red": "#e96962", "green": "#a8b665", "yellow": "#d7a657"}},
		{"dracula", "", "", map[string]string{"red": "#ff5555", "yellow": "#f1fa8c", "cyan": "#8be9fd"}},
		{"rose-pine", "", "", map[string]string{"red": "#eb6f92", "yellow": "#f6c177", "magenta": "#c4a7e7"}},
	}
	for _, c := range cases {
		out := parse(t, settings(c.theme, c.flavor, c.accent), c.theme)
		for name, hex := range c.want {
			if got := slot(out, name); got != hex {
				t.Errorf("%s %s: %s = %s (from %s), want %s",
					c.theme, c.flavor, name, got, out.Sources["terminal.normal."+name], hex)
			}
		}
	}
}

func TestSettingsChooseTheFlavorAndAccent(t *testing.T) {
	out := parse(t, settings("catppuccin", "macchiato", "peach"), "catppuccin")
	if out.Name != "catppuccin-macchiato" || out.UI.Primary != "#f5a97f" {
		t.Errorf("name %q, primary %s; want catppuccin-macchiato and peach #f5a97f", out.Name, out.UI.Primary)
	}
}

func TestAccentMovesOnlyTheAccent(t *testing.T) {
	green := parse(t, settings("catppuccin", "macchiato", "green"), "catppuccin")
	peach := parse(t, settings("catppuccin", "macchiato", "peach"), "catppuccin")
	if peach.Terminal.Normal != green.Terminal.Normal || peach.Terminal.Bright != green.Terminal.Bright {
		t.Errorf("the accent moved the terminal colors:\n green %+v\n peach %+v", green.Terminal.Normal, peach.Terminal.Normal)
	}
}

func TestNoChoiceUsesTheThemeDefaults(t *testing.T) {
	out := parse(t, settings("catppuccin", "", ""), "catppuccin")
	if out.Name != "catppuccin-mocha" || out.UI.Primary != "#cba6f7" {
		t.Errorf("name %q, primary %s; want the defaults, catppuccin-mocha with mauve #cba6f7", out.Name, out.UI.Primary)
	}
}

// Themes disagree on which text role holds the plain text color: Nord's
// backgroundText is teal, Tokyo Night's surfaceText is. The plain one becomes
// on_surface either way.
func TestTextIsTheUntintedTextRole(t *testing.T) {
	for name, want := range map[string]string{"nord": "#eceff4", "tokyoNight": "#d5d6db"} {
		if got := parse(t, settings(name, "", ""), name).UI.OnSurface; got != want {
			t.Errorf("%s: on_surface = %s, want %s", name, got, want)
		}
	}
}

func TestRejectsAWallpaperTheme(t *testing.T) {
	_, err := Parse([]byte(`{"currentThemeName": "dynamic"}`), theme(t, "nord"))
	if err == nil || !strings.Contains(err.Error(), "dynamic") {
		t.Errorf("err = %v, want one naming the non-fixed theme", err)
	}
}

func TestUnsafeThemeIDIsRejected(t *testing.T) {
	evil := strings.Replace(string(theme(t, "nord")), `"id": "nord"`, `"id": "nord\"\n[shell]"`, 1)
	in, err := Parse(settings("nord", "", ""), []byte(evil))
	if err == nil {
		_, err = mordye.Complete(in)
	}
	if err == nil || !strings.Contains(err.Error(), `name "nord\"\n[shell]"`) {
		t.Errorf("err = %v, want one rejecting the theme's name", err)
	}
}

func TestRejectsAnUnknownFlavorOrAccent(t *testing.T) {
	for _, s := range [][]byte{
		settings("catppuccin", "espresso", "green"),
		settings("catppuccin", "macchiato", "chartreuse"),
	} {
		if _, err := Parse(s, theme(t, "catppuccin")); err == nil {
			t.Errorf("Parse(%s) succeeded, want an error", s)
		}
	}
}
