package mordye

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"
)

// Palette is a theme's official terminal palette, bundled with Mordye. The
// palette tool (tools/palettes) writes each one from the upstream port; nobody
// edits them by hand (see docs/decisions/0002).
type Palette struct {
	// ID names the palette, after the upstream port's own file name, such as
	// "catppuccin-mocha". Adapters map their theme names to it.
	ID string `json:"id"`
	// Mode is "dark" or "light".
	Mode     string        `json:"mode"`
	Source   PaletteSource `json:"source"`
	License  License       `json:"license"`
	Terminal TerminalInput `json:"terminal"`
}

// PaletteSource records where a palette's colors came from, so the palette
// tool can rebuild it and prove the bundled copy is unchanged.
type PaletteSource struct {
	Repo   string `json:"repo"`   // GitHub owner/name
	Commit string `json:"commit"` // the full commit SHA the file was read at
	Path   string `json:"path"`   // the port file, within the repo
	Format string `json:"format"` // the converter that read it, such as "kitty"
	// Options pick a variant of a Vim colorscheme, set as global variables
	// before it loads, such as everforest_background=hard. Only the vim
	// format takes them.
	Options map[string]string `json:"options,omitempty"`
}

// License records the upstream license and where it's stated, at the same
// commit. Notice is the license text, under palettes/notices/, which the
// release ships alongside the binary.
type License struct {
	SPDX   string `json:"spdx"`
	URL    string `json:"url"`
	Notice string `json:"notice"`
}

// The whole directory is embedded, rather than palettes/*.json, so the
// package still builds before any palette exists; the palette tool that
// writes the first one imports it.
//
//go:embed palettes
var paletteFiles embed.FS

// bundledPalettes reads every embedded palette once, keyed by id.
var bundledPalettes = sync.OnceValues(func() (map[string]Palette, error) {
	files, err := paletteFiles.ReadDir("palettes")
	if err != nil {
		return nil, err
	}
	out := map[string]Palette{}
	for _, f := range files {
		if f.IsDir() || path.Ext(f.Name()) != ".json" {
			continue
		}
		b, err := paletteFiles.ReadFile(path.Join("palettes", f.Name()))
		if err != nil {
			return nil, err
		}
		p, err := ParsePalette(b)
		if err != nil {
			return nil, fmt.Errorf("bundled palette %s: %w", f.Name(), err)
		}
		if want := strings.TrimSuffix(f.Name(), ".json"); p.ID != want {
			return nil, fmt.Errorf("bundled palette %s: id %q, want %q", f.Name(), p.ID, want)
		}
		out[p.ID] = p
	}
	return out, nil
})

// ParsePalette reads one palette file. Unknown fields are an error, so a
// misspelled field can't drop a color silently.
func ParsePalette(b []byte) (Palette, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var p Palette
	if err := dec.Decode(&p); err != nil {
		return Palette{}, err
	}
	if !safeName(p.ID) {
		return Palette{}, fmt.Errorf("id %q: %s", p.ID, nameRule)
	}
	if p.Mode != "dark" && p.Mode != "light" {
		return Palette{}, fmt.Errorf("mode %q: want dark or light", p.Mode)
	}
	return p, nil
}

// BundledPalette returns the bundled palette with this id. Its color maps are
// copies, so an edit to them can't reach the cache every lookup shares.
func BundledPalette(id string) (Palette, bool, error) {
	all, err := bundledPalettes()
	if err != nil {
		return Palette{}, false, err
	}
	p, ok := all[id]
	p.Terminal.Normal = maps.Clone(p.Terminal.Normal)
	p.Terminal.Bright = maps.Clone(p.Terminal.Bright)
	return p, ok, nil
}

// BundledPaletteIDs lists every bundled palette's id, sorted.
func BundledPaletteIDs() ([]string, error) {
	all, err := bundledPalettes()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids, nil
}
