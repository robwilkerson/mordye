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

// duxWithKitty runs the dux subcommand on a fixture theme and its kitty port,
// plus any extra arguments.
func duxWithKitty(t *testing.T, theme, port string, extra ...string) mordye.Output {
	t.Helper()
	args := append([]string{"dux", "--theme", "../../adapter/dux/testdata/" + theme + ".json",
		"--terminal-palette", "../../adapter/kitty/testdata/" + port + ".conf"}, extra...)
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

// Both ports fill slots by name, which matching by hue alone would never do:
// Dracula's blue is its purple, and Rosé Pine's green is its pine.
func TestKittyPortSetsTheTerminalColors(t *testing.T) {
	for _, c := range []struct{ theme, slot, got, want string }{
		{"dracula", "blue", duxWithKitty(t, "dracula", "dracula").Terminal.Normal.Blue, "#bd93f9"},
		{"rose-pine", "green", duxWithKitty(t, "rose-pine", "rose-pine").Terminal.Normal.Green, "#31748f"},
	} {
		if c.got != c.want {
			t.Errorf("%s: %s = %s, want the port's %s", c.theme, c.slot, c.got, c.want)
		}
	}
}

func TestAccentDoesNotMoveTheKittyPortsColors(t *testing.T) {
	mauve := duxWithKitty(t, "catppuccin", "catppuccin-mocha")
	settings := t.TempDir() + "/settings.json"
	body := `{"currentThemeName": "custom", "registryThemeVariants": {"catppuccin": {"dark": {"flavor": "mocha", "accent": "peach"}}}}`
	if err := os.WriteFile(settings, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	peach := duxWithKitty(t, "catppuccin", "catppuccin-mocha", "--settings", settings)
	if peach.UI.Primary == mauve.UI.Primary {
		t.Fatalf("primary %s under both accents; the settings didn't apply", peach.UI.Primary)
	}
	if peach.Terminal != mauve.Terminal {
		t.Errorf("the accent moved the terminal:\n mauve %+v\n peach %+v", mauve.Terminal, peach.Terminal)
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

func TestRunRejectsUnknownFields(t *testing.T) {
	in := `{"version": 1, "colors": {"primary": "#a6da95", "on_surface": "#cad3f5"}, "colours": {}}`
	err := run(strings.NewReader(in), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "colours") {
		t.Errorf("err = %v, want one naming the unknown field", err)
	}
}
