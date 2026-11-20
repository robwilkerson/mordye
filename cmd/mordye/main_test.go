package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/robwilkerson/mordye"
)

func TestRunCompletesAPaletteFromStdin(t *testing.T) {
	in := `{"version": 1, "colors": {"primary": "#a6da95", "on_surface": "#cad3f5", "error": "#ed8796"}}`
	var out bytes.Buffer
	if err := run(strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Terminal struct {
			Normal struct{ Red string } `json:"normal"`
		} `json:"terminal"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	if got.Terminal.Normal.Red != "#ed8796" {
		t.Errorf("red = %q, want #ed8796", got.Terminal.Normal.Red)
	}
}

func TestDuxReadsTheFilesItIsGiven(t *testing.T) {
	settings := t.TempDir() + "/settings.json"
	body := `{"currentThemeName": "custom", "registryThemeVariants": {"catppuccin": {"dark": {"flavor": "macchiato", "accent": "peach"}}}}`
	if err := os.WriteFile(settings, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	args := []string{"dux", "--settings", settings, "--theme", "../../adapter/dux/testdata/catppuccin.json"}
	if err := dispatch(args, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Name string
		UI   struct{ Primary string } `json:"ui"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	if got.Name != "catppuccin-macchiato" || got.UI.Primary != "#f5a97f" {
		t.Errorf("name %q, primary %s; want catppuccin-macchiato and peach #f5a97f", got.Name, got.UI.Primary)
	}
}

// duxOutput runs the dux subcommand on a fixture theme, plus any extra arguments.
func duxOutput(t *testing.T, theme string, extra ...string) mordye.Output {
	t.Helper()
	args := append([]string{"dux", "--theme", "../../adapter/dux/testdata/" + theme + ".json"}, extra...)
	var out bytes.Buffer
	if err := dispatch(args, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	var got mordye.Output
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out.String())
	}
	return got
}

// Both palettes fill slots by name, which matching by hue alone would never
// do: Dracula's blue is its purple, and Rosé Pine's green is its pine.
func TestBundledPaletteSetsTheTerminalColors(t *testing.T) {
	for _, c := range []struct{ theme, slot, got, want string }{
		{"dracula", "blue", duxOutput(t, "dracula").Terminal.Normal.Blue, "#bd93f9"},
		{"rose-pine", "green", duxOutput(t, "rose-pine").Terminal.Normal.Green, "#31748f"},
	} {
		if c.got != c.want {
			t.Errorf("%s: %s = %s, want the palette's %s", c.theme, c.slot, c.got, c.want)
		}
	}
}

func TestAccentMovesOnlyTheCursorOverABundledPalette(t *testing.T) {
	mauve := duxOutput(t, "catppuccin")
	settings := t.TempDir() + "/settings.json"
	body := `{"currentThemeName": "custom", "registryThemeVariants": {"catppuccin": {"dark": {"flavor": "mocha", "accent": "peach"}}}}`
	if err := os.WriteFile(settings, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	peach := duxOutput(t, "catppuccin", "--settings", settings)
	if peach.UI.Primary == mauve.UI.Primary {
		t.Fatalf("primary %s under both accents; the settings didn't apply", peach.UI.Primary)
	}
	// The accent drives the cursor, even over the palette's; everything else
	// in the palette stays put.
	for name, tm := range map[string]mordye.Output{"mauve": mauve, "peach": peach} {
		if tm.Terminal.Cursor != tm.UI.Primary || tm.Terminal.CursorText != tm.UI.OnPrimary {
			t.Errorf("%s: cursor %s on %s, want the accent %s on %s",
				name, tm.Terminal.Cursor, tm.Terminal.CursorText, tm.UI.Primary, tm.UI.OnPrimary)
		}
	}
	mauve.Terminal.Cursor, mauve.Terminal.CursorText = "", ""
	peach.Terminal.Cursor, peach.Terminal.CursorText = "", ""
	if peach.Terminal != mauve.Terminal {
		t.Errorf("the accent moved more than the cursor:\n mauve %+v\n peach %+v", mauve.Terminal, peach.Terminal)
	}
}

// Without an accent menu, the palette's own cursor passes through.
func TestPalettesCursorPassesThroughWithoutAnAccent(t *testing.T) {
	if got := duxOutput(t, "dracula").Terminal.Cursor; got != "#f8f8f2" {
		t.Errorf("cursor = %s, want the palette's #f8f8f2", got)
	}
}

func TestDuxNeedsATheme(t *testing.T) {
	err := dispatch([]string{"dux"}, strings.NewReader(""), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--theme") {
		t.Errorf("err = %v, want one asking for --theme", err)
	}
}

func TestVersionFlagPrintsTheBuildVersion(t *testing.T) {
	var out bytes.Buffer
	if err := dispatch([]string{"--version"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	name, version, ok := strings.Cut(strings.TrimSpace(out.String()), " ")
	if !ok || name != "mordye" || version == "" {
		t.Errorf("output %q, want \"mordye <version>\"", out.String())
	}
}

func TestVersionFlagPrefersTheStampedVersion(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = "1.2.3"
	var out bytes.Buffer
	if err := dispatch([]string{"--version"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "mordye 1.2.3" {
		t.Errorf("output %q, want the stamped \"mordye 1.2.3\"", got)
	}
}

func TestUnknownSubcommandIsAnError(t *testing.T) {
	err := dispatch([]string{"noctalia"}, strings.NewReader(""), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "noctalia") {
		t.Errorf("err = %v, want one naming the unknown source", err)
	}
}

func TestHelpPrintsUsageAndSucceeds(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"dux", "-h"}, {"dux", "--help"}} {
		var out bytes.Buffer
		if err := dispatch(args, strings.NewReader(""), &out); err != nil {
			t.Errorf("%v: err = %v, want none", args, err)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("%v: output %q, want usage", args, out.String())
		}
	}
}

// A caller's script needs to tell its own mistake from a broken theme.
func TestAMistakeInTheCommandExitsTwo(t *testing.T) {
	for _, args := range [][]string{{"noctalia"}, {"dux"}, {"dux", "--colour", "x"}} {
		err := dispatch(args, strings.NewReader(""), &bytes.Buffer{})
		if got := exitCode(err); got != 2 {
			t.Errorf("%v: exit %d (err %v), want 2", args, got, err)
		}
	}
}

func TestBadInputExitsOne(t *testing.T) {
	for name, c := range map[string]struct {
		args  []string
		stdin string
	}{
		"invalid input":    {nil, `{"version": 1}`},
		"unreadable theme": {[]string{"dux", "--theme", "testdata/no-such-theme.json"}, ""},
	} {
		err := dispatch(c.args, strings.NewReader(c.stdin), &bytes.Buffer{})
		if got := exitCode(err); got != 1 {
			t.Errorf("%s: exit %d (err %v), want 1", name, got, err)
		}
	}
}

func TestRunRejectsTextAfterTheInput(t *testing.T) {
	in := `{"version": 1, "colors": {"primary": "#a6da95", "on_surface": "#cad3f5"}} garbage`
	var out bytes.Buffer
	err := run(strings.NewReader(in), &out)
	if err == nil || !strings.Contains(err.Error(), "after the input") {
		t.Errorf("err = %v, want one about text after the input", err)
	}
	if out.Len() > 0 {
		t.Errorf("wrote a palette despite the error:\n%s", out.String())
	}
}

func TestDuxRejectsExtraArguments(t *testing.T) {
	args := []string{"dux", "--theme", "../../adapter/dux/testdata/nord.json", "extra"}
	err := dispatch(args, strings.NewReader(""), &bytes.Buffer{})
	if got := exitCode(err); got != 2 || !strings.Contains(err.Error(), `"extra"`) {
		t.Errorf("exit %d (err %v), want 2 naming the extra argument", got, err)
	}
}

func TestRunRejectsUnknownFields(t *testing.T) {
	in := `{"version": 1, "colors": {"primary": "#a6da95", "on_surface": "#cad3f5"}, "colours": {}}`
	err := run(strings.NewReader(in), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "colours") {
		t.Errorf("err = %v, want one naming the unknown field", err)
	}
}
