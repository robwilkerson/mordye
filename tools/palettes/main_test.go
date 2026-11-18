package main

import (
	"os"
	"strings"
	"testing"

	"github.com/robwilkerson/mordye"
)

// check rebuilds a palette from what its file records, so reading the spec
// back must give exactly what add was given.
func TestSpecOfRecoversWhatABundledPaletteWasBuiltFrom(t *testing.T) {
	b, err := os.ReadFile("../../palettes/catppuccin-mocha.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := mordye.ParsePalette(b)
	if err != nil {
		t.Fatal(err)
	}
	got, err := specOf(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.id != "catppuccin-mocha" || got.mode != "dark" || got.src != p.Source ||
		got.spdx != "MIT" || got.licensePath != "LICENSE" || got.licLine != "" {
		t.Errorf("spec %+v, want the id, mode, source, and LICENSE the file records", got)
	}
}

// A project that states its license only in its README is credited by line,
// and check has to find that line again.
func TestSpecOfRecoversALicenseStatedOnAReadmeLine(t *testing.T) {
	src := mordye.PaletteSource{Repo: "morhetz/gruvbox", Commit: strings.Repeat("a", 40), Path: "colors/gruvbox.vim", Format: "vim"}
	p := mordye.Palette{ID: "gruvbox-dark", Mode: "dark", Source: src, License: mordye.License{
		SPDX: "MIT", URL: "https://github.com/morhetz/gruvbox/blob/" + src.Commit + "/README.md#L117",
	}}
	got, err := specOf(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.licensePath != "README.md" || got.licLine != "117" {
		t.Errorf("license %s line %q, want README.md line 117", got.licensePath, got.licLine)
	}
}

// A license read from another repo or commit wouldn't be the one that covers
// these colors.
func TestSpecOfRejectsALicenseFromAnotherCommit(t *testing.T) {
	src := mordye.PaletteSource{Repo: "dracula/kitty", Commit: strings.Repeat("a", 40)}
	p := mordye.Palette{Source: src, License: mordye.License{
		URL: "https://github.com/dracula/kitty/blob/" + strings.Repeat("b", 40) + "/LICENSE",
	}}
	if _, err := specOf(p); err == nil {
		t.Error("err = nil, want one saying the license isn't at the source's commit")
	}
}

// check names the files it compares from each palette's id, so a file under
// another name would never be compared. It fails before any fetch.
func TestCheckRejectsAPaletteFileNamedForAnotherID(t *testing.T) {
	b, err := os.ReadFile("../../palettes/catppuccin-mocha.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/nord.json", b, 0o644); err != nil {
		t.Fatal(err)
	}
	err = check()
	if err == nil || !strings.Contains(err.Error(), "nord.json") || !strings.Contains(err.Error(), "catppuccin-mocha") {
		t.Errorf("err = %v, want one naming nord.json and the id it holds", err)
	}
}

// Both fail before any fetch, so a bad spec never reaches the network.
func TestBuildRejectsAMovableCommitOrAnUnknownFormat(t *testing.T) {
	for _, c := range []struct {
		name string
		src  mordye.PaletteSource
		want string
	}{
		{"a branch name", mordye.PaletteSource{Commit: "main", Format: "kitty"}, "40-character"},
		{"a short SHA", mordye.PaletteSource{Commit: "177d447", Format: "kitty"}, "40-character"},
		{"an unknown format", mordye.PaletteSource{Commit: strings.Repeat("a", 40), Format: "alacritty"}, "alacritty"},
	} {
		_, err := build(spec{src: c.src})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one mentioning %q", c.name, err, c.want)
		}
	}
}
