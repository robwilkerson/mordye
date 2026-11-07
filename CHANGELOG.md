# Changelog

What changes in each release, written for humans. Full detail lives in git;
this is the readable digest.

Versions follow [semantic versioning](https://semver.org). Until 1.0, a minor
version may still change the input or output format
([#13](https://github.com/robwilkerson/mordye/issues/13)).

Items under **Unreleased** are merged but not yet released. `just release`
moves them under a version heading when it cuts the release.

## Unreleased

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
- **Linux binaries** for x86_64 and ARM64 (including Asahi), with SHA-256
  checksums and build provenance. `mordye --version` reports the release.
