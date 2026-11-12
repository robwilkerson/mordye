package kitty

import (
	"os"
	"strings"
	"testing"
)

func port(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".conf")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReadsAllSixteenColors(t *testing.T) {
	got, err := Parse(port(t, "dracula"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Normal) != 8 || len(got.Bright) != 8 {
		t.Fatalf("normal %v, bright %v; want all eight of each", got.Normal, got.Bright)
	}
	for set, want := range map[string]map[string]string{
		"normal": {"black": "#21222c", "blue": "#bd93f9", "white": "#f8f8f2"},
		"bright": {"black": "#6272a4", "blue": "#d6acff", "white": "#ffffff"},
	} {
		colors := map[string]map[string]string{"normal": got.Normal, "bright": got.Bright}[set]
		for slot, hex := range want {
			if colors[slot] != hex {
				t.Errorf("%s.%s = %q, want %s", set, slot, colors[slot], hex)
			}
		}
	}
}

func TestReadsTheTerminalFields(t *testing.T) {
	got, err := Parse(port(t, "catppuccin-mocha"))
	if err != nil {
		t.Fatal(err)
	}
	for field, pair := range map[string][2]string{
		"foreground":   {got.Foreground, "#cdd6f4"},
		"background":   {got.Background, "#1e1e2e"},
		"cursor":       {got.Cursor, "#f5e0dc"},
		"cursor_text":  {got.CursorText, "#1e1e2e"},
		"selection_fg": {got.SelectionFg, "#1e1e2e"},
		"selection_bg": {got.SelectionBg, "#f5e0dc"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %q, want %s", field, pair[0], pair[1])
		}
	}
}

// Dracula sets cursor_text_color to "background", kitty's way of saying "use
// another color". There's no hex to pass through, so Mordye fills it.
func TestAKeywordInPlaceOfAColorIsLeftForMordye(t *testing.T) {
	got, err := Parse(port(t, "dracula"))
	if err != nil {
		t.Fatal(err)
	}
	if got.CursorText != "" {
		t.Errorf("cursor_text = %q, want it left empty", got.CursorText)
	}
	got, err = Parse([]byte("selection_foreground none\n"))
	if err != nil || got.SelectionFg != "" {
		t.Errorf("selection_fg %q, err %v; want it left empty", got.SelectionFg, err)
	}
}

func TestTheLastSettingWins(t *testing.T) {
	got, err := Parse([]byte("color1 #111111\ncolor1 #222222\n"))
	if err != nil || got.Normal["red"] != "#222222" {
		t.Errorf("red %q, err %v; want the later #222222, as kitty reads it", got.Normal["red"], err)
	}
}

func TestRejectsAColorMordyeCannotRead(t *testing.T) {
	for _, conf := range []string{"color1 red\n", "color1 #f00\n", "foreground\n"} {
		_, err := Parse([]byte(conf))
		if err == nil || !strings.Contains(err.Error(), "line 1") {
			t.Errorf("Parse(%q): err = %v, want one naming line 1", conf, err)
		}
	}
}

// An included file could hold colors Mordye never sees, so it says so rather
// than completing the palette without them.
func TestRejectsAnInclude(t *testing.T) {
	_, err := Parse([]byte("# colors\ninclude dracula.conf\n"))
	if err == nil || !strings.Contains(err.Error(), "include") {
		t.Errorf("err = %v, want one naming the include", err)
	}
}
