package converters

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/robwilkerson/mordye"
)

// kittyFields maps kitty's names for the non-ANSI terminal colors to the
// mordye.TerminalInput field each one sets.
var kittyFields = map[string]func(*mordye.TerminalInput) *string{
	"foreground":           func(t *mordye.TerminalInput) *string { return &t.Foreground },
	"background":           func(t *mordye.TerminalInput) *string { return &t.Background },
	"cursor":               func(t *mordye.TerminalInput) *string { return &t.Cursor },
	"cursor_text_color":    func(t *mordye.TerminalInput) *string { return &t.CursorText },
	"selection_foreground": func(t *mordye.TerminalInput) *string { return &t.SelectionFg },
	"selection_background": func(t *mordye.TerminalInput) *string { return &t.SelectionBg },
}

// kittyKeywords gives the one word kitty accepts in place of a color on each
// field that takes one, meaning "work it out from another color". It carries
// no color to pass through, so Mordye fills the field itself. The other
// fields, colorN included, take only a color.
var kittyKeywords = map[string]string{
	"cursor":               "none",
	"cursor_text_color":    "background",
	"selection_foreground": "none",
	"selection_background": "none",
}

// kittyIncludes pull in another file, whose colors Kitty can't see.
var kittyIncludes = map[string]bool{"include": true, "globinclude": true, "envinclude": true, "geninclude": true}

// Kitty reads the terminal colors from a kitty theme file. Many themes ship
// an official kitty port, and each line is a plain key and value. Every
// setting other than a color (tab bar, borders, marks) is ignored. As in
// kitty, a later setting replaces an earlier one.
func Kitty(conf []byte) (mordye.TerminalInput, error) {
	t := newTerminal()
	lines := bufio.NewScanner(bytes.NewReader(conf))
	for n := 1; lines.Scan(); n++ {
		words := strings.Fields(lines.Text())
		if len(words) == 0 || strings.HasPrefix(words[0], "#") {
			continue
		}
		key, value := words[0], strings.Join(words[1:], " ")
		if kittyIncludes[key] {
			return mordye.TerminalInput{}, fmt.Errorf("line %d: %s isn't supported; convert the included file instead", n, key)
		}
		set, slot := ansiSlot(&t, key)
		field, isField := kittyFields[key]
		if set == nil && !isField {
			continue
		}
		switch {
		case kittyKeywords[key] != "" && strings.EqualFold(value, kittyKeywords[key]):
			value = ""
		case !isHex(value):
			return mordye.TerminalInput{}, fmt.Errorf("line %d: %s %q: want #rrggbb", n, key, value)
		}
		if isField {
			*field(&t) = value
		} else {
			set[slot] = value
		}
	}
	if err := lines.Err(); err != nil {
		return mordye.TerminalInput{}, err
	}
	if isEmpty(t) {
		return mordye.TerminalInput{}, fmt.Errorf("no terminal colors found; want a kitty theme file")
	}
	return t, nil
}
