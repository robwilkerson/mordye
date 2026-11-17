package dux

import (
	"testing"

	"github.com/robwilkerson/mordye"
)

// A mapping to a palette Mordye doesn't bundle would make Complete fail for
// that theme, and a theme engine would quietly keep its old palette.
func TestEveryMappedPaletteIsBundled(t *testing.T) {
	for name, id := range palettes {
		_, ok, err := mordye.BundledPalette(id)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("%s maps to %q, which isn't bundled", name, id)
		}
	}
}

func TestAMappedThemeNamesItsPalette(t *testing.T) {
	for name, want := range map[string]string{"catppuccin": "catppuccin-mocha", "dracula": "dracula", "rose-pine": "rose-pine"} {
		in, err := Parse(nil, theme(t, name))
		if err != nil {
			t.Fatal(err)
		}
		if in.Palette != want {
			t.Errorf("%s: palette %q, want %q", name, in.Palette, want)
		}
	}
}

// Gruvbox Material has no qualifying port yet (#40), so its terminal colors
// stay generated.
func TestAnUnmappedThemeNamesNoPalette(t *testing.T) {
	in, err := Parse(nil, theme(t, "gruvbox-material"))
	if err != nil {
		t.Fatal(err)
	}
	if in.Palette != "" {
		t.Errorf("palette %q, want none", in.Palette)
	}
}
