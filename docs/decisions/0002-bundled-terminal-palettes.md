# 2. Bundled Terminal Palettes

Date: 2026-11-15

## Status

Proposed ([#35](https://github.com/robwilkerson/mordye/issues/35)). Builds on [ADR 0001](0001-kitty-ports-for-terminal-colors.md).

## Context

Since ADR 0001, `--terminal-palette FILE` gives a theme its exact terminal colors. Without the file, Mordye matches colors to slots by hue, which won't equal the official palette. Every user of a popular theme has to find and download the same port file.

Shipping official palettes inside Mordye removes that step. It also reverses an earlier position: hardcoding palettes was avoided in #32 because it ties Mordye to specific themes.

Kitty's theme collection would be the obvious source, but it's GPL-3.0. Building it into Mordye's MIT binary would bring the binary under GPL terms.

## Decision

Mordye ships a curated set of official terminal palettes, embedded in the binary with `go:embed`.

- **Only official ports,** from the theme's own authors, under a license that allows bundling in MIT software. Each is copied unchanged and credited in the README. Kitty's GPL collection stays out.
- **Applied by theme name.** When a theme's name matches a bundled palette, its colors pass through as if given with `--terminal-palette`.
- **The flag wins.** `--terminal-palette FILE` overrides a bundled palette, for themes Mordye lacks or ports a user prefers.
- **Visible in `sources`,** so a reader can tell a bundled palette's colors from a file's or a generated guess.
- **Contributions** add one palette file plus a credit line. Starting set: Catppuccin's four flavors, Dracula, and Rosé Pine.

Mordye still never reads or downloads anything it isn't given; the palettes are part of the binary.

## Consequences

- Popular themes come out right with no extra file, and callers like a theme engine get simpler.
- Mordye now carries theme data. When a theme updates its port, Mordye's copy needs an update and a release.
- A new bundled palette changes the output for that theme. That's intended, but the changelog should name it.
- Matching by name needs a rule for case, accents, and spacing, plus aliases where names differ. The rule becomes part of the contract.
