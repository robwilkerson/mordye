package converters

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/robwilkerson/mordye"
)

// xresourcesDirectives change which colors apply. Following them is xrdb's
// job, and reading past them could pick the wrong colors, so Xresources
// rejects them.
var xresourcesDirectives = map[string]bool{"include": true, "if": true, "ifdef": true, "ifndef": true}

// looseBinding matches a run of bindings that holds a *.
var looseBinding = regexp.MustCompile(`[.*]*\*[.*]*`)

// Xresources reads the terminal colors from an X resources file, such as
// Nord's official port. It reads the resources terminals share (foreground,
// background, cursorColor, and color0 through color15) under any class or
// wildcard prefix, so *.color0 and URxvt.color0 both count. A value may name
// a #define, which is how Nord writes its palette. As in xrdb, a later
// setting replaces an earlier one, and other resources are ignored. A color
// set under two prefixes is an error: X would pick the more specific one,
// which file order can't tell.
func Xresources(conf []byte) (mordye.TerminalInput, error) {
	t := newTerminal()
	defines := map[string]string{}
	type setting struct {
		prefix string
		line   int
	}
	seen := map[string]setting{}
	lines := bufio.NewScanner(bytes.NewReader(conf))
	for n := 1; lines.Scan(); n++ {
		line := strings.TrimSpace(lines.Text())
		if line == "" || strings.HasPrefix(line, "!") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "#"); ok {
			words := strings.Fields(rest)
			switch {
			case len(words) == 3 && words[0] == "define":
				defines[words[1]] = words[2]
			case len(words) > 0 && xresourcesDirectives[words[0]]:
				return mordye.TerminalInput{}, fmt.Errorf("line %d: #%s isn't supported", n, words[0])
			}
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		cut := strings.LastIndexAny(name, ".*") + 1
		prefix, key := strings.TrimSpace(name[:cut]), strings.TrimSpace(name[cut:])
		value = strings.TrimSpace(value)
		if v, ok := defines[value]; ok {
			value = v
		}
		set, slot := ansiSlot(&t, key)
		field := xresourcesField(&t, key)
		if set == nil && field == nil {
			continue
		}
		// X reads a run of bindings holding a * as a single *, so *color1 and
		// *.color1 name the same resource.
		prefix = looseBinding.ReplaceAllString(prefix, "*")
		if was, ok := seen[key]; ok && was.prefix != prefix {
			return mordye.TerminalInput{}, fmt.Errorf("line %d: %s%s conflicts with line %d's %s%s; X would pick the more specific one",
				n, prefix, key, was.line, was.prefix, key)
		}
		seen[key] = setting{prefix, n}
		if !isHex(value) {
			return mordye.TerminalInput{}, fmt.Errorf("line %d: %s %q: want #rrggbb", n, key, value)
		}
		if field != nil {
			*field = value
		} else {
			set[slot] = value
		}
	}
	if err := lines.Err(); err != nil {
		return mordye.TerminalInput{}, err
	}
	if isEmpty(t) {
		return mordye.TerminalInput{}, fmt.Errorf("no terminal colors found; want an X resources file")
	}
	return t, nil
}

// xresourcesField returns the field a non-ANSI resource sets, or nil.
func xresourcesField(t *mordye.TerminalInput, key string) *string {
	switch key {
	case "foreground":
		return &t.Foreground
	case "background":
		return &t.Background
	case "cursorColor":
		return &t.Cursor
	}
	return nil
}
