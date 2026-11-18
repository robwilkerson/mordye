// Package converters turns a theme's terminal port into terminal colors, one
// function per port format.
//
// Converters run only ahead of time: the palette tool (tools/palettes) uses
// them to build Mordye's bundled palettes from each theme's official port
// (see docs/decisions/0002). Mordye never reads a port file at runtime. The
// port's format is only where the colors come from; Mordye's output is
// written for whatever terminal the caller themes.
//
// Like adapters, converters take file contents, never paths.
package converters

import (
	"strconv"
	"strings"

	"github.com/robwilkerson/mordye"
)

// newTerminal returns an empty TerminalInput ready to fill.
func newTerminal() mordye.TerminalInput {
	return mordye.TerminalInput{Normal: map[string]string{}, Bright: map[string]string{}}
}

// ansiSlot returns the set and slot that ANSI color n fills, for a key such
// as "color9", or a nil set for any other key. color0 through color7 are the
// normal set and color8 through color15 the bright one, in ANSI order.
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

// isEmpty reports whether t sets no color at all. A port that sets none is
// the wrong file, and converting it anyway would hide that.
func isEmpty(t mordye.TerminalInput) bool {
	scalars := []string{t.Background, t.Foreground, t.Cursor, t.CursorText, t.SelectionBg, t.SelectionFg}
	for _, hex := range scalars {
		if hex != "" {
			return false
		}
	}
	return len(t.Normal) == 0 && len(t.Bright) == 0
}

func isHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 32)
	return err == nil
}
