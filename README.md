# <img src="assets/logo.svg" alt="" height="48" align="top"> Mordye

[![Continuous Integration](https://github.com/robwilkerson/mordye/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/robwilkerson/mordye/actions/workflows/ci.yml)

Give Mordye a partial color palette and it returns a complete one: every interface role an app needs, plus all 16 terminal colors.

Desktop shells and theme tools describe palettes in Material roles (`primary`, `surface`, `error`, and so on), and most themes fill in only some of them. Terminals need 16 distinct colors, and few themes define them all. Tools that fill the gaps usually shift every hue toward the accent, so "blue" and "cyan" turn into shades of your accent color.

Mordye fills the gaps without bending the palette:

- **A color the theme defines is used as-is.** Its red is your red, and its yellow is your yellow.
- **A missing color is generated at its usual hue**, at the palette's own chroma and lightness, so a muted theme gets a muted green and a pastel theme a pastel one.
- **The accent stays yours.** Changing it moves `primary` and `secondary`, and leaves every other color alone.

## How It Fits Together

```
source files ──► adapter ──► generator ──► complete palette ──► your app templates
```

- **Adapters** translate one source's theme format for the generator. They are built in, so Mordye is one binary with no dependencies.
- **The generator** fills the gaps. It knows nothing about any particular source.
- **Your templates** read the complete palette, which always has the same fields.

Mordye reads only the files you name on its command line. Finding them, such as where a source keeps its settings, is the caller's job:

```bash
mordye dux --settings ~/.config/dms/settings.json \
  --theme ~/.config/dms/themes/catppuccin/theme.json > palette.json
```

| Source | Files It Takes |
|---|---|
| `dux` ([Dux Shell](https://github.com/duxlabs-dev/dux-shell), formerly DankMaterialShell) | The theme file; optionally, the settings for the flavor and accent you chose, and the theme's kitty port |

To make the terminal match the theme's own terminal port, also pass that port as a kitty theme file (see [Terminal Colors](#terminal-colors)):

```bash
mordye dux --theme ~/.config/dms/themes/dracula/theme.json \
  --terminal-palette ~/themes/dracula/kitty.conf > palette.json
```

A source with no built-in adapter can write the input format below and pipe it in: `mordye < input.json`.

## Input and Output

The input is a sparse palette:

- **`colors`:** any subset of the known roles. Only `primary` and `on_surface` are required.
- **`swatches`:** extra theme colors with no role, such as Catppuccin's accent menu. They can fill terminal slots but never become interface roles.
- **`accent`:** on a theme with an accent menu, the swatch you chose. It keeps `primary` and `secondary` out of the terminal colors, so changing the accent never moves them.
- **`terminal`:** any terminal colors the source already has. These pass through unchanged.

The palette's `name` and every swatch name must start with an ASCII letter or digit, then use only letters, digits, `.`, `_`, and `-`. Names come from theme files and end up in generated config, so Mordye rejects anything else rather than pass it along.

The output always has the same 37 fields:

- **15 interface roles**, such as `primary`, `background`, `surface_container_high`, and `on_surface`
- **22 terminal fields**: background, foreground, cursor, cursor text, selection, and the 16 normal and bright colors

A `sources` map records where each field came from: passed through, taken from a role or swatch, or generated.

## How Mordye Fills the Gaps

Every color Mordye adds comes from a rule that reads the theme's own colors. This section says what each rule reads and why; the exact math lives in [`palette.go`](palette.go). If a result looks wrong, you can point at the rule or the input behind it, not just the color.

The math happens in [OKLCH](https://bottosson.github.io/posts/oklab/), which separates lightness, chroma (how tinted a color is), and hue, and where equal steps look equal.

### Surfaces

`surface_container`, `surface_container_high`, and `surface_container_highest` form a ladder, each a step beyond the one below (lighter, in a dark theme). A missing rung is built from the rungs the theme defines:

- **Tint:** it takes the hue and chroma of the nearest defined rung, so a slate theme stays slate.
- **Lightness:** it sits the theme's own spacing away, in the theme's own direction: the average gap between the rungs the theme defines.

A theme that defines one rung gives no spacing to copy, so Mordye uses Catppuccin Macchiato's. A theme that defines none gets the whole ladder generated, along with any missing `background`, `surface`, or `outline`, as near-grays at fixed lightness with a hint of the theme's outline (or text) tint.

**Example: Nord** defines `surface_container` (`#434c5e`) and `surface_container_high` (`#4c566a`), 0.037 apart in lightness. Mordye puts `surface_container_highest` one more step up, in the same slate: `#566075`.

### Terminal Colors

**Where they come from:** most themes publish an official terminal port, the canonical terminal palette as the theme's authors define it, with a color for all 16 slots. Pass the path to one with `--terminal-palette`, and its colors (plus cursor and selection) pass through unchanged. Mordye fills only what the file leaves out.

**Getting a port:** Mordye doesn't download anything, so you need the file on disk first. Look for a kitty port linked from the theme's own site or repository. If the theme doesn't have one, kitty's [theme collection](https://github.com/kovidgoyal/kitty-themes) probably does. On a theme with several flavors, grab the port for the flavor you chose, since each flavor has its own file.

The port is a kitty theme file whatever terminal you use, because kitty ports are the easiest to find and to read: most themes ship one, and each line is a plain key and value. The format is only where the colors come from; your templates write them out for any terminal. [ADR 0001](docs/decisions/0001-kitty-ports-for-terminal-colors.md) has the reasoning.

**When they may differ from a port:** without `--terminal-palette`, Mordye matches the theme's colors to slots by hue, using the rules below. Themes that fill slots by name come out differently: Dracula's blue slot holds its purple, and Rosé Pine's green holds its pine. Ports of the same theme can also disagree with each other (Rosé Pine's foot port puts foam in green), so the colors match the port you pass, not every port.

Without a port, each chromatic slot has a usual hue: red, yellow, green, cyan, blue, and magenta. A theme color fills a slot when its hue is within 30° of the slot's, and it can fill only its nearest slot. Near-grays never fill one.

- **Ties:** a named role (such as `error`) beats a swatch, and then the closer hue wins.
- **Status colors:** `error` and `warning` claim red and yellow first, when their hue is in range.
- **Accent:** on a theme with an accent menu, `primary` and `secondary` are left out, so changing the accent never moves a terminal color.

A slot no theme color fills is generated at its usual hue, at the palette's typical lightness and a little under its typical chroma. Typical means the median across the theme's colors, so a muted theme gets a muted green and a pastel theme a pastel one.

Black is the highest surface rung the theme itself defines, and bright black is a step lighter. Bright white is the theme's least-tinted text color, and white is a step dimmer.

**Example: Dracula's** theme file defines no green. Without its port, Mordye generates one at green's usual hue, as light as Dracula's other colors and nearly as saturated: `#86c279`.

### Interface Roles

- `secondary` falls back to `primary`, and `tertiary` to `secondary`.
- `error`, `warning`, and `info` fall back to the terminal red, yellow, and blue.
- `on_primary` is `background` or `on_surface`, whichever contrasts more with `primary`.
- `on_surface_variant` is `on_surface`, a step dimmer.

Any non-ANSI terminal field a port doesn't set comes from the interface roles: the background from `background`, the foreground from `on_surface`, the cursor from `primary`, and the selection from `surface_container_high`.

### Fixed Numbers

A few rules need a number no theme supplies. They were tuned against the seven themes below, and they're the first place to look when a result seems off:

- **30°:** how far a theme color's hue may sit from a slot's and still fill it.
- **0.04 chroma:** below this, a color counts as a near-gray and fills no slot. Text colors need 0.08, since most are faintly tinted.
- **0.1 chroma** for generated colors when a theme has no tinted colors at all, at its text's lightness.
- **80%:** the share of the palette's typical chroma a generated color takes, so it sits behind the theme's own.
- **+0.05 lightness** for a generated yellow, which reads darker than other hues at the same lightness.
- **0.06 lightness:** the step from black to bright black, and from bright white to white.
- **0.1 lightness:** how much dimmer `on_surface_variant` is than `on_surface`.
- **25%:** the share of the outline's chroma a generated gray keeps.
- **0.07 lightness:** the spacing between surface rungs when a theme defines only one.
- **Catppuccin Macchiato's lightness** for each surface, when a theme defines none.

## Status

Early development. The color rules were prototyped against seven themes: Catppuccin Macchiato and Mocha, Tokyo Night, Nord, Gruvbox Material, Dracula, and Rosé Pine.

## Known Limitations

- **Dark themes only.** Mordye completes a theme's dark variant, and its tuning assumes a dark background. Light themes are [#27](https://github.com/robwilkerson/mordye/issues/27).
- **One built-in source.** Dux Shell is the only theme adapter so far, and kitty the only terminal port format. Any other source has to write Mordye's input format itself.
- **The format may still change.** How the input and output formats evolve, and what `sources` values look like, aren't settled yet ([#13](https://github.com/robwilkerson/mordye/issues/13), [#14](https://github.com/robwilkerson/mordye/issues/14)). Pin a version once releases exist.
- **No releases yet.** Installing means building from source with Go 1.27+ ([#2](https://github.com/robwilkerson/mordye/issues/2), [#3](https://github.com/robwilkerson/mordye/issues/3)).

## Credits

The test fixtures in `adapter/dux/testdata/` are theme files from the [Dux plugin registry](https://github.com/duxlabs-dev/dux-plugin-registry), copied unchanged so the adapter is tested against real themes. They are the work of their authors, and the palettes they carry belong to their own projects: [Catppuccin](https://catppuccin.com), [Dracula](https://draculatheme.com), [Gruvbox Material](https://github.com/sainnhe/gruvbox-material), [Nord](https://www.nordtheme.com), [Rosé Pine](https://rosepinetheme.com), and [Tokyo Night](https://github.com/folke/tokyonight.nvim).

The fixtures in `adapter/kitty/testdata/` are official kitty ports, also copied unchanged and MIT licensed: [Catppuccin's](https://github.com/catppuccin/kitty), [Dracula's](https://github.com/dracula/kitty), and [Rosé Pine's](https://github.com/rose-pine/kitty).

## About the Robot

Yes, an LLM helped write this. So did an editor, a compiler, a linter,
autocomplete, and a number of other tools. It's just a tool. One that "talks"
back, but still a tool. And it helps me salvage some of my weekend. It types
faster than I do, reads faster than I do, and, much to my chagrin, knows more
than I do.

Every line here was read, discussed, sometimes argued, and ultimately signed
off by a human (me). I theme my own desktop with it every day and live with the
result.

The AI doesn't get the credit and it doesn't get the blame. If something here
is broken, sloppy, or wrong, that's 100% on me and I welcome the feedback. A
bug report or a PR is always appreciated.

## License

[MIT](LICENSE). The test fixtures are credited above and are not covered by it.

## The Name

A mordant is what dyers use to fix color into fabric. It binds the dye so the colors hold and sit together evenly. Mordye is mordant plus dye, and it reads as "more dye," which is the job: adding the colors a palette is missing, and making them hold together with the ones already there.
