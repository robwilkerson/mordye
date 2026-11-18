package converters

import (
	"os"
	"strings"
	"testing"
)

func TestXresourcesResolvesNordsDefines(t *testing.T) {
	b, err := os.ReadFile("testdata/xresources/nord")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Xresources(b)
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct{ got, want string }{
		"foreground":   {got.Foreground, "#D8DEE9"},
		"background":   {got.Background, "#2E3440"},
		"cursor":       {got.Cursor, "#D8DEE9"},
		"black":        {got.Normal["black"], "#3B4252"},
		"blue":         {got.Normal["blue"], "#81A1C1"},
		"bright cyan":  {got.Bright["cyan"], "#8FBCBB"},
		"bright white": {got.Bright["white"], "#ECEFF4"},
	} {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
	if len(got.Normal) != 8 || len(got.Bright) != 8 {
		t.Errorf("%d normal and %d bright colors, want 8 of each", len(got.Normal), len(got.Bright))
	}
}

func TestXresourcesReadsAnyResourcePrefix(t *testing.T) {
	got, err := Xresources([]byte("URxvt.color1: #aa0000\n*color9:#ff0000\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Normal["red"] != "#aa0000" || got.Bright["red"] != "#ff0000" {
		t.Errorf("red %q, bright red %q; want #aa0000 and #ff0000", got.Normal["red"], got.Bright["red"])
	}
}

// X picks the more specific prefix whatever the order, so last-wins would
// read the wrong color; the converter refuses rather than guess.
func TestXresourcesRejectsAColorSetUnderTwoPrefixes(t *testing.T) {
	_, err := Xresources([]byte("URxvt.color1: #111111\n*.color1: #222222\n"))
	if err == nil || !strings.Contains(err.Error(), "line 2") || !strings.Contains(err.Error(), "line 1") {
		t.Errorf("err = %v, want one naming lines 1 and 2", err)
	}
}

func TestXresourcesLetsALaterSettingReplaceAnEarlierOne(t *testing.T) {
	got, err := Xresources([]byte("*.color1: #111111\n*.color1: #222222\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Normal["red"] != "#222222" {
		t.Errorf("red %q, want the later #222222", got.Normal["red"])
	}
}

func TestXresourcesRejectsConditionals(t *testing.T) {
	_, err := Xresources([]byte("#ifdef DARK\n*.color0: #000000\n#endif\n"))
	if err == nil || !strings.Contains(err.Error(), "ifdef") {
		t.Errorf("err = %v, want one naming #ifdef", err)
	}
}

func TestXresourcesRejectsAColorItCannotRead(t *testing.T) {
	_, err := Xresources([]byte("*.color1: rgb:aa/00/00\n"))
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Errorf("err = %v, want one naming line 1", err)
	}
}

func TestXresourcesRejectsAFileWithNoTerminalColors(t *testing.T) {
	_, err := Xresources([]byte("*fading: 35\n"))
	if err == nil || !strings.Contains(err.Error(), "no terminal colors") {
		t.Errorf("err = %v, want one saying it has no terminal colors", err)
	}
}
