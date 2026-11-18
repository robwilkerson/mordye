// Command palettes builds Mordye's bundled terminal palettes from official
// upstream ports, and proves the bundled copies are unchanged.
//
//	go run ./tools/palettes add -id catppuccin-mocha -mode dark \
//	    -repo catppuccin/kitty -commit SHA -path themes/mocha.conf \
//	    -format kitty -spdx MIT -license-path LICENSE
//	go run ./tools/palettes check
//
// add fetches the port and its license at one commit, converts the port, and
// writes palettes/<id>.json plus the license text under palettes/notices/.
// check rebuilds every bundled palette from its recorded source and fails if
// any file differs. Run both from the repository root.
//
// It's a maintainer tool: it reads the network, and it never ships in the
// binary (see docs/decisions/0002).
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/robwilkerson/mordye"
	"github.com/robwilkerson/mordye/converters"
)

// formatConverters maps a port format to the converter that reads it.
var formatConverters = map[string]func([]byte) (mordye.TerminalInput, error){
	"kitty":      converters.Kitty,
	"xresources": converters.Xresources,
}

// dir is where the bundled palettes live, relative to the repository root.
const dir = "palettes"

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "palettes:", err)
		os.Exit(1)
	}
}

func dispatch(args []string) error {
	if len(args) == 0 {
		return errors.New("want a command: add or check")
	}
	switch args[0] {
	case "add":
		return add(args[1:])
	case "check":
		return check()
	default:
		return fmt.Errorf("unknown command %q; commands: add, check", args[0])
	}
}

// spec is everything needed to build one palette. add takes it from flags,
// and check from a palette file already bundled.
type spec struct {
	id, mode                   string
	src                        mordye.PaletteSource
	spdx, licensePath, licLine string
}

func add(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	var s spec
	fs.StringVar(&s.id, "id", "", "the palette id, after the upstream port's file name (required)")
	fs.StringVar(&s.mode, "mode", "", "dark or light (required)")
	fs.StringVar(&s.src.Repo, "repo", "", "the upstream GitHub repo, owner/name (required)")
	fs.StringVar(&s.src.Commit, "commit", "", "the full commit SHA to read at (required)")
	fs.StringVar(&s.src.Path, "path", "", "the port file within the repo (required)")
	fs.StringVar(&s.src.Format, "format", "", "the port's format: "+strings.Join(formats(), ", ")+" (required)")
	fs.StringVar(&s.spdx, "spdx", "", "the upstream license's SPDX id, such as MIT (required)")
	fs.StringVar(&s.licensePath, "license-path", "", "the file that states the license, within the repo (required)")
	fs.StringVar(&s.licLine, "license-line", "", "the line that states it, when that file is a README rather than a LICENSE")
	if err := fs.Parse(args); err != nil {
		return err
	}
	for name, v := range map[string]string{
		"id": s.id, "mode": s.mode, "repo": s.src.Repo, "commit": s.src.Commit,
		"path": s.src.Path, "format": s.src.Format, "spdx": s.spdx, "license-path": s.licensePath,
	} {
		if v == "" {
			return fmt.Errorf("add: -%s is required", name)
		}
	}
	files, err := build(s)
	if err != nil {
		return err
	}
	for path, b := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}
	return nil
}

// check rebuilds every bundled palette and its notice, and reports any that
// differ from what's bundled.
func check() error {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no palettes under %s/; run from the repository root", dir)
	}
	var stale []string
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		p, err := mordye.ParsePalette(b)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		// The files compared below are named from the id, so a file under
		// another name would never be compared with its source.
		if want := filepath.Join(dir, p.ID+".json"); path != want {
			return fmt.Errorf("%s holds palette %q; it belongs at %s", path, p.ID, want)
		}
		s, err := specOf(p)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		files, err := build(s)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for f, want := range files {
			if got, err := os.ReadFile(f); err != nil || !bytes.Equal(got, want) {
				stale = append(stale, f)
			}
		}
	}
	if len(stale) > 0 {
		slices.Sort(stale)
		return fmt.Errorf("these differ from their recorded sources; rerun add instead of editing them:\n  %s",
			strings.Join(slices.Compact(stale), "\n  "))
	}
	fmt.Printf("%d palettes match their sources\n", len(paths))
	return nil
}

// specOf recovers the spec a bundled palette was built from.
func specOf(p mordye.Palette) (spec, error) {
	prefix := "https://github.com/" + p.Source.Repo + "/blob/" + p.Source.Commit + "/"
	rest, ok := strings.CutPrefix(p.License.URL, prefix)
	if !ok {
		return spec{}, fmt.Errorf("license url %q isn't in %s at %s", p.License.URL, p.Source.Repo, p.Source.Commit)
	}
	licPath, line, _ := strings.Cut(rest, "#L")
	return spec{id: p.ID, mode: p.Mode, src: p.Source, spdx: p.License.SPDX, licensePath: licPath, licLine: line}, nil
}

var sha = regexp.MustCompile(`^[0-9a-f]{40}$`)

// build fetches a palette's port and license, and returns the files it
// becomes, keyed by path.
func build(s spec) (map[string][]byte, error) {
	if !sha.MatchString(s.src.Commit) {
		return nil, fmt.Errorf("commit %q: want a full 40-character SHA, so the source can't move", s.src.Commit)
	}
	convert, ok := formatConverters[s.src.Format]
	if !ok {
		return nil, fmt.Errorf("format %q has no converter; formats: %v", s.src.Format, formats())
	}
	port, err := fetch(s.src.Repo, s.src.Commit, s.src.Path)
	if err != nil {
		return nil, err
	}
	terminal, err := convert(port)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.src.Path, err)
	}
	license, err := fetch(s.src.Repo, s.src.Commit, s.licensePath)
	if err != nil {
		return nil, err
	}
	url := "https://github.com/" + s.src.Repo + "/blob/" + s.src.Commit + "/" + s.licensePath
	if s.licLine != "" {
		url += "#L" + s.licLine
	}
	notice := "notices/" + strings.ReplaceAll(s.src.Repo, "/", "-") + ".txt"
	p := mordye.Palette{
		ID: s.id, Mode: s.mode, Source: s.src,
		License:  mordye.License{SPDX: s.spdx, URL: url, Notice: notice},
		Terminal: terminal,
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	// Read it back as Mordye will, so a palette that can't load is never written.
	if _, err := mordye.ParsePalette(b); err != nil {
		return nil, err
	}
	return map[string][]byte{
		filepath.Join(dir, s.id+".json"): b,
		filepath.Join(dir, notice):       license,
	}, nil
}

var client = &http.Client{Timeout: 30 * time.Second}

// fetch reads one file from a GitHub repo at a commit.
func fetch(repo, commit, path string) ([]byte, error) {
	url := "https://raw.githubusercontent.com/" + repo + "/" + commit + "/" + path
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func formats() []string {
	var out []string
	for f := range formatConverters {
		out = append(out, f)
	}
	slices.Sort(out)
	return out
}
