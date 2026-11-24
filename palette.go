// Package mordye completes a sparse theme palette: given whatever colors a
// source defines, it returns every interface role an app needs plus all 16
// terminal colors, generating only what's missing.
//
// The rule: a color the theme defines is used as-is; a missing one is
// generated at its usual hue, at the palette's own chroma and lightness. The
// accent (primary and secondary) never moves the terminal colors.
package mordye

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// Version is the input and output format version.
const Version = 1

// Input is a sparse palette, as an adapter writes it.
type Input struct {
	Version int    `json:"version"`
	Name    string `json:"name,omitempty"`
	Mode    string `json:"mode,omitempty"`
	// Colors holds any subset of the interface roles (see Roles). primary
	// and on_surface are required.
	Colors map[string]string `json:"colors"`
	// Swatches are extra theme colors with no role, such as Catppuccin's
	// accent menu. They are candidates for terminal slots, and reach the
	// interface only as a missing error, warning, or info, by way of the
	// terminal red, yellow, or blue.
	Swatches map[string]string `json:"swatches,omitempty"`
	// Accent names the swatch the user chose as their accent, on a theme
	// that offers a menu of them. It tells Mordye that primary and secondary
	// are a choice rather than part of the theme's palette, so they stay out
	// of the terminal colors: changing the accent must never move those. The
	// chosen color stays among the swatches, so the palette keeps it. Leave
	// it empty for a theme with one fixed palette.
	Accent string `json:"accent,omitempty"`
	// Terminal holds any terminal colors the source already has. Each one
	// passes through unchanged, except the cursor when there's an Accent.
	Terminal TerminalInput `json:"terminal,omitzero"`
	// Palette names a bundled terminal palette (see BundledPaletteIDs). It
	// fills Terminal when the source has no terminal colors of its own; when
	// it has some, those win whole, so two palettes never mix.
	Palette string `json:"palette,omitempty"`
}

// TerminalInput is any subset of the output's terminal fields.
type TerminalInput struct {
	Background  string            `json:"background,omitempty"`
	Foreground  string            `json:"foreground,omitempty"`
	Cursor      string            `json:"cursor,omitempty"`
	CursorText  string            `json:"cursor_text,omitempty"`
	SelectionBg string            `json:"selection_bg,omitempty"`
	SelectionFg string            `json:"selection_fg,omitempty"`
	Normal      map[string]string `json:"normal,omitempty"`
	Bright      map[string]string `json:"bright,omitempty"`
}

// scalars returns the terminal fields that aren't ANSI slots, keyed by output
// name. validate and passThrough share it, so a field can't pass through
// without being checked.
func (t TerminalInput) scalars() map[string]string {
	return map[string]string{
		"background": t.Background, "foreground": t.Foreground,
		"cursor": t.Cursor, "cursor_text": t.CursorText,
		"selection_bg": t.SelectionBg, "selection_fg": t.SelectionFg,
	}
}

// empty reports whether t sets no color at all.
func (t TerminalInput) empty() bool {
	for _, hex := range t.scalars() {
		if hex != "" {
			return false
		}
	}
	return len(t.Normal) == 0 && len(t.Bright) == 0
}

// Output is a complete palette. Every field is always set.
type Output struct {
	Version  int      `json:"version"`
	Name     string   `json:"name,omitempty"`
	Mode     string   `json:"mode"`
	UI       UI       `json:"ui"`
	Terminal Terminal `json:"terminal"`
	// Sources records where each color came from: "colors.<role>",
	// "swatches.<name>", "terminal" (passed through), "palette.<id>" (from a
	// bundled palette), "generated", or a note on how it was derived from
	// another field.
	Sources map[string]string `json:"sources"`
}

// UI holds the interface roles, named as in Material Design.
type UI struct {
	Primary                 string `json:"primary"`
	OnPrimary               string `json:"on_primary"`
	Secondary               string `json:"secondary"`
	Tertiary                string `json:"tertiary"`
	Error                   string `json:"error"`
	Warning                 string `json:"warning"`
	Info                    string `json:"info"`
	Background              string `json:"background"`
	Surface                 string `json:"surface"`
	SurfaceContainer        string `json:"surface_container"`
	SurfaceContainerHigh    string `json:"surface_container_high"`
	SurfaceContainerHighest string `json:"surface_container_highest"`
	Outline                 string `json:"outline"`
	OnSurface               string `json:"on_surface"`
	OnSurfaceVariant        string `json:"on_surface_variant"`
}

// Terminal holds the terminal colors.
type Terminal struct {
	Background  string `json:"background"`
	Foreground  string `json:"foreground"`
	Cursor      string `json:"cursor"`
	CursorText  string `json:"cursor_text"`
	SelectionBg string `json:"selection_bg"`
	SelectionFg string `json:"selection_fg"`
	Normal      ANSI   `json:"normal"`
	Bright      ANSI   `json:"bright"`
}

// ANSI holds the eight colors of one terminal set, normal or bright.
type ANSI struct {
	Black   string `json:"black"`
	Red     string `json:"red"`
	Green   string `json:"green"`
	Yellow  string `json:"yellow"`
	Blue    string `json:"blue"`
	Magenta string `json:"magenta"`
	Cyan    string `json:"cyan"`
	White   string `json:"white"`
}

// Roles lists the interface roles an Input may set, in output order.
var Roles = []string{
	"primary", "on_primary", "secondary", "tertiary", "error", "warning", "info",
	"background", "surface", "surface_container", "surface_container_high",
	"surface_container_highest", "outline", "on_surface", "on_surface_variant",
}

// Slots lists the eight terminal slots, in ANSI order.
var Slots = []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}

// Tuning. These were set against seven themes (Catppuccin Macchiato and
// Mocha, Tokyo Night, Nord, Gruvbox Material, Dracula, Rosé Pine).
var (
	// slotHues is each chromatic slot's usual hue, in OKLCH degrees.
	slotHues = map[string]float64{
		"red": 15, "yellow": 85, "green": 140, "cyan": 200, "blue": 260, "magenta": 320,
	}
	// roleSlots gives status roles their conventional slot when their hue is
	// in range, so Dracula's green-leaning yellow still lands in yellow.
	roleSlots = map[string]string{"error": "red", "warning": "yellow"}
)

const (
	hueWindow  = 30   // how far a theme color may sit from a slot's hue and still fill it
	minChroma  = 0.04 // below this, an accent or status color counts as a neutral
	tintedText = 0.08 // text above this chroma counts as a color (Tokyo Night's teal)
	yellowLift = 0.05 // lightness added to a generated yellow, which reads dark otherwise
	brightStep = 0.06 // lightness from black to bright black, and white to bright white
	// rungStep is the lightness between surface rungs when a theme defines only
	// one and so gives no spacing to copy: Catppuccin Macchiato's own spacing.
	rungStep = 0.07
	// A generated gray keeps this share of its source's chroma: a hint of the
	// theme's warmth or coolness, short of tinting it (Gruvbox went brown at 0.6).
	neutralChroma = 0.25
	// A generated color takes this share of the palette's typical chroma, so
	// a color the theme never chose sits behind the ones it did.
	generatedChroma = 0.8
)

// Lightness of each generated neutral, set near Catppuccin Macchiato's own
// (base, mantle, surface0, surface1, overlay0).
var neutralLightness = map[string]float64{
	"background": 0.28, "surface": 0.25, "surface_container": 0.28,
	"surface_container_high": 0.35, "surface_container_highest": 0.42, "outline": 0.55,
}

// accentRoles may fill a terminal slot; structural roles (surfaces, outline)
// never do.
var accentRoles = []string{"primary", "secondary", "tertiary", "error", "warning", "info"}

var textRoles = []string{"on_surface", "on_surface_variant"}

type candidate struct {
	source string // "colors.<role>" or "swatches.<name>"
	role   string // the role name, empty for a swatch
	hex    string
	lch    lch
}

// Complete fills every gap in a sparse palette.
func Complete(in Input) (Output, error) {
	termSrc := "terminal"
	if in.Palette != "" {
		pal, ok, err := BundledPalette(in.Palette)
		if err != nil {
			return Output{}, err
		}
		if !ok {
			ids, _ := BundledPaletteIDs()
			return Output{}, fmt.Errorf("palette %q is not bundled; bundled: %v", in.Palette, ids)
		}
		if in.Terminal.empty() {
			in.Terminal, termSrc = pal.Terminal, "palette."+pal.ID
		}
	}
	if err := validate(in); err != nil {
		return Output{}, err
	}
	p := &builder{in: in, termSrc: termSrc, ui: map[string]string{}, term: map[string]string{}, src: map[string]string{}}
	for role, hex := range in.Colors {
		p.ui[role], p.src["ui."+role] = hex, "colors."+role
	}
	p.fillNeutrals()
	p.fillSlots()
	p.fillUI()
	p.fillTerminalExtras()
	p.passThrough()
	return p.output(), nil
}

// nameRule describes the characters a palette or swatch name may use.
const nameRule = "start with an ASCII letter or digit, then use only letters, digits, '.', '_', and '-'"

// safeName reports whether s follows nameRule. Names reach templates that
// write config files, where a newline, quote, or escape code could inject
// config. The leading letter or digit keeps names like ".." and "-rf" out of
// file paths and shell commands.
func safeName(s string) bool {
	for i, r := range s {
		alnum := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		if !alnum && (i == 0 || r != '.' && r != '_' && r != '-') {
			return false
		}
	}
	return s != ""
}

func validate(in Input) error {
	if in.Version != Version {
		return fmt.Errorf("input version %d: want %d", in.Version, Version)
	}
	if in.Name != "" && !safeName(in.Name) {
		return fmt.Errorf("name %q: %s", in.Name, nameRule)
	}
	for _, role := range []string{"primary", "on_surface"} {
		if in.Colors[role] == "" {
			return fmt.Errorf("input is missing colors.%s, which every palette needs", role)
		}
	}
	check := func(field, hex string) error {
		if _, err := parseHex(hex); err != nil {
			return fmt.Errorf("%s: %w", field, err)
		}
		return nil
	}
	for role, hex := range in.Colors {
		if !slices.Contains(Roles, role) {
			return fmt.Errorf("colors: %q is not a known role; known roles: %v", role, Roles)
		}
		if err := check("colors."+role, hex); err != nil {
			return err
		}
	}
	for name, hex := range in.Swatches {
		if !safeName(name) {
			return fmt.Errorf("swatch name %q: %s", name, nameRule)
		}
		if err := check("swatches."+name, hex); err != nil {
			return err
		}
	}
	if _, ok := in.Swatches[in.Accent]; in.Accent != "" && !ok {
		return fmt.Errorf("accent %q is not among the swatches; the chosen accent must stay in the menu", in.Accent)
	}
	for set, colors := range map[string]map[string]string{"normal": in.Terminal.Normal, "bright": in.Terminal.Bright} {
		for slot, hex := range colors {
			if !slices.Contains(Slots, slot) {
				return fmt.Errorf("terminal.%s: %q is not a terminal slot; slots: %v", set, slot, Slots)
			}
			if err := check("terminal."+set+"."+slot, hex); err != nil {
				return err
			}
		}
	}
	for field, hex := range in.Terminal.scalars() {
		if hex == "" {
			continue
		}
		if err := check("terminal."+field, hex); err != nil {
			return err
		}
	}
	return nil
}

type builder struct {
	in Input
	// termSrc is where in.Terminal came from, for sources: "terminal" or
	// "palette.<id>".
	termSrc string
	ui      map[string]string // role -> hex
	term    map[string]string // "normal.red", "cursor", ... -> hex
	src     map[string]string // output field -> where it came from
}

func (p *builder) lchOf(hex string) lch {
	c, _ := parseHex(hex) // validated or generated, so always well-formed
	return c.lch()
}

// neutralBase is the hue and chroma generated grays take: the theme's
// outline, else its text color.
func (p *builder) neutralBase() lch {
	if hex, ok := p.ui["outline"]; ok {
		return p.lchOf(hex)
	}
	return p.lchOf(p.ui["on_surface"])
}

// rungs are the surface ladder, darkest first: each sits a step lighter than
// the one below it.
var rungs = []string{"surface_container", "surface_container_high", "surface_container_highest"}

// fillRungs generates missing surface rungs from the ones the theme defines.
// A missing rung takes the hue and chroma of the nearest defined rung, and
// sits the theme's own spacing away from it: the average gap between its
// defined rungs, or rungStep when it defines only one. With no rungs defined
// there is nothing to copy, and fillNeutrals generates them.
func (p *builder) fillRungs() {
	var defined []int
	for i, role := range rungs {
		if _, ok := p.ui[role]; ok {
			defined = append(defined, i)
		}
	}
	if len(defined) == 0 {
		return
	}
	first, last := defined[0], defined[len(defined)-1]
	step := rungStep
	if last > first {
		step = (p.lchOf(p.ui[rungs[last]]).L - p.lchOf(p.ui[rungs[first]]).L) / float64(last-first)
	}
	for i, role := range rungs {
		if _, ok := p.ui[role]; ok {
			continue
		}
		near := defined[0]
		for _, d := range defined {
			if abs(float64(d-i)) < abs(float64(near-i)) {
				near = d
			}
		}
		c := p.lchOf(p.ui[rungs[near]])
		c.L += float64(i-near) * step
		p.ui[role], p.src["ui."+role] = c.hex(), "generated"
	}
}

// fillNeutrals generates any surface or outline role the theme leaves out.
func (p *builder) fillNeutrals() {
	p.fillRungs()
	base := p.neutralBase()
	for _, role := range []string{"background", "surface", "surface_container", "surface_container_high", "surface_container_highest", "outline"} {
		if _, ok := p.ui[role]; ok {
			continue
		}
		p.ui[role] = lch{L: neutralLightness[role], C: base.C * neutralChroma, H: base.H}.hex()
		p.src["ui."+role] = "generated"
	}
}

// candidates are the theme colors that may fill a chromatic slot. With an
// accent choice (see Input.Accent), primary and secondary are left out;
// otherwise they are the theme's own colors and count like any other.
func (p *builder) candidates() []candidate {
	var out []candidate
	add := func(source, role, hex string, min float64) {
		if c := p.lchOf(hex); c.C >= min {
			out = append(out, candidate{source: source, role: role, hex: hex, lch: c})
		}
	}
	for _, role := range accentRoles {
		if p.in.Accent != "" && (role == "primary" || role == "secondary") {
			continue
		}
		if hex, ok := p.in.Colors[role]; ok {
			add("colors."+role, role, hex, minChroma)
		}
	}
	for _, role := range textRoles {
		if hex, ok := p.in.Colors[role]; ok {
			add("colors."+role, role, hex, tintedText)
		}
	}
	for name, hex := range p.in.Swatches {
		add("swatches."+name, "", hex, minChroma)
	}
	return out
}

// slotFor is the one slot a theme color may fill: a status role's
// conventional slot when its hue is in range, else the nearest hue.
func slotFor(c candidate) string {
	if home, ok := roleSlots[c.role]; ok && hueGap(slotHues[home], c.lch.H) <= hueWindow {
		return home
	}
	best, gap := "", 360.0
	for _, slot := range Slots {
		h, ok := slotHues[slot]
		if ok && hueGap(h, c.lch.H) < gap {
			best, gap = slot, hueGap(h, c.lch.H)
		}
	}
	return best
}

func median(xs []float64) float64 {
	sort.Float64s(xs)
	return xs[len(xs)/2]
}

// fillSlots fills the 16 terminal colors. A generated color takes the
// candidates' typical lightness and chroma, which is what keeps it at the
// palette's intensity.
func (p *builder) fillSlots() {
	cands := p.candidates()
	var ls, cs []float64
	for _, c := range cands {
		ls, cs = append(ls, c.lch.L), append(cs, c.lch.C)
	}
	if len(ls) == 0 {
		// Only neutrals to go on: take the lightness of the text color and a
		// modest chroma, so generated colors are at least legible.
		ls, cs = []float64{p.lchOf(p.ui["on_surface"]).L}, []float64{0.1}
	}
	typical := lch{L: median(ls), C: median(cs)}

	for _, slot := range Slots {
		hue, ok := slotHues[slot]
		if !ok {
			continue
		}
		// Named roles first (the theme's own semantic picks), then swatches;
		// within each group, the nearest hue, then the name for stability.
		var best *candidate
		better := func(a, b candidate) bool {
			if (a.role == "") != (b.role == "") {
				return a.role != ""
			}
			if ga, gb := hueGap(a.lch.H, hue), hueGap(b.lch.H, hue); ga != gb {
				return ga < gb
			}
			return a.source < b.source
		}
		for i, c := range cands {
			if hueGap(c.lch.H, hue) > hueWindow || slotFor(c) != slot {
				continue
			}
			if best == nil || better(c, *best) {
				best = &cands[i]
			}
		}
		if best != nil {
			p.setSlot(slot, best.hex, best.source)
			continue
		}
		L := typical.L
		if slot == "yellow" {
			L += yellowLift
		}
		p.setSlot(slot, lch{L: L, C: typical.C * generatedChroma, H: hue}.hex(), "generated")
	}

	// Black: the theme's own raised surface, preferring the highest. Only when
	// the theme defines neither does it take the generated one.
	blackRole := "surface_container_highest"
	if _, ok := p.in.Colors[blackRole]; !ok {
		if _, ok := p.in.Colors["surface_container_high"]; ok {
			blackRole = "surface_container_high"
		}
	}
	black := p.ui[blackRole]
	p.set("normal.black", black, "ui."+blackRole)
	// Bright black steps from that surface even when the source passes its own
	// black. Programs draw dim text in bright black on the background, and a
	// step from a pure-black black would be invisible; themes with one pair it
	// with a mid gray instead.
	brightBlackSrc := "normal.black, lighter"
	if _, ok := p.in.Terminal.Normal["black"]; ok {
		brightBlackSrc = "ui." + blackRole + ", lighter"
	}
	b := p.lchOf(black)
	p.set("bright.black", lch{L: b.L + brightStep, C: b.C, H: b.H}.hex(), brightBlackSrc)

	// Whites: bright white is the least-tinted text color that is at least as
	// light as the terminal foreground, so it never reads dimmer than plain
	// text. The foreground itself always qualifies. White is a step dimmer.
	fg, fgSrc := p.ui["on_surface"], "colors.on_surface"
	if hex := p.in.Terminal.Foreground; hex != "" {
		fg, fgSrc = hex, p.termSrc+".foreground"
	}
	white, whiteSrc := fg, fgSrc
	fgL := p.lchOf(fg).L
	for _, role := range []string{"on_surface", "on_surface_variant"} {
		hex, ok := p.in.Colors[role]
		if !ok {
			continue
		}
		if c := p.lchOf(hex); c.L >= fgL && c.C < p.lchOf(white).C {
			white, whiteSrc = hex, "colors."+role
		}
	}
	p.set("bright.white", white, whiteSrc)
	// White follows a bright white the source passes, which passThrough
	// applies, so the pair stays a pair.
	if hex, ok := p.in.Terminal.Bright["white"]; ok {
		white = hex
	}
	w := p.lchOf(white)
	p.set("normal.white", lch{L: w.L - brightStep, C: w.C, H: w.H}.hex(), "bright.white, dimmer")
}

func (p *builder) setSlot(slot, hex, source string) {
	p.set("normal."+slot, hex, source)
	p.set("bright."+slot, hex, source)
}

func (p *builder) set(field, hex, source string) {
	p.term[field], p.src["terminal."+field] = hex, source
}

// fillUI fills the interface roles still missing after the neutrals.
func (p *builder) fillUI() {
	fallback := func(role, from, note string) {
		if _, ok := p.ui[role]; !ok {
			p.ui[role], p.src["ui."+role] = p.ui[from], note
		}
	}
	fallback("secondary", "primary", "ui.primary")
	fallback("tertiary", "secondary", "ui.secondary")
	if _, ok := p.ui["on_surface_variant"]; !ok {
		t := p.lchOf(p.ui["on_surface"])
		p.ui["on_surface_variant"] = lch{L: t.L - 0.1, C: t.C, H: t.H}.hex()
		p.src["ui.on_surface_variant"] = "ui.on_surface, dimmer"
	}
	// A missing status role takes the terminal's red, yellow, or blue as the
	// terminal will show it: the source's own color passes through later, so
	// it's taken here first, over any color Mordye picked for the slot.
	for role, slot := range map[string]string{"error": "red", "warning": "yellow", "info": "blue"} {
		if _, ok := p.ui[role]; !ok {
			hex, ok := p.in.Terminal.Normal[slot]
			if !ok {
				hex = p.term["normal."+slot]
			}
			p.ui[role], p.src["ui."+role] = hex, "terminal.normal."+slot
		}
	}
	if _, ok := p.ui["on_primary"]; !ok {
		// Whichever of the background and the text contrasts more with primary.
		pl := p.lchOf(p.ui["primary"]).L
		role := "background"
		if abs(p.lchOf(p.ui["on_surface"]).L-pl) > abs(p.lchOf(p.ui["background"]).L-pl) {
			role = "on_surface"
		}
		p.ui["on_primary"], p.src["ui.on_primary"] = p.ui[role], "ui."+role
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// fillTerminalExtras fills the non-ANSI terminal fields from the interface
// roles. The cursor is chrome rather than a color with a meaning, so on a
// theme with an accent menu it shows the accent the user chose. Otherwise it's
// the foreground, as most themes' official terminal ports draw it.
func (p *builder) fillTerminalExtras() {
	roles := map[string]string{
		"background": "background", "foreground": "on_surface",
		"cursor": "on_surface", "cursor_text": "background",
		"selection_bg": "surface_container_high", "selection_fg": "on_surface",
	}
	if p.in.Accent != "" {
		roles["cursor"], roles["cursor_text"] = "primary", "on_primary"
	}
	for field, role := range roles {
		p.set(field, p.ui[role], "ui."+role)
	}
}

// passThrough applies the source's own terminal colors over everything else,
// except that a chosen accent keeps the cursor: it's chrome, and showing the
// accent there is the point of choosing one.
func (p *builder) passThrough() {
	t := p.in.Terminal
	for field, hex := range t.scalars() {
		accentOwns := p.in.Accent != "" && (field == "cursor" || field == "cursor_text")
		if hex != "" && !accentOwns {
			p.set(field, hex, p.termSrc)
		}
	}
	// Without an accent the cursor is the foreground, so a source that sets its
	// own foreground but no cursor gets a cursor in that foreground.
	if p.in.Accent == "" {
		if t.Cursor == "" && t.Foreground != "" {
			p.set("cursor", t.Foreground, p.termSrc+".foreground")
		}
		if t.CursorText == "" && t.Background != "" {
			p.set("cursor_text", t.Background, p.termSrc+".background")
		}
	}
	for slot, hex := range t.Normal {
		p.set("normal."+slot, hex, p.termSrc)
	}
	for slot, hex := range t.Bright {
		p.set("bright."+slot, hex, p.termSrc)
	}
}

// output assembles the palette. Every color is lowercased here, whatever case
// it arrived in, so callers can compare colors as strings.
func (p *builder) output() Output {
	term := func(field string) string { return strings.ToLower(p.term[field]) }
	ansi := func(set string) ANSI {
		g := func(slot string) string { return term(set + "." + slot) }
		return ANSI{g("black"), g("red"), g("green"), g("yellow"), g("blue"), g("magenta"), g("cyan"), g("white")}
	}
	u := func(role string) string { return strings.ToLower(p.ui[role]) }
	mode := p.in.Mode
	if mode == "" {
		mode = "dark"
	}
	return Output{
		Version: Version,
		Name:    p.in.Name,
		Mode:    mode,
		UI: UI{
			u("primary"), u("on_primary"), u("secondary"), u("tertiary"), u("error"), u("warning"), u("info"),
			u("background"), u("surface"), u("surface_container"), u("surface_container_high"),
			u("surface_container_highest"), u("outline"), u("on_surface"), u("on_surface_variant"),
		},
		Terminal: Terminal{
			Background: term("background"), Foreground: term("foreground"),
			Cursor: term("cursor"), CursorText: term("cursor_text"),
			SelectionBg: term("selection_bg"), SelectionFg: term("selection_fg"),
			Normal: ansi("normal"), Bright: ansi("bright"),
		},
		Sources: p.src,
	}
}

// get returns one slot's color by name.
func (a ANSI) get(slot string) string {
	return map[string]string{
		"black": a.Black, "red": a.Red, "green": a.Green, "yellow": a.Yellow,
		"blue": a.Blue, "magenta": a.Magenta, "cyan": a.Cyan, "white": a.White,
	}[slot]
}
