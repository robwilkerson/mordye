// Package kitty reads the terminal colors from a kitty theme file.
//
// It supplements another source rather than standing alone. Most theme
// sources, Dux among them, carry a handful of colors and no terminal palette,
// so Mordye would have to guess all 16 slots by hue. A theme's terminal port
// already says which color goes in each slot, so reading one makes the
// terminal match it.
//
// Kitty is the format because its theme files are the easiest ports to get
// and to read: most themes ship an official one, and each line is a
// plain key and value. Mordye's output is written for whatever terminal the
// caller themes; the port's format is only where the colors come from.
//
// Like every adapter, it takes file contents, never paths.
package kitty

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/robwilkerson/mordye"
)

// fields maps kitty's names for the non-ANSI terminal colors to the
// mordye.TerminalInput field each one sets.
var fields = map[string]func(*mordye.TerminalInput) *string{
	"foreground":           func(t *mordye.TerminalInput) *string { return &t.Foreground },
	"background":           func(t *mordye.TerminalInput) *string { return &t.Background },
	"cursor":               func(t *mordye.TerminalInput) *string { return &t.Cursor },
	"cursor_text_color":    func(t *mordye.TerminalInput) *string { return &t.CursorText },
	"selection_foreground": func(t *mordye.TerminalInput) *string { return &t.SelectionFg },
	"selection_background": func(t *mordye.TerminalInput) *string { return &t.SelectionBg },
}

// keywords are values kitty accepts in place of a color on the non-ANSI
// fields, meaning "work it out from another one". They carry no color to pass
// through, so Mordye fills those fields itself. Kitty never accepts them on
// colorN.
var keywords = map[string]bool{"none": true, "background": true}

// includes pull in another file, whose colors Parse can't see.
var includes = map[string]bool{"include": true, "globinclude": true, "envinclude": true, "geninclude": true}

// Parse reads the terminal colors from a kitty theme file. color0 through
// color7 are the normal set and color8 through color15 the bright one, in
// ANSI order. Every other setting (tab bar, borders, marks) is ignored. As in
// kitty, a later setting replaces an earlier one.
func Parse(conf []byte) (mordye.TerminalInput, error) {
	t := mordye.TerminalInput{Normal: map[string]string{}, Bright: map[string]string{}}
	lines := bufio.NewScanner(bytes.NewReader(conf))
	for n := 1; lines.Scan(); n++ {
		words := strings.Fields(lines.Text())
		if len(words) == 0 || strings.HasPrefix(words[0], "#") {
			continue
		}
		key, value := words[0], strings.Join(words[1:], " ")
		if includes[key] {
			return mordye.TerminalInput{}, fmt.Errorf("line %d: %s isn't supported; pass the included file instead", n, key)
		}
		set, slot := ansiSlot(&t, key)
		field, isField := fields[key]
		if set == nil && !isField {
			continue
		}
		switch {
		case isField && keywords[value]:
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
	// A file that sets nothing is the wrong file, such as a theme passed in
	// place of its terminal palette. Completing without it would hide that.
	if empty(t) {
		return mordye.TerminalInput{}, fmt.Errorf("no terminal colors found; want a kitty theme file")
	}
	return t, nil
}

func empty(t mordye.TerminalInput) bool {
	for _, f := range fields {
		if *f(&t) != "" {
			return false
		}
	}
	return len(t.Normal) == 0 && len(t.Bright) == 0
}

// ansiSlot returns the set and slot a colorN setting fills, or a nil set for
// any other setting.
func ansiSlot(t *mordye.TerminalInput, key string) (map[string]string, string) {
	digits, ok := strings.CutPrefix(key, "color")
	i, err := strconv.Atoi(digits)
	if !ok || err != nil || i < 0 || i > 15 {
		return nil, ""
	}
	if i >= 8 {
		return t.Bright, mordye.Slots[i-8]
	}
	return t.Normal, mordye.Slots[i]
}

func isHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 32)
	return err == nil
}
