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
	CurrentThemeName string `json:"currentThemeName"`
	// Each theme's choice, in the shape its variants take: an object for a
	// flavors theme, a plain option id for an options theme.
	RegistryThemeVariants map[string]json.RawMessage `json:"registryThemeVariants"`
}

// choice is what the settings pick for one theme: a flavor and accent, or an
// option.
type choice struct {
	Flavor string `json:"flavor"`
	Accent string `json:"accent"`
	Option string `json:"-"`
}

type themeFile struct {
	ID       string         `json:"id"`
	Dark     map[string]any `json:"dark"`
	Variants *variants      `json:"variants"`
}

// variants comes in two shapes. A flavors theme (type "multi", such as
// Catppuccin) has flavors, each with an accent menu. An options theme (such
// as Rosé Pine) has a plain list of options, each overriding some of the
// theme's colors.
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

	Default string `json:"default"`
	Options []struct {
		ID   string         `json:"id"`
		Dark map[string]any `json:"dark"`
	} `json:"options"`
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

// palettes maps a Dux theme, by the name Parse gives it, to the bundled
// palette of that theme's official terminal port. Where Dux ships several
// themes for one port family, each takes the port with the closest dark
// background: Dux's three Tokyo Night themes other than Storm all use Night's
// darkest shade. Light themes, such as Rosé Pine's Dawn, wait for light
// support (#27).
var palettes = map[string]string{
	"catppuccin-mocha":     "catppuccin-mocha",
	"catppuccin-macchiato": "catppuccin-macchiato",
	"catppuccin-frappe":    "catppuccin-frappe",
	"dracula":              "dracula",
	"flexoki":              "flexoki-dark",
	"kanagawaWl":           "kanagawa",
	"nord":                 "nord",
	"rosePine":             "rose-pine",
	"rosePine-moon":        "rose-pine-moon",
	"solarizedOsakaNight":  "solarized-osaka-dark",
	"tokyoNight":           "tokyonight-night",
	"tokyoNightDark":       "tokyonight-night",
	"tokyoNightNightMoon":  "tokyonight-night",
	"tokyoNightStorm":      "tokyonight-storm",
}

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

	dark, swatches, name, accent, err := resolve(t, pick)
	if err != nil {
		return mordye.Input{}, err
	}

	in := mordye.Input{
		Version: mordye.Version, Name: name, Mode: "dark", Accent: accent,
		Colors: map[string]string{}, Swatches: swatches, Palette: palettes[name],
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
	raw := s.RegistryThemeVariants[t.ID]
	var c choice
	var err error
	switch {
	case len(raw) == 0:
	case raw[0] == '"':
		err = json.Unmarshal(raw, &c.Option)
	default:
		var v struct {
			Dark choice `json:"dark"`
		}
		err = json.Unmarshal(raw, &v)
		c = v.Dark
	}
	if err != nil {
		return choice{}, fmt.Errorf("reading the settings' choice for %s: %w", t.ID, err)
	}
	return c, nil
}

// resolve returns the colors the choice selects, whichever shape the theme's
// variants take, along with the swatches, name, and accent for Mordye's input.
func resolve(t themeFile, pick choice) (dark map[string]any, swatches map[string]string, name, accent string, err error) {
	switch {
	case t.Variants != nil && t.Variants.Type == "multi":
		return resolveVariant(t, pick)
	case t.Variants != nil && len(t.Variants.Options) > 0:
		dark, name, err = resolveOption(t, pick)
		return dark, map[string]string{}, name, "", err
	default:
		return t.Dark, map[string]string{}, t.ID, "", nil
	}
}

// resolveOption lays the chosen option's colors over the theme's own, as Dux
// does, falling back to the theme's default option. The default keeps the
// theme's bare id as its name; another option adds its own id.
func resolveOption(t themeFile, pick choice) (map[string]any, string, error) {
	v := t.Variants
	id := pick.Option
	if id == "" {
		id = v.Default
	}
	var ids []string
	for _, o := range v.Options {
		ids = append(ids, o.ID)
		if o.ID != id {
			continue
		}
		if o.Dark == nil {
			return nil, "", fmt.Errorf("theme %s: option %q has no dark colors", t.ID, id)
		}
		dark := map[string]any{}
		for k, val := range t.Dark {
			dark[k] = val
		}
		for k, val := range o.Dark {
			dark[k] = val
		}
		name := t.ID
		if id != v.Default {
			name += "-" + id
		}
		return dark, name, nil
	}
	return nil, "", fmt.Errorf("theme %s has no option %q; options: %v", t.ID, id, ids)
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
