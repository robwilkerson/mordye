package mordye

import (
	"math"
	"testing"
)

func TestParseHexRejectsMalformedInput(t *testing.T) {
	for _, in := range []string{"", "ed8796", "#ed879", "#ed87966", "#gg8796"} {
		if _, err := parseHex(in); err == nil {
			t.Errorf("parseHex(%q) succeeded, want an error", in)
		}
	}
}

// Reference values from Björn Ottosson's OKLab definition: sRGB red is
// L 0.628, C 0.258, H 29.2°.
func TestToLCHMatchesOKLabReference(t *testing.T) {
	c := mustHex(t, "#ff0000").lch()
	if math.Abs(c.L-0.628) > 0.001 || math.Abs(c.C-0.258) > 0.001 || math.Abs(c.H-29.2) > 0.1 {
		t.Errorf("#ff0000 = L %.3f C %.3f H %.1f, want L 0.628 C 0.258 H 29.2", c.L, c.C, c.H)
	}
}

func TestLCHRoundTripsToTheSameHex(t *testing.T) {
	for _, h := range []string{"#ed8796", "#24273a", "#a6da95", "#8aadf4", "#000000", "#ffffff"} {
		if got := mustHex(t, h).lch().hex(); got != h {
			t.Errorf("round trip of %s = %s", h, got)
		}
	}
}

// A requested chroma beyond what sRGB can show at that lightness and hue is
// reduced until the color fits, keeping lightness and hue.
func TestOutOfGamutColorKeepsLightnessAndHue(t *testing.T) {
	got := mustHex(t, lch{L: 0.9, C: 0.4, H: 140}.hex()).lch()
	if math.Abs(got.L-0.9) > 0.01 || hueGap(got.H, 140) > 2 || got.C >= 0.4 {
		t.Errorf("clipped color = L %.3f C %.3f H %.1f, want L 0.9, H 140, lower C", got.L, got.C, got.H)
	}
}

func TestChromaSeparatesTintedFromPlain(t *testing.T) {
	plain, err := Chroma("#eceff4")
	if err != nil {
		t.Fatal(err)
	}
	teal, _ := Chroma("#8fbcbb")
	if plain >= teal {
		t.Errorf("chroma of plain white %.3f is not below teal %.3f", plain, teal)
	}
	if _, err := Chroma("teal"); err == nil {
		t.Error("Chroma(\"teal\") succeeded, want an error")
	}
}

func TestHueGapWrapsAroundTheCircle(t *testing.T) {
	if g := hueGap(350, 10); math.Abs(g-20) > 1e-9 {
		t.Errorf("hueGap(350, 10) = %v, want 20", g)
	}
}

func mustHex(t *testing.T, h string) rgb {
	t.Helper()
	c, err := parseHex(h)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
