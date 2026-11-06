package mordye

import (
	"encoding/json"
	"strings"
	"testing"
)

// macchiato is Catppuccin Macchiato as the DMS adapter writes it: the green
// accent as primary, and the whole accent menu, green included, as swatches.
func macchiato() Input {
	return Input{
		Version: 1,
		Name:    "catppuccin-macchiato",
		Accent:  "green",
		Colors: map[string]string{
			"primary": "#a6da95", "on_primary": "#24273a", "secondary": "#8bd5ca",
			"error": "#ed8796", "warning": "#f5a97f", "info": "#8aadf4",
			"background": "#24273a", "surface": "#1e2030", "surface_container": "#24273a",
			"surface_container_high": "#363a4f", "surface_container_highest": "#494d64",
			"outline": "#6e738d", "on_surface": "#cad3f5", "on_surface_variant": "#a5adcb",
		},
		Swatches: map[string]string{
			"rosewater": "#f4dbd6", "flamingo": "#f0c6c6", "pink": "#f5bde6", "mauve": "#c6a0f6",
			"red": "#ed8796", "maroon": "#ee99a0", "peach": "#f5a97f", "yellow": "#eed49f",
			"green": "#a6da95", "teal": "#8bd5ca", "sky": "#91d7e3", "sapphire": "#7dc4e4", "blue": "#8aadf4",
			"lavender": "#b7bdf8",
		},
	}
}

// sparse is a theme that defines only a few colors, like Gruvbox Material's
// DMS theme: no surfaces, no green or blue.
func sparse() Input {
	return Input{
		Version: 1,
		Colors: map[string]string{
			"primary": "#d7a657", "error": "#e96962",
			"outline": "#a89984", "on_surface": "#ddc7a1",
		},
	}
}

func complete(t *testing.T, in Input) Output {
	t.Helper()
	out, err := Complete(in)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestRequiresPrimaryAndOnSurface(t *testing.T) {
	for _, role := range []string{"primary", "on_surface"} {
		in := macchiato()
		delete(in.Colors, role)
		if _, err := Complete(in); err == nil || !strings.Contains(err.Error(), role) {
			t.Errorf("without %s: err = %v, want one naming %s", role, err, role)
		}
	}
}

func TestRejectsUnknownRoles(t *testing.T) {
	in := macchiato()
	in.Colors["surfaceText"] = "#cad3f5"
	if _, err := Complete(in); err == nil || !strings.Contains(err.Error(), "surfaceText") {
		t.Errorf("err = %v, want one naming the unknown role", err)
	}
}

func TestRejectsMalformedColors(t *testing.T) {
	in := macchiato()
	in.Swatches["teal"] = "teal"
	if _, err := Complete(in); err == nil || !strings.Contains(err.Error(), "teal") {
		t.Errorf("err = %v, want one naming the bad swatch", err)
	}
}

func TestRejectsMalformedTerminalFields(t *testing.T) {
	for field, set := range map[string]func(*TerminalInput){
		"background":   func(ti *TerminalInput) { ti.Background = "nope" },
		"foreground":   func(ti *TerminalInput) { ti.Foreground = "nope" },
		"cursor":       func(ti *TerminalInput) { ti.Cursor = "nope" },
		"cursor_text":  func(ti *TerminalInput) { ti.CursorText = "nope" },
		"selection_bg": func(ti *TerminalInput) { ti.SelectionBg = "nope" },
		"selection_fg": func(ti *TerminalInput) { ti.SelectionFg = "nope" },
	} {
		in := macchiato()
		set(&in.Terminal)
		if _, err := Complete(in); err == nil || !strings.Contains(err.Error(), "terminal."+field+":") {
			t.Errorf("bad %s: err = %v, want one naming terminal.%s", field, err, field)
		}
	}
}

func TestThemeColorsFillTheirSlots(t *testing.T) {
	out := complete(t, macchiato())
	want := map[string]string{
		"red": "#ed8796", "green": "#a6da95", "yellow": "#eed49f", "blue": "#8aadf4",
	}
	for slot, hex := range want {
		if got := out.Terminal.Normal.get(slot); got != hex {
			t.Errorf("normal %s = %s, want %s", slot, got, hex)
		}
	}
}

func TestNamedRolesWinOverSwatches(t *testing.T) {
	out := complete(t, macchiato())
	if got := out.Sources["terminal.normal.red"]; got != "colors.error" {
		t.Errorf("red came from %s, want colors.error over the red, maroon and flamingo swatches", got)
	}
}

func TestAccentDoesNotMoveTerminalColors(t *testing.T) {
	green := complete(t, macchiato())
	mauve := macchiato()
	mauve.Colors["primary"], mauve.Colors["secondary"] = "#c6a0f6", "#b7bdf8"
	if got := complete(t, mauve).Terminal; got.Normal != green.Terminal.Normal || got.Bright != green.Terminal.Bright {
		t.Errorf("changing the accent changed the 16 terminal colors:\n green %+v\n mauve %+v", green.Terminal.Normal, got.Normal)
	}
}

func TestAccentMustNameASwatch(t *testing.T) {
	in := macchiato()
	in.Accent = "chartreuse"
	if _, err := Complete(in); err == nil || !strings.Contains(err.Error(), "chartreuse") {
		t.Errorf("err = %v, want one naming the missing accent swatch", err)
	}
}

// Without an accent choice, primary and secondary are the theme's own colors
// and fill slots like any other, even when the input has swatches (here,
// Tokyo Night's teal text color).
func TestThemeWithoutAccentChoiceUsesPrimaryForSlots(t *testing.T) {
	in := Input{
		Version: 1,
		Colors: map[string]string{
			"primary": "#7aa2f7", "secondary": "#bb9af7", "error": "#f7768e", "on_surface": "#d5d6db",
		},
		Swatches: map[string]string{"surfaceText": "#73daca"},
	}
	out := complete(t, in)
	if out.Terminal.Normal.Blue != "#7aa2f7" || out.Terminal.Normal.Magenta != "#bb9af7" {
		t.Errorf("blue %s, magenta %s; want primary #7aa2f7 and secondary #bb9af7",
			out.Terminal.Normal.Blue, out.Terminal.Normal.Magenta)
	}
}

func TestSwatchesNeverBecomeInterfaceRoles(t *testing.T) {
	in := macchiato()
	delete(in.Colors, "secondary")
	if got := complete(t, in).UI.Secondary; got != in.Colors["primary"] {
		t.Errorf("secondary = %s, want it to fall back to primary, not a swatch", got)
	}
}

func TestMissingSlotIsGeneratedAtItsHue(t *testing.T) {
	out := complete(t, sparse())
	if src := out.Sources["terminal.normal.green"]; src != "generated" {
		t.Fatalf("green came from %s, want generated", src)
	}
	if h := mustHex(t, out.Terminal.Normal.Green).lch().H; hueGap(h, slotHues["green"]) > 5 {
		t.Errorf("generated green has hue %.0f°, want near %.0f°", h, slotHues["green"])
	}
}

// A generated color takes the palette's own chroma, so a muted theme gets a
// muted green rather than a vivid one.
func TestGeneratedColorMatchesPaletteIntensity(t *testing.T) {
	out := complete(t, sparse())
	green := mustHex(t, out.Terminal.Normal.Green).lch()
	red := mustHex(t, out.Terminal.Normal.Red).lch()
	if green.C > red.C*1.5 {
		t.Errorf("generated green chroma %.3f is far above the palette's red %.3f", green.C, red.C)
	}
}

func TestStatusRoleClaimsItsConventionalSlot(t *testing.T) {
	in := sparse()
	in.Colors["primary"] = "#bd93f9" // Dracula's purple, so nothing else competes for yellow
	in.Colors["warning"] = "#f1fa8c" // Dracula's yellow, on the border of green
	if got := complete(t, in).Terminal.Normal.Yellow; got != "#f1fa8c" {
		t.Errorf("yellow = %s, want warning's #f1fa8c", got)
	}
}

func TestStructuralRolesAreNotSlotCandidates(t *testing.T) {
	in := sparse()
	in.Colors["outline"] = "#6272a4" // Dracula's blue-gray outline
	if src := complete(t, in).Sources["terminal.normal.blue"]; src == "colors.outline" {
		t.Errorf("blue came from the outline color")
	}
}

func TestTerminalColorsFromTheSourcePassThrough(t *testing.T) {
	in := macchiato()
	in.Terminal = TerminalInput{Normal: map[string]string{"magenta": "#f5bde6"}, Cursor: "#f4dbd6"}
	out := complete(t, in)
	if out.Terminal.Normal.Magenta != "#f5bde6" || out.Terminal.Cursor != "#f4dbd6" {
		t.Errorf("magenta %s, cursor %s; want the source's #f5bde6 and #f4dbd6", out.Terminal.Normal.Magenta, out.Terminal.Cursor)
	}
	if got := out.Sources["terminal.normal.magenta"]; got != "terminal" {
		t.Errorf("magenta source = %s, want terminal", got)
	}
}

func TestMissingNeutralsAreGenerated(t *testing.T) {
	out := complete(t, sparse())
	for name, hex := range map[string]string{
		"background": out.UI.Background, "surface": out.UI.Surface,
		"surface_container_high": out.UI.SurfaceContainerHigh, "black": out.Terminal.Normal.Black,
	} {
		if c := mustHex(t, hex).lch(); c.L > 0.5 || c.C > 0.05 {
			t.Errorf("%s = %s (L %.2f C %.3f), want a dark, nearly neutral gray", name, hex, c.L, c.C)
		}
	}
}

// Tokyo Night defines surface_container_high but not _highest. Black must use
// the theme's own color, not a generated one.
func TestBlackPrefersTheThemesOwnSurface(t *testing.T) {
	in := macchiato()
	delete(in.Colors, "surface_container_highest")
	if got := complete(t, in).Terminal.Normal.Black; got != "#363a4f" {
		t.Errorf("black = %s, want the theme's surface_container_high #363a4f", got)
	}
}

// A generated gray takes only a hint of the theme's warmth or coolness, so
// Gruvbox gets a near-neutral background rather than a brown one.
func TestGeneratedNeutralsAreNearlyGray(t *testing.T) {
	out := complete(t, sparse())
	if c := mustHex(t, out.UI.Background).lch().C; c > 0.01 {
		t.Errorf("background %s has chroma %.4f, want under 0.01", out.UI.Background, c)
	}
}

// A generated color sits slightly behind the theme's own, so a color the
// theme never chose doesn't outshine the ones it did.
func TestGeneratedColorsSitBehindTheThemesOwn(t *testing.T) {
	out := complete(t, sparse())
	green := mustHex(t, out.Terminal.Normal.Green).lch()
	red := mustHex(t, out.Terminal.Normal.Red).lch()
	if green.C >= red.C {
		t.Errorf("generated green chroma %.3f is not below the theme's red %.3f", green.C, red.C)
	}
}

func TestBrightBlackAndWhiteStepAwayFromNormal(t *testing.T) {
	term := complete(t, macchiato()).Terminal
	n, b := term.Normal, term.Bright
	if mustHex(t, b.Black).lch().L <= mustHex(t, n.Black).lch().L {
		t.Errorf("bright black %s is not lighter than black %s", b.Black, n.Black)
	}
	if mustHex(t, b.White).lch().L <= mustHex(t, n.White).lch().L {
		t.Errorf("bright white %s is not lighter than white %s", b.White, n.White)
	}
}

func TestOutputAlwaysHasEveryField(t *testing.T) {
	for name, in := range map[string]Input{"full": macchiato(), "sparse": sparse()} {
		raw, err := json.Marshal(complete(t, in))
		if err != nil {
			t.Fatal(err)
		}
		var out struct {
			UI       map[string]string          `json:"ui"`
			Terminal map[string]json.RawMessage `json:"terminal"`
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		colors := map[string]string{}
		for k, v := range out.UI {
			colors["ui."+k] = v
		}
		for k, v := range out.Terminal {
			if k == "normal" || k == "bright" {
				var ansi map[string]string
				if err := json.Unmarshal(v, &ansi); err != nil {
					t.Fatal(err)
				}
				for slot, hex := range ansi {
					colors["terminal."+k+"."+slot] = hex
				}
				continue
			}
			var hex string
			if err := json.Unmarshal(v, &hex); err != nil {
				t.Fatal(err)
			}
			colors["terminal."+k] = hex
		}
		if len(colors) != 37 {
			t.Errorf("%s: %d fields, want 37", name, len(colors))
		}
		for field, hex := range colors {
			if _, err := parseHex(hex); err != nil {
				t.Errorf("%s: %s = %q, want a #rrggbb color", name, field, hex)
			}
		}
	}
}
