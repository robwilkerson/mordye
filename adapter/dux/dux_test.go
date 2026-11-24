package dux

import (
	"encoding/json"
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

// The colors each theme defines land in their slots unchanged when no bundled
// palette applies, so hue matching is checked against real themes. These are
// the exact matches the prototype found against each theme's official
// terminal port.
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
		in, err := Parse(settings(c.theme, c.flavor, c.accent), theme(t, c.theme))
		if err != nil {
			t.Fatal(err)
		}
		in.Palette = ""
		out, err := mordye.Complete(in)
		if err != nil {
			t.Fatal(err)
		}
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

// Every output field stays put across accents except the accent's own roles:
// primary, the text on it, and secondary with tertiary, which follows it as
// Dux renders it. In the terminal, only the cursor shows the accent; the 16
// colors keep their meaning.
func TestAccentMovesNothingButTheAccentRoles(t *testing.T) {
	fields := func(o mordye.Output) map[string]string {
		b, err := json.Marshal(struct {
			UI       mordye.UI         `json:"ui"`
			Terminal mordye.Terminal   `json:"terminal"`
			Sources  map[string]string `json:"sources"`
		}{o.UI, o.Terminal, o.Sources})
		if err != nil {
			t.Fatal(err)
		}
		var nested map[string]map[string]any
		if err := json.Unmarshal(b, &nested); err != nil {
			t.Fatal(err)
		}
		flat := map[string]string{}
		for group, m := range nested {
			for k, v := range m {
				if sub, ok := v.(map[string]any); ok {
					for slot, hex := range sub {
						flat[group+"."+k+"."+slot] = hex.(string)
					}
					continue
				}
				flat[group+"."+k] = v.(string)
			}
		}
		return flat
	}
	moves := map[string]bool{
		"ui.primary": true, "ui.on_primary": true, "ui.secondary": true, "ui.tertiary": true,
		"terminal.cursor": true, "terminal.cursor_text": true,
	}
	green := fields(parse(t, settings("catppuccin", "macchiato", "green"), "catppuccin"))
	blue := fields(parse(t, settings("catppuccin", "macchiato", "blue"), "catppuccin"))
	for field, hex := range green {
		if !moves[field] && blue[field] != hex {
			t.Errorf("%s moved with the accent: %s under green, %s under blue", field, hex, blue[field])
		}
	}
	if green["terminal.cursor"] != green["ui.primary"] || blue["terminal.cursor"] != blue["ui.primary"] {
		t.Errorf("cursor %s and %s, want each accent's primary %s and %s",
			green["terminal.cursor"], blue["terminal.cursor"], green["ui.primary"], blue["ui.primary"])
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

// optionSettings selects a fixed theme and records each option choice as Dux
// does for an options theme: a plain variant id, not a flavor and accent.
func optionSettings(choices map[string]string) []byte {
	b, _ := json.Marshal(map[string]any{"currentThemeName": "custom", "registryThemeVariants": choices})
	return b
}

func TestAnOptionsThemeUsesTheChosenOption(t *testing.T) {
	out := parse(t, optionSettings(map[string]string{"rosePine": "moon"}), "rose-pine")
	if out.Name != "rosePine-moon" || out.UI.Background != "#232136" {
		t.Errorf("name %q, background %s; want rosePine-moon on Moon's #232136", out.Name, out.UI.Background)
	}
	if out.Sources["terminal.background"] != "palette.rose-pine-moon" {
		t.Errorf("terminal from %q, want Moon's own palette", out.Sources["terminal.background"])
	}
}

// The default option keeps the theme's bare id, so its name doesn't change.
func TestAnOptionsThemeWithNoChoiceUsesItsDefault(t *testing.T) {
	for themeName, want := range map[string]string{"rose-pine": "rosePine", "gruvbox-material": "gruvboxMaterial"} {
		if got := parse(t, nil, themeName).Name; got != want {
			t.Errorf("%s: name %q, want %q", themeName, got, want)
		}
	}
}

// With no default and no choice, Dux draws the theme's own colors, with no
// option laid over them.
func TestAnOptionsThemeWithNoDefaultUsesItsOwnColors(t *testing.T) {
	var raw map[string]any
	if err := json.Unmarshal(theme(t, "rose-pine"), &raw); err != nil {
		t.Fatal(err)
	}
	delete(raw["variants"].(map[string]any), "default")
	b, _ := json.Marshal(raw)
	in, err := Parse(nil, b)
	if err != nil {
		t.Fatal(err)
	}
	// Rosé Pine's own colors leave the surfaces to its options.
	want := raw["dark"].(map[string]any)["primary"]
	if in.Name != "rosePine" || in.Colors["primary"] != want || in.Colors["background"] != "" {
		t.Errorf("name %q, primary %s, background %q; want rosePine with its own %s and no option's background",
			in.Name, in.Colors["primary"], in.Colors["background"], want)
	}
}

// Dawn is a light theme, and its palette waits for light support (#27).
func TestDawnNamesNoPaletteYet(t *testing.T) {
	in, err := Parse(optionSettings(map[string]string{"rosePine": "dawn"}), theme(t, "rose-pine"))
	if err != nil {
		t.Fatal(err)
	}
	if in.Name != "rosePine-dawn" || in.Palette != "" {
		t.Errorf("name %q, palette %q; want rosePine-dawn with no palette", in.Name, in.Palette)
	}
}

// Settings hold every theme's choice, so an options theme's plain id must not
// stop a different theme from parsing.
func TestAnOptionChoiceDoesNotBreakAnotherTheme(t *testing.T) {
	s := []byte(`{"currentThemeName": "custom", "registryThemeVariants": {
		"rosePine": "moon",
		"catppuccin": {"dark": {"flavor": "macchiato", "accent": "green"}}}}`)
	if got := parse(t, s, "catppuccin").Name; got != "catppuccin-macchiato" {
		t.Errorf("name %q, want catppuccin-macchiato", got)
	}
}

func TestRejectsAnUnknownOption(t *testing.T) {
	_, err := Parse(optionSettings(map[string]string{"rosePine": "eclipse"}), theme(t, "rose-pine"))
	if err == nil || !strings.Contains(err.Error(), "eclipse") {
		t.Errorf("err = %v, want one naming the unknown option", err)
	}
}

func TestRejectsAColorThatIsNotAString(t *testing.T) {
	for _, role := range []string{"primary", "surfaceText"} {
		bad := strings.Replace(string(theme(t, "nord")), `"`+role+`": "`, `"`+role+`": [1], "x": "`, 1)
		_, err := Parse(nil, []byte(bad))
		if err == nil || !strings.Contains(err.Error(), role) {
			t.Errorf("%s: err = %v, want one naming the role", role, err)
		}
	}
}

// Latte is a light flavor; its dark block is empty.
func TestAFlavorWithNoDarkColorsSaysSo(t *testing.T) {
	_, err := Parse(settings("catppuccin", "latte", "mauve"), theme(t, "catppuccin"))
	if err == nil || !strings.Contains(err.Error(), `flavor "latte" has no dark colors`) {
		t.Errorf("err = %v, want one naming the missing dark flavor", err)
	}
}

// A theme id with terminal escape codes could otherwise retitle or clear the
// terminal that shows the error.
func TestThemeTextInErrorsIsEscaped(t *testing.T) {
	id := `x\u001b]0;pwned\u0007`
	evil := strings.Replace(string(theme(t, "catppuccin")), `"id": "catppuccin"`, `"id": "`+id+`"`, 1)
	_, err := Parse(settings(id, "espresso", "mauve"), []byte(evil))
	if err == nil || strings.ContainsRune(err.Error(), '\x1b') {
		t.Errorf("err = %q, want an error with the escape code quoted", err)
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
