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
| `dux` ([Dux Shell](https://github.com/duxlabs-dev/dux-shell), formerly DankMaterialShell) | The theme file, plus settings for the flavor and accent you chose |

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

## Status

Early development. The color rules were prototyped against seven themes: Catppuccin Macchiato and Mocha, Tokyo Night, Nord, Gruvbox Material, Dracula, and Rosé Pine.

## Known Limitations

- **Dark themes only.** Mordye completes a theme's dark variant, and its tuning assumes a dark background. Light themes are [#27](https://github.com/robwilkerson/mordye/issues/27).
- **One built-in source.** Dux Shell is the only adapter so far. Any other source has to write Mordye's input format itself.
- **The format may still change.** How the input and output formats evolve, and what `sources` values look like, aren't settled yet ([#13](https://github.com/robwilkerson/mordye/issues/13), [#14](https://github.com/robwilkerson/mordye/issues/14)). Pin a version once releases exist.
- **No releases yet.** Installing means building from source with Go 1.27+ ([#2](https://github.com/robwilkerson/mordye/issues/2), [#3](https://github.com/robwilkerson/mordye/issues/3)).

## Credits

The test fixtures in `adapter/dux/testdata/` are theme files from the [Dux plugin registry](https://github.com/duxlabs-dev/dux-plugin-registry), copied unchanged so the adapter is tested against real themes. They are the work of their authors, and the palettes they carry belong to their own projects: [Catppuccin](https://catppuccin.com), [Dracula](https://draculatheme.com), [Gruvbox Material](https://github.com/sainnhe/gruvbox-material), [Nord](https://www.nordtheme.com), [Rosé Pine](https://rosepinetheme.com), and [Tokyo Night](https://github.com/folke/tokyonight.nvim).

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
