package dux

import (
	"bytes"
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
	for name, want := range map[string]string{
		"catppuccin": "catppuccin-mocha", "dracula": "dracula", "rose-pine": "rose-pine",
		"gruvbox-material": "gruvbox-material-dark-medium",
	} {
		in, err := Parse(nil, theme(t, name))
		if err != nil {
			t.Fatal(err)
		}
		if in.Palette != want {
			t.Errorf("%s: palette %q, want %q", name, in.Palette, want)
		}
	}
}

// A theme with no qualifying port keeps generated terminal colors. Every
// fixture now has one, so this renames a real one.
func TestAnUnmappedThemeNamesNoPalette(t *testing.T) {
	b := bytes.Replace(theme(t, "dracula"), []byte(`"id": "dracula"`), []byte(`"id": "unported"`), 1)
	if bytes.Equal(b, theme(t, "dracula")) {
		t.Fatal("the dracula fixture's id moved; update this test")
	}
	in, err := Parse(nil, b)
	if err != nil {
		t.Fatal(err)
	}
	if in.Palette != "" {
		t.Errorf("palette %q, want none", in.Palette)
	}
}
