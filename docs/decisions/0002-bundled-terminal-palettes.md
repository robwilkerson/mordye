# 2. Bundled Terminal Palettes

Date: 2026-11-17

## Status

Accepted ([#35](https://github.com/robwilkerson/mordye/issues/35)). Supersedes [ADR 0001](0001-kitty-ports-for-terminal-colors.md).

## Context

ADR 0001 added `--terminal-palette FILE`, which reads a theme's kitty port at runtime. Without the file, Mordye matches colors to slots by hue, which won't equal the theme's own palette. Every caller of a well-known theme would have to find and download the same port, and match theme names to port files itself.

Mordye can carry those palettes as reference data instead, so every caller gets them for free. Two constraints shape how:

- **Licensing.** Mordye is MIT. Kitty's theme collection is GPL-3.0, and many of its files declare no license. Building either into the binary isn't an option.
- **Fidelity.** Ports of the same theme can disagree (iTerm2's Catppuccin brights are lighter than Catppuccin's own). Only a port by the theme's authors defines what "true to the theme" means.

## Decision

Mordye embeds a curated set of terminal palettes in the binary.

**What qualifies**

- A port published by the theme's own authors, under a license compatible with MIT (MIT, Apache-2.0, BSD, and the like).
- Any port format. The format only matters to the converter that reads it.
- Light and dark variants alike. Light palettes wait on [#27](https://github.com/robwilkerson/mordye/issues/27) to be used, but they're stored now.

**Starting set:** the themes Dux ships that have a qualifying port. Others arrive by contribution.

| Theme | Variants | Official port | License |
|---|---|---|---|
| Catppuccin | Mocha, Macchiato, Frappé, Latte | catppuccin/kitty | MIT |
| Dracula | Dracula | dracula/kitty | MIT |
| Flexoki | Dark, Light | kepano/flexoki | MIT |
| Kanagawa | Wave, Dragon, Lotus | rebelot/kanagawa.nvim | MIT |
| Nord | Nord | nordtheme/xresources | MIT |
| Rosé Pine | Main, Moon, Dawn | rose-pine/kitty | MIT |
| Solarized Osaka | Dark, Light | craftzdog/solarized-osaka.nvim | Apache-2.0 |
| Tokyo Night | Night, Storm, Moon, Day | folke/tokyonight.nvim | Apache-2.0 |

**Second batch:** themes whose authors publish terminal colors only inside a Vim colorscheme, built from a palette table. They follow once the first set ships, through one Vim converter:

- It runs Neovim headless, loads the colorscheme with the options that pick a variant (such as `everforest_background=hard`), and reads back `g:terminal_color_0` through `g:terminal_color_15`.
- It reads what Neovim renders, so it needs no Vimscript parsing and works for any colorscheme that sets terminal colors.
- The options are part of the palette's recorded source, so CI reproduces the same colors.
- Neovim is needed only to convert, by maintainers and CI. Mordye itself gains nothing at runtime.

| Theme | Variants | Official source | License |
|---|---|---|---|
| Everforest | Hard, Medium, Soft; dark and light | sainnhe/everforest | MIT |
| Gruvbox | Hard, Medium, Soft; dark and light | morhetz/gruvbox | MIT, stated in its README |
| Gruvbox Material | Hard, Medium, Soft; dark and light | sainnhe/gruvbox-material | MIT |

**How palettes are stored**

- Each palette is one JSON file in Mordye's own terminal shape, named by its id. Mordye reads only that at runtime.
- An id follows the upstream port's own naming: Catppuccin's `mocha.conf` becomes `catppuccin-mocha`.
- Each file records its upstream URL, the commit it was taken from, and the upstream copyright and license. MIT and Apache both require the notice to travel with the colors.
- The license record points to where the license is stated, at that same commit: a LICENSE file, or a README line when that's all the project has. If a project later changes its license, Mordye drops the palette.
- A converter per port format (kitty first, then Xresources for Nord) writes these files from the upstream port at a pinned commit. Nobody edits them by hand.
- Converters live in one `converters` package, a file per format, and run only ahead of time, in the palette tool. Adapters stay about sources: they pass a palette id and never read a port.
- `--terminal-palette` is dropped before it ships, so Mordye reads no port file at runtime. A caller with its own terminal colors passes them in Mordye's input format.
- CI reruns the converters from each file's recorded source. Any difference fails the build.

**How a palette is chosen**

- `Input.Palette` names a palette id. `Name` stays as it is, since the output echoes it.
- Each adapter maps its own theme names to ids. The Dux adapter maps `rosePine` to `rose-pine`, and Catppuccin with the `mocha` flavor to `catppuccin-mocha`. Knowledge of a source's names stays in its adapter.
- A bundled palette applies only when the input has no terminal colors of its own. Input terminal colors win, all or nothing; two palettes never mix.
- Colors from a bundled palette show in `sources` as `palette.<id>`. Colors from the input stay `terminal`.
- An unknown id is an error. A test checks that every id an adapter maps to is bundled.

**Adding a theme**

1. Write a converter, if the port's format is new.
2. Run the converter on the official port, writing the palette where Mordye expects it.
3. Map the theme in each adapter that knows it, and credit the port in the README.
4. Open a PR with all of the above.

Mordye still never reads or downloads anything at runtime that it isn't given; the palettes are part of the binary.

## Consequences

- Well-known themes come out right with no extra file, and callers such as a theme engine need no palette collection or name matching of their own.
- Mordye now carries theme data. When a theme updates its port, Mordye's copy needs a rerun of the converter and a release.
- A newly bundled palette changes the output for that theme. That's intended, and the changelog names it.
- Themes with no official, compatible source (Modus is GPL; Adwaita has only third-party ports) keep generated terminal colors.
- The release must ship the upstream license notices alongside the binary.
