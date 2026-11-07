// Package dux translates a Dux Shell (formerly DankMaterialShell) theme into
// Mordye input.
//
// It takes file contents, never paths: the caller reads Dux's settings and
// the theme file those settings select, and hands both over. Settings carry
// the choice (which flavor and accent); the theme file carries the colors.
package dux

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/robwilkerson/mordye"
)

type settingsFile struct {
	CurrentThemeName      string `json:"currentThemeName"`
	RegistryThemeVariants map[string]struct {
		Dark choice `json:"dark"`
	} `json:"registryThemeVariants"`
}

type choice struct {
	Flavor string `json:"flavor"`
	Accent string `json:"accent"`
}

type themeFile struct {
	ID       string         `json:"id"`
	Dark     map[string]any `json:"dark"`
	Variants *variants      `json:"variants"`
}

type variants struct {
	Type     string `json:"type"`
	Defaults struct {
		Dark choice `json:"dark"`
	} `json:"defaults"`
	Flavors []struct {
		ID   string         `json:"id"`
		Dark map[string]any `json:"dark"`
	} `json:"flavors"`
	Accents []map[string]any `json:"accents"`
}

// roles maps Dux theme roles to Mordye's. The two text roles are handled
// separately; see textRoles.
var roles = map[string]string{
	"primary": "primary", "primaryText": "on_primary", "secondary": "secondary",
	"error": "error", "warning": "warning", "info": "info",
	"background": "background", "surface": "surface",
	"surfaceContainer":        "surface_container",
	"surfaceContainerHigh":    "surface_container_high",
	"surfaceContainerHighest": "surface_container_highest",
	"outline":                 "outline", "surfaceVariantText": "on_surface_variant",
}

// textRoles hold the plain text color, but themes disagree on which: Nord's
// backgroundText is teal, Tokyo Night's surfaceText is. The less saturated
// one becomes on_surface, and the other goes in as a swatch, so a tinted text
// color can still fill a terminal slot.
var textRoles = []string{"backgroundText", "surfaceText"}

// Parse translates Dux settings and the theme file they select into Mordye
// input. Settings may be nil, in which case the theme's defaults apply.
func Parse(settings, theme []byte) (mordye.Input, error) {
	var t themeFile
	if err := json.Unmarshal(theme, &t); err != nil {
		return mordye.Input{}, fmt.Errorf("reading the theme file: %w", err)
	}
	if t.ID == "" {
		return mordye.Input{}, fmt.Errorf("the theme file has no id")
	}
	pick, err := choiceFor(settings, t)
	if err != nil {
		return mordye.Input{}, err
	}

	dark, swatches, name, accent := t.Dark, map[string]string{}, t.ID, ""
	if t.Variants != nil && t.Variants.Type == "multi" {
		dark, swatches, name, accent, err = resolveVariant(t, pick)
		if err != nil {
			return mordye.Input{}, err
		}
	}

	in := mordye.Input{
		Version: mordye.Version, Name: name, Mode: "dark", Accent: accent,
		Colors: map[string]string{}, Swatches: swatches,
	}
	for from, to := range roles {
		if hex, ok := dark[from].(string); ok {
			in.Colors[to] = hex
		}
	}
	texts, err := plainestFirst(dark)
	if err != nil {
		return mordye.Input{}, fmt.Errorf("theme %s: %w", t.ID, err)
	}
	if len(texts) == 0 {
		return mordye.Input{}, fmt.Errorf("theme %s has no text color (%v)", t.ID, textRoles)
	}
	in.Colors["on_surface"] = dark[texts[0]].(string)
	for _, role := range texts[1:] {
		if hex := dark[role].(string); hex != in.Colors["on_surface"] {
			in.Swatches[role] = hex
		}
	}
	if len(in.Swatches) == 0 {
		in.Swatches = nil
	}
	return in, nil
}

// choiceFor reads the flavor and accent the settings choose for this theme.
// The settings also name the theme file (customThemeFile); resolving it is
// the caller's job, since theme folder names don't reliably follow theme ids.
func choiceFor(settings []byte, t themeFile) (choice, error) {
	if settings == nil {
		return choice{}, nil
	}
	var s settingsFile
	if err := json.Unmarshal(settings, &s); err != nil {
		return choice{}, fmt.Errorf("reading the settings: %w", err)
	}
	if s.CurrentThemeName != "custom" {
		return choice{}, fmt.Errorf("the active theme (%s) isn't a fixed theme; mordye reads fixed themes only", s.CurrentThemeName)
	}
	return s.RegistryThemeVariants[t.ID].Dark, nil
}

// resolveVariant picks the chosen flavor and accent from a multi-variant
// theme, falling back to the theme's defaults. The whole accent menu becomes
// swatches, the chosen accent included, and the accent is returned for
// mordye.Input.Accent.
func resolveVariant(t themeFile, pick choice) (dark map[string]any, swatches map[string]string, name, accent string, err error) {
	v := t.Variants
	if pick.Flavor == "" {
		pick.Flavor = v.Defaults.Dark.Flavor
	}
	if pick.Accent == "" {
		pick.Accent = v.Defaults.Dark.Accent
	}

	var flavors []string
	for _, f := range v.Flavors {
		flavors = append(flavors, f.ID)
		if f.ID == pick.Flavor {
			dark = map[string]any{}
			for k, val := range f.Dark {
				dark[k] = val
			}
		}
	}
	if dark == nil {
		return nil, nil, "", "", fmt.Errorf("theme %s has no flavor %q; flavors: %v", t.ID, pick.Flavor, flavors)
	}

	swatches = map[string]string{}
	var accents []string
	found := false
	for _, a := range v.Accents {
		id, _ := a["id"].(string)
		colors, _ := a[pick.Flavor].(map[string]any)
		if id == "" || colors == nil {
			continue
		}
		accents = append(accents, id)
		if hex, ok := colors["primary"].(string); ok {
			swatches[id] = hex
		}
		if id == pick.Accent {
			found = true
			for k, val := range colors {
				dark[k] = val
			}
		}
	}
	if !found {
		sort.Strings(accents)
		return nil, nil, "", "", fmt.Errorf("theme %s has no accent %q for flavor %s; accents: %v", t.ID, pick.Accent, pick.Flavor, accents)
	}
	return dark, swatches, t.ID + "-" + pick.Flavor, pick.Accent, nil
}

// plainestFirst returns the text roles the theme defines, least saturated
// first.
func plainestFirst(dark map[string]any) ([]string, error) {
	chroma := map[string]float64{}
	var present []string
	for _, role := range textRoles {
		hex, ok := dark[role].(string)
		if !ok {
			continue
		}
		c, err := mordye.Chroma(hex)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", role, err)
		}
		chroma[role] = c
		present = append(present, role)
	}
	slices.SortStableFunc(present, func(a, b string) int {
		switch {
		case chroma[a] < chroma[b]:
			return -1
		case chroma[a] > chroma[b]:
			return 1
		}
		return 0
	})
	return present, nil
}
