# Changelog

What changes in each release, written for humans. Full detail lives in git;
this is the readable digest.

Versions follow [semantic versioning](https://semver.org). Until 1.0, a minor
version may still change the input or output format
([#13](https://github.com/robwilkerson/mordye/issues/13)).

Items under **Unreleased** are merged but not yet released. `just release`
moves them under a version heading when it cuts the release.

## Unreleased

- **Terminal colors from the theme's own palette.** Mordye now carries the
  official terminal palettes of Catppuccin, Dracula, Flexoki, Kanagawa, Nord,
  Rosé Pine, Solarized Osaka, and Tokyo Night. `mordye dux` uses them with no
  extra step, so your terminal matches the theme, including themes like
  Dracula that put colors in slots by name. These themes' terminal colors
  change from Mordye's guesses to the official ones, and `sources` shows them
  as `palette.<id>`. Mordye's own input takes a new `palette` field to choose
  one by id.
- **Each release ships its license notices.** The tarball and RPM now include
  Mordye's license and the license of every bundled palette.
- **The cursor follows the accent only when there is one.** On a theme with an
  accent menu, the terminal cursor takes the accent you chose, even over a
  terminal palette's own cursor. On a theme without one, it's now the
  foreground color, as most themes draw it, instead of `primary`.
- **Every color comes out lowercase.** Colors a theme writes in uppercase used
  to pass through that way, mixed with lowercase generated ones. Now the whole
  palette is lowercase `#rrggbb`.
- **Dux theme options work.** Choosing an option on a theme like Rosé Pine
  (Moon or Dawn) or Gruvbox Material (Hard or Soft) now gives that option's
  colors, and Moon gets its own terminal palette. Before, every option came out
  as the theme's default. Once you'd chosen one, `mordye dux` also failed for
  every theme, even ones without options.
- **Help, and exit codes that say what went wrong.** `mordye --help` and
  `mordye dux --help` print usage. A mistake in the command, such as an
  unknown flag or a missing `--theme`, now exits 2 instead of 1, so a script
  can tell its own mistake from a bad theme, which still exits 1.
- **Bad input fails instead of slipping through.** Text after the JSON on
  stdin, an extra argument to `mordye dux`, or a theme color that isn't a
  string is now an error. Before, Mordye ignored it, or generated the color
  without saying so. Choosing a flavor with no dark colors, such as
  Catppuccin Latte, now says so. Error messages quote any text from the
  theme, so a theme can't send escape codes to your terminal.
- **Status colors match the terminal palette.** On a theme with no `error`,
  `warning`, or `info` color, those come from the terminal red, yellow, and
  blue. They now use the theme's terminal palette when it has one, matching
  what your terminal shows.

## 0.1.0

Released 2026-11-07.

- **First release.** Mordye completes a sparse theme palette into 15 interface
  roles and 22 terminal fields. A color the theme defines comes through
  unchanged, and a missing one is generated at its usual hue, at the palette's
  own lightness and chroma.
- **Dux Shell themes.** `mordye dux` reads a Dux theme file, plus the settings
  that choose its flavor and accent. Changing the accent moves `primary` and
  `secondary` and leaves the terminal colors alone.
- **Rules you can read.** The README explains how each gap is filled and lists
  the few tuned numbers behind it. Missing surfaces follow the theme's own
  spacing and tint.
- **Strict input.** Mordye rejects a color that isn't `#rrggbb` and a name that
  could inject lines into generated config.
- **Linux packages** for x86_64 and ARM64 (including Asahi): an RPM that
  installs `/usr/bin/mordye`, and a `tar.gz` archive, with checksums and build
  provenance. `mordye --version` reports the release.
