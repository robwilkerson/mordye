package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/robwilkerson/mordye"
)

// dumpScript prints what Neovim renders, in kitty's format, so the kitty
// converter reads it: Normal's colors, then g:terminal_color_0 through 15. A
// color the colorscheme leaves unset prints empty, which the converter
// rejects.
const dumpScript = `let s:out = []
let s:n = synIDtrans(hlID('Normal'))
call add(s:out, 'background ' . synIDattr(s:n, 'bg#'))
call add(s:out, 'foreground ' . synIDattr(s:n, 'fg#'))
for s:i in range(16)
  call add(s:out, 'color' . s:i . ' ' . get(g:, 'terminal_color_' . s:i, ''))
endfor
call writefile(s:out, $MORDYE_DUMP)
qa!
`

// optionName and optionValue keep options to the plain words a colorscheme's
// settings use, since each becomes a line of Vim script.
var (
	optionName  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	optionValue = regexp.MustCompile(`^[a-z0-9_]+$`)
)

// renderVim loads a Vim colorscheme in headless Neovim, with the background
// set from the palette's mode and each option set as a global variable, and
// returns the colors it renders in kitty's format. Neovim reads what the
// colorscheme sets, so no Vim script is parsed here (see docs/decisions/0002).
func renderVim(s spec) ([]byte, error) {
	if path.Dir(s.src.Path) != "colors" || path.Ext(s.src.Path) != ".vim" {
		return nil, fmt.Errorf("path %q: want colors/<name>.vim", s.src.Path)
	}
	setup := []string{"set termguicolors background=" + s.mode}
	for _, k := range slices.Sorted(maps.Keys(s.src.Options)) {
		v := s.src.Options[k]
		if !optionName.MatchString(k) || !optionValue.MatchString(v) {
			return nil, fmt.Errorf("option %s=%s: want lowercase words, digits, and underscores", k, v)
		}
		setup = append(setup, fmt.Sprintf("let g:%s = '%s'", k, v))
	}
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		return nil, errors.New("the vim format needs Neovim (nvim) on the PATH")
	}

	tmp, err := os.MkdirTemp("", "mordye-vim-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	repo := filepath.Join(tmp, "repo")
	if err := fetchRepo(s.src.Repo, s.src.Commit, repo); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(s.src.Path))); err != nil {
		return nil, fmt.Errorf("%s isn't in %s at %s", s.src.Path, s.src.Repo, s.src.Commit)
	}
	script := filepath.Join(tmp, "dump.vim")
	if err := os.WriteFile(script, []byte(dumpScript), 0o644); err != nil {
		return nil, err
	}
	dump := filepath.Join(tmp, "dump.conf")

	args := []string{"--headless", "--clean", "-u", "NONE", "-i", "NONE", "--cmd", "set rtp^=" + repo}
	for _, line := range setup {
		args = append(args, "-c", line)
	}
	name := strings.TrimSuffix(path.Base(s.src.Path), ".vim")
	args = append(args, "-c", "colorscheme "+name, "-S", script)
	cmd := exec.Command(nvim, args...)
	cmd.Env = append(os.Environ(), "MORDYE_DUMP="+dump)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("nvim: %w: %s", err, out)
	}
	b, err := os.ReadFile(dump)
	if err != nil {
		return nil, fmt.Errorf("nvim wrote no colors; does colorscheme %s load? %w", name, err)
	}
	return b, nil
}

// fetchRepo unpacks a GitHub repo at a commit into dir, since a colorscheme
// loads files beyond its own (autoload/, lua/).
func fetchRepo(repo, commit, dir string) error {
	url := "https://codeload.github.com/" + repo + "/tar.gz/" + commit
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// GitHub puts everything under one top-level directory; drop it.
		_, rel, ok := strings.Cut(h.Name, "/")
		if !ok || rel == "" {
			continue
		}
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("%s: archive entry %q escapes the repo", url, h.Name)
		}
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			f, err := os.Create(dst)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, tr)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		}
	}
}

// Options is only meaningful to the vim format; another format ignoring them
// would record a variant choice that chose nothing.
func checkOptions(src mordye.PaletteSource) error {
	if src.Format != "vim" && len(src.Options) > 0 {
		return fmt.Errorf("options are for the vim format, not %s", src.Format)
	}
	return nil
}
