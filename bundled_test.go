package mordye

import (
	"os"
	"strings"
	"testing"
)

func minimalInput() Input {
	return Input{Version: Version, Colors: map[string]string{"primary": "#a6da95", "on_surface": "#cad3f5"}}
}

// A bundled palette that can't load or validate would break every theme
// mapped to it, so each one must complete a palette.
func TestEveryBundledPaletteCompletesAPalette(t *testing.T) {
	ids, err := BundledPaletteIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) == 0 {
		t.Fatal("no bundled palettes")
	}
	for _, id := range ids {
		in := minimalInput()
		in.Palette = id
		if _, err := Complete(in); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

// The release ships each palette's license next to the binary, so a notice
// that's missing would ship a palette without its license.
func TestEveryBundledPaletteHasItsNotice(t *testing.T) {
	ids, _ := BundledPaletteIDs()
	for _, id := range ids {
		p, _, _ := BundledPalette(id)
		if _, err := os.Stat("palettes/" + p.License.Notice); err != nil {
			t.Errorf("%s: notice %s: %v", id, p.License.Notice, err)
		}
	}
}

func TestBundledPaletteColorsAreSourcedToThePalette(t *testing.T) {
	in := minimalInput()
	in.Palette = "dracula"
	out, err := Complete(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Terminal.Normal.Blue != "#bd93f9" {
		t.Errorf("blue = %s, want Dracula's purple #bd93f9", out.Terminal.Normal.Blue)
	}
	for _, field := range []string{"terminal.normal.blue", "terminal.bright.red", "terminal.background"} {
		if got := out.Sources[field]; got != "palette.dracula" {
			t.Errorf("sources[%s] = %q, want palette.dracula", field, got)
		}
	}
}

// Two ports of one theme can disagree, so mixing them would match neither.
func TestInputTerminalColorsWinWholeOverABundledPalette(t *testing.T) {
	in := minimalInput()
	in.Palette = "dracula"
	in.Terminal = TerminalInput{Normal: map[string]string{"red": "#ff0000"}}
	out, err := Complete(in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Terminal.Normal.Red != "#ff0000" || out.Sources["terminal.normal.red"] != "terminal" {
		t.Errorf("red %s from %q, want the input's #ff0000 from terminal", out.Terminal.Normal.Red, out.Sources["terminal.normal.red"])
	}
	if out.Terminal.Normal.Blue == "#bd93f9" || strings.HasPrefix(out.Sources["terminal.normal.blue"], "palette.") {
		t.Errorf("blue %s from %q; the palette mixed in with the input's colors", out.Terminal.Normal.Blue, out.Sources["terminal.normal.blue"])
	}
}

func TestAnUnknownPaletteIsAnError(t *testing.T) {
	in := minimalInput()
	in.Palette = "dracula-pro"
	_, err := Complete(in)
	if err == nil || !strings.Contains(err.Error(), "dracula-pro") {
		t.Errorf("err = %v, want one naming the unknown palette", err)
	}
}

func TestParsePaletteRejectsAnUnknownField(t *testing.T) {
	_, err := ParsePalette([]byte(`{"id": "x", "mode": "dark", "colour": {}}`))
	if err == nil || !strings.Contains(err.Error(), "colour") {
		t.Errorf("err = %v, want one naming the unknown field", err)
	}
}
