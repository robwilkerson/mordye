# mordye

[![Continuous Integration](https://github.com/robwilkerson/mordye/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/robwilkerson/mordye/actions/workflows/ci.yml)

Give mordye a partial color palette and it returns a complete one: every interface role an app needs, plus all 16 terminal colors.

Desktop shells and theme tools describe palettes in Material roles (`primary`, `surface`, `error`, and so on), and most themes fill in only some of them. Terminals need 16 distinct colors, and few themes define them all. Tools that fill the gaps usually shift every hue toward the accent, so "blue" and "cyan" turn into shades of your accent color.

mordye fills the gaps without bending the palette:

- **A color the theme defines is used as-is.** Its red is your red, and its yellow is your yellow.
- **A missing color is generated at its usual hue**, at the palette's own chroma and lightness, so a muted theme gets a muted green and a pastel theme a pastel one.
- **The accent stays yours.** Changing it moves `primary` and `secondary`, and leaves every other color alone.

## How It Fits Together

```
source files ──► adapter ──► generator ──► complete palette ──► your app templates
```

- **Adapters** translate one source's theme format for the generator. They are built in, so mordye is one binary with no dependencies.
- **The generator** fills the gaps. It knows nothing about any particular source.
- **Your templates** read the complete palette, which always has the same fields.

mordye reads only the files you name on its command line. Finding them, such as where a source keeps its settings, is the caller's job:

```bash
mordye dux --settings ~/.config/dms/settings.json \
  --theme ~/.config/dms/themes/catppuccin/theme.json > palette.json
```

| Source | Files It Takes |
|---|---|
| `dux` ([Dux Shell](https://github.com/duxlabs-dev/dux-shell), formerly DankMaterialShell) | The theme file, plus settings for the flavor and accent you chose |

A source with no built-in adapter can write the input format below and pipe it in: `mordye < input.json`.

## Input and Output

The input is a sparse palette:

- **`colors`:** any subset of the known roles. Only `primary` and `on_surface` are required.
- **`swatches`:** extra theme colors with no role, such as Catppuccin's accent menu. They can fill terminal slots but never become interface roles.
- **`accent`:** on a theme with an accent menu, the swatch you chose. It keeps `primary` and `secondary` out of the terminal colors, so changing the accent never moves them.
- **`terminal`:** any terminal colors the source already has. These pass through unchanged.

The palette's `name` and every swatch name must start with an ASCII letter or digit, then use only letters, digits, `.`, `_`, and `-`. Names come from theme files and end up in generated config, so mordye rejects anything else rather than pass it along.

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

Each chromatic slot has a usual hue: red, yellow, green, cyan, blue, and magenta. A theme color fills a slot when its hue is within 30° of the slot's, and it can fill only its nearest slot. Near-grays never fill one.

- **Ties:** a named role (such as `error`) beats a swatch, and then the closer hue wins.
- **Status colors:** `error` and `warning` claim red and yellow first, when their hue is in range.
- **Accent:** on a theme with an accent menu, `primary` and `secondary` are left out, so changing the accent never moves a terminal color.

A slot no theme color fills is generated at its usual hue, at the palette's typical lightness and a little under its typical chroma. Typical means the median across the theme's colors, so a muted theme gets a muted green and a pastel theme a pastel one.

Black is the highest surface rung the theme itself defines, and bright black is a step lighter. Bright white is the theme's least-tinted text color, and white is a step dimmer.

**Example: Dracula's** theme file defines no green. Mordye generates one at green's usual hue, as light as Dracula's other colors and nearly as saturated: `#86c279`.

### Interface Roles

- `secondary` falls back to `primary`, and `tertiary` to `secondary`.
- `error`, `warning`, and `info` fall back to the terminal red, yellow, and blue.
- `on_primary` is `background` or `on_surface`, whichever contrasts more with `primary`.
- `on_surface_variant` is `on_surface`, a step dimmer.

The non-ANSI terminal fields come from the interface roles: the background from `background`, the foreground from `on_surface`, the cursor from `primary`, and the selection from `surface_container_high`.

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

## Credits

The test fixtures in `adapter/dux/testdata/` are theme files from the [Dux plugin registry](https://github.com/duxlabs-dev/dux-plugin-registry), copied unchanged so the adapter is tested against real themes. They are the work of their authors, and the palettes they carry belong to their own projects: [Catppuccin](https://catppuccin.com), [Dracula](https://draculatheme.com), [Gruvbox Material](https://github.com/sainnhe/gruvbox-material), [Nord](https://www.nordtheme.com), [Rosé Pine](https://rosepinetheme.com), and [Tokyo Night](https://github.com/folke/tokyonight.nvim).

## License

[MIT](LICENSE). The test fixtures are credited above and are not covered by it.

## The Name

A mordant is what dyers use to fix color into fabric. It binds the dye so the colors hold and sit together evenly. mordye is mordant plus dye, and it reads as "more dye," which is the job: adding the colors a palette is missing, and making them hold together with the ones already there.
